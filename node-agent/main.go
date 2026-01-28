package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

var (
	configPath = getEnv("CONFIG_PATH", "/etc/sing-box/config.json")
	apiToken   = getEnv("API_TOKEN", "change-me")
	listenAddr = getEnv("LISTEN_ADDR", ":8880")
	startTime  = time.Now()
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Middleware для проверки токена
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-API-Token")
		if token != apiToken {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// Health check
func healthHandler(w http.ResponseWriter, r *http.Request) {
	// Проверяем что sing-box конфиг существует
	singboxUp := false
	if _, err := os.Stat(configPath); err == nil {
		singboxUp = true
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"online":     true,
		"singbox_up": singboxUp,
		"version":    "1.0.0",
		"uptime":     int64(time.Since(startTime).Seconds()),
	})
}

// Получить/обновить конфиг
func configHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		data, err := os.ReadFile(configPath)
		if err != nil {
			http.Error(w, "Config not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)

	case "POST":
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}

		// Валидируем что это валидный JSON
		var config map[string]interface{}
		if err := json.Unmarshal(body, &config); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Форматируем красиво
		formatted, _ := json.MarshalIndent(config, "", "  ")

		// Записываем конфиг
		if err := os.WriteFile(configPath, formatted, 0644); err != nil {
			http.Error(w, "Failed to write config", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Перезапуск sing-box (отправляем SIGHUP для reload)
func restartHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Пытаемся перезапустить через docker
	cmd := exec.Command("docker", "restart", "zen-singbox")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to restart sing-box: %v, output: %s", err, output)
		http.Error(w, fmt.Sprintf("Failed to restart: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Генерация REALITY ключей
func generateKeysHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Используем sing-box для генерации ключей
	cmd := exec.Command("sing-box", "generate", "reality-keypair")
	output, err := cmd.Output()
	if err != nil {
		http.Error(w, "Failed to generate keys", http.StatusInternalServerError)
		return
	}

	// Парсим вывод
	var privateKey, publicKey string
	fmt.Sscanf(string(output), "PrivateKey: %s\nPublicKey: %s", &privateKey, &publicKey)

	// Генерируем short_id
	shortIDBytes := make([]byte, 8)
	rand.Read(shortIDBytes)
	shortID := hex.EncodeToString(shortIDBytes)

	json.NewEncoder(w).Encode(map[string]string{
		"private_key": privateKey,
		"public_key":  publicKey,
		"short_id":    shortID,
	})
}

// Статистика (пока заглушка)
func statsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: получать реальную статистику из sing-box через clash API
	json.NewEncoder(w).Encode(map[string]interface{}{
		"users": []map[string]interface{}{},
	})
}

func main() {
	log.Printf("Node Agent starting on %s", listenAddr)
	log.Printf("Config path: %s", configPath)

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/config", authMiddleware(configHandler))
	http.HandleFunc("/restart", authMiddleware(restartHandler))
	http.HandleFunc("/generate-keys", authMiddleware(generateKeysHandler))
	http.HandleFunc("/stats", authMiddleware(statsHandler))

	log.Fatal(http.ListenAndServe(listenAddr, nil))
}
