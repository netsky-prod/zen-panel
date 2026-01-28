package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

const (
	defaultConfigPath     = "/etc/sing-box/config.json"
	defaultListenAddr     = ":9090"
	defaultSingboxAPI     = "http://127.0.0.1:10085"
	singboxContainerName  = "singbox"
)

var (
	apiToken     string
	configPath   string
	singboxAPI   string
	dockerClient *client.Client
	configMu     sync.RWMutex
)

// Response types
type HealthResponse struct {
	Status  string `json:"status"`
	Singbox string `json:"singbox"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Message string `json:"message"`
}

type KeyPair struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
	ShortID    string `json:"short_id"`
}

type UserStats struct {
	Name     string `json:"name"`
	Upload   int64  `json:"upload"`
	Download int64  `json:"download"`
}

type StatsResponse struct {
	Users []UserStats `json:"users"`
}

// V2Ray API response types
type V2RayStatsResponse struct {
	Stat []V2RayStat `json:"stat"`
}

type V2RayStat struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

func main() {
	// Load configuration from environment
	apiToken = os.Getenv("API_TOKEN")
	if apiToken == "" {
		log.Fatal("API_TOKEN environment variable is required")
	}

	configPath = os.Getenv("SINGBOX_CONFIG")
	if configPath == "" {
		configPath = defaultConfigPath
	}

	singboxAPI = os.Getenv("SINGBOX_API")
	if singboxAPI == "" {
		singboxAPI = defaultSingboxAPI
	}

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = defaultListenAddr
	}

	// Initialize Docker client
	var err error
	dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Printf("Warning: Failed to create Docker client: %v", err)
		log.Println("Some features may not work properly")
	}

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/health", authMiddleware(handleHealth))
	mux.HandleFunc("/config", authMiddleware(handleConfig))
	mux.HandleFunc("/restart", authMiddleware(handleRestart))
	mux.HandleFunc("/stats", authMiddleware(handleStats))
	mux.HandleFunc("/generate-keys", authMiddleware(handleGenerateKeys))

	// Start server
	log.Printf("Node agent starting on %s", listenAddr)
	log.Printf("Config path: %s", configPath)
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		log.Fatal(err)
	}
}

// authMiddleware validates the X-API-Token header
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-API-Token")
		if token == "" || token != apiToken {
			writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
			return
		}
		next(w, r)
	}
}

// handleHealth returns the health status of the agent and sing-box
func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}

	status := "stopped"
	if isSingboxRunning() {
		status = "running"
	}

	writeJSON(w, http.StatusOK, HealthResponse{
		Status:  "ok",
		Singbox: status,
	})
}

// handleConfig handles GET (read) and POST (write) config operations
func handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getConfig(w, r)
	case http.MethodPost:
		postConfig(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
	}
}

// getConfig returns the current sing-box configuration
func getConfig(w http.ResponseWriter, r *http.Request) {
	configMu.RLock()
	defer configMu.RUnlock()

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "config not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// postConfig writes new configuration and restarts sing-box
func postConfig(w http.ResponseWriter, r *http.Request) {
	configMu.Lock()
	defer configMu.Unlock()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "failed to read request body"})
		return
	}
	defer r.Body.Close()

	// Validate JSON
	var jsonCheck interface{}
	if err := json.Unmarshal(body, &jsonCheck); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid JSON: " + err.Error()})
		return
	}

	// Write config to file
	if err := os.WriteFile(configPath, body, 0644); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to write config: " + err.Error()})
		return
	}

	// Restart sing-box
	if err := restartSingbox(); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "config saved but failed to restart: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Message: "config updated and sing-box restarted"})
}

// handleRestart restarts the sing-box process
func handleRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}

	if err := restartSingbox(); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Message: "sing-box restarted"})
}

// handleStats returns traffic statistics from sing-box API
func handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}

	stats, err := getSingboxStats()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// handleGenerateKeys generates a new REALITY keypair
func handleGenerateKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}

	keyPair, err := generateRealityKeys()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, keyPair)
}

// isSingboxRunning checks if the sing-box container is running
func isSingboxRunning() bool {
	if dockerClient == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	containers, err := dockerClient.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		log.Printf("Failed to list containers: %v", err)
		return false
	}

	for _, c := range containers {
		for _, name := range c.Names {
			if strings.TrimPrefix(name, "/") == singboxContainerName || strings.Contains(name, singboxContainerName) {
				return c.State == "running"
			}
		}
	}

	return false
}

// restartSingbox restarts the sing-box container
func restartSingbox() error {
	if dockerClient == nil {
		return fmt.Errorf("docker client not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Find the sing-box container
	containers, err := dockerClient.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	var containerID string
	for _, c := range containers {
		for _, name := range c.Names {
			if strings.TrimPrefix(name, "/") == singboxContainerName || strings.Contains(name, singboxContainerName) {
				containerID = c.ID
				break
			}
		}
		if containerID != "" {
			break
		}
	}

	if containerID == "" {
		return fmt.Errorf("sing-box container not found")
	}

	// Restart the container
	timeout := 10
	if err := dockerClient.ContainerRestart(ctx, containerID, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("failed to restart container: %w", err)
	}

	log.Println("sing-box container restarted successfully")
	return nil
}

// getSingboxStats fetches traffic statistics from sing-box v2ray API
func getSingboxStats() (*StatsResponse, error) {
	// Query uplink stats
	uplinkResp, err := queryV2RayAPI("/stats/query?pattern=user>>>.*>>>traffic>>>uplink&reset=false")
	if err != nil {
		return nil, fmt.Errorf("failed to query uplink stats: %w", err)
	}

	// Query downlink stats
	downlinkResp, err := queryV2RayAPI("/stats/query?pattern=user>>>.*>>>traffic>>>downlink&reset=false")
	if err != nil {
		return nil, fmt.Errorf("failed to query downlink stats: %w", err)
	}

	// Parse and combine stats
	userStats := make(map[string]*UserStats)

	// Process uplink stats
	for _, stat := range uplinkResp.Stat {
		name := extractUserName(stat.Name)
		if name == "" {
			continue
		}
		if _, exists := userStats[name]; !exists {
			userStats[name] = &UserStats{Name: name}
		}
		userStats[name].Upload = stat.Value
	}

	// Process downlink stats
	for _, stat := range downlinkResp.Stat {
		name := extractUserName(stat.Name)
		if name == "" {
			continue
		}
		if _, exists := userStats[name]; !exists {
			userStats[name] = &UserStats{Name: name}
		}
		userStats[name].Download = stat.Value
	}

	// Convert to slice
	result := &StatsResponse{
		Users: make([]UserStats, 0, len(userStats)),
	}
	for _, stats := range userStats {
		result.Users = append(result.Users, *stats)
	}

	return result, nil
}

// queryV2RayAPI queries the sing-box v2ray experimental API
func queryV2RayAPI(path string) (*V2RayStatsResponse, error) {
	url := singboxAPI + path

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result V2RayStatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// extractUserName extracts user name from stat name (format: user>>>username>>>traffic>>>uplink/downlink)
func extractUserName(statName string) string {
	parts := strings.Split(statName, ">>>")
	if len(parts) >= 2 && parts[0] == "user" {
		return parts[1]
	}
	return ""
}

// generateRealityKeys generates a new REALITY keypair using sing-box
func generateRealityKeys() (*KeyPair, error) {
	// Try using sing-box command directly first
	cmd := exec.Command("sing-box", "generate", "reality-keypair")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Try via docker if direct command fails
		cmd = exec.Command("docker", "exec", singboxContainerName, "sing-box", "generate", "reality-keypair")
		output, err = cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to generate keys: %w, output: %s", err, string(output))
		}
	}

	// Parse output (format: PrivateKey: xxx\nPublicKey: xxx)
	lines := strings.Split(string(output), "\n")
	keyPair := &KeyPair{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "PrivateKey:") {
			keyPair.PrivateKey = strings.TrimSpace(strings.TrimPrefix(line, "PrivateKey:"))
		} else if strings.HasPrefix(line, "PublicKey:") {
			keyPair.PublicKey = strings.TrimSpace(strings.TrimPrefix(line, "PublicKey:"))
		}
	}

	if keyPair.PrivateKey == "" || keyPair.PublicKey == "" {
		return nil, fmt.Errorf("failed to parse key output: %s", string(output))
	}

	// Generate short_id
	shortIDCmd := exec.Command("sing-box", "generate", "rand", "--hex", "8")
	shortIDOutput, err := shortIDCmd.CombinedOutput()
	if err != nil {
		shortIDCmd = exec.Command("docker", "exec", singboxContainerName, "sing-box", "generate", "rand", "--hex", "8")
		shortIDOutput, err = shortIDCmd.CombinedOutput()
		if err != nil {
			// Generate a simple hex short_id as fallback
			keyPair.ShortID = generateHexShortID()
		} else {
			keyPair.ShortID = strings.TrimSpace(string(shortIDOutput))
		}
	} else {
		keyPair.ShortID = strings.TrimSpace(string(shortIDOutput))
	}

	return keyPair, nil
}

// generateHexShortID generates a random 8-character hex string
func generateHexShortID() string {
	b := make([]byte, 4)
	f, err := os.Open("/dev/urandom")
	if err != nil {
		return "deadbeef"
	}
	defer f.Close()
	f.Read(b)
	return fmt.Sprintf("%02x%02x%02x%02x", b[0], b[1], b[2], b[3])
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(data); err != nil {
		log.Printf("Failed to encode JSON: %v", err)
		return
	}
	w.Write(buf.Bytes())
}
