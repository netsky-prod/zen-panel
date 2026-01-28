package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"zen-admin/models"
)

// NodeClient - HTTP клиент для связи с агентами на нодах
type NodeClient struct {
	httpClient *http.Client
}

// NewNodeClient создаёт новый клиент для работы с нодами
func NewNodeClient() *NodeClient {
	return &NodeClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// NodeStatus - статус ноды
type NodeStatus struct {
	Online      bool      `json:"online"`
	SingboxUp   bool      `json:"singbox_up"`
	Version     string    `json:"version"`
	Uptime      int64     `json:"uptime"`
	LastChecked time.Time `json:"last_checked"`
}

// RealityKeys - REALITY keypair
type RealityKeys struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
	ShortID    string `json:"short_id"`
}

// UserTraffic - трафик пользователя с ноды
type UserTraffic struct {
	UUID     string `json:"uuid"`
	Upload   int64  `json:"upload"`
	Download int64  `json:"download"`
}

// NodeStats - статистика с ноды
type NodeStats struct {
	Users []UserTraffic `json:"users"`
}

// getNodeURL формирует URL для запроса к ноде
func (c *NodeClient) getNodeURL(node *models.Node, path string) string {
	return fmt.Sprintf("http://%s:%d%s", node.Address, node.APIPort, path)
}

// doRequest выполняет HTTP запрос к ноде с авторизацией
func (c *NodeClient) doRequest(method, url string, body interface{}, token string) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("ошибка сериализации body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-API-Token", token)
	}

	return c.httpClient.Do(req)
}

// GetStatus проверяет статус ноды (health check)
func (c *NodeClient) GetStatus(node *models.Node) (*NodeStatus, error) {
	url := c.getNodeURL(node, "/health")
	resp, err := c.doRequest("GET", url, nil, node.APIToken)
	if err != nil {
		// Нода недоступна
		return &NodeStatus{
			Online:      false,
			LastChecked: time.Now(),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &NodeStatus{
			Online:      false,
			LastChecked: time.Now(),
		}, nil
	}

	var status NodeStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return &NodeStatus{
			Online:      true,
			LastChecked: time.Now(),
		}, nil
	}

	status.Online = true
	status.LastChecked = time.Now()
	return &status, nil
}

// PushConfig отправляет конфигурацию sing-box на ноду
func (c *NodeClient) PushConfig(node *models.Node, config interface{}) error {
	url := c.getNodeURL(node, "/config")
	resp, err := c.doRequest("POST", url, config, node.APIToken)
	if err != nil {
		return fmt.Errorf("ошибка отправки конфига: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ошибка применения конфига: %s", string(body))
	}

	return nil
}

// RestartSingbox перезапускает sing-box на ноде
func (c *NodeClient) RestartSingbox(node *models.Node) error {
	url := c.getNodeURL(node, "/restart")
	resp, err := c.doRequest("POST", url, nil, node.APIToken)
	if err != nil {
		return fmt.Errorf("ошибка перезапуска sing-box: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ошибка перезапуска: %s", string(body))
	}

	return nil
}

// GetStats получает статистику трафика с ноды
func (c *NodeClient) GetStats(node *models.Node) (*NodeStats, error) {
	url := c.getNodeURL(node, "/stats")
	resp, err := c.doRequest("GET", url, nil, node.APIToken)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения статистики: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ошибка статистики: %s", string(body))
	}

	var stats NodeStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("ошибка парсинга статистики: %w", err)
	}

	return &stats, nil
}

// GenerateKeys генерирует REALITY keypair на ноде
func (c *NodeClient) GenerateKeys(node *models.Node) (*RealityKeys, error) {
	url := c.getNodeURL(node, "/generate-keys")
	resp, err := c.doRequest("POST", url, nil, node.APIToken)
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации ключей: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ошибка генерации: %s", string(body))
	}

	var keys RealityKeys
	if err := json.NewDecoder(resp.Body).Decode(&keys); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ключей: %w", err)
	}

	return &keys, nil
}

// GetConfig получает текущий конфиг sing-box с ноды
func (c *NodeClient) GetConfig(node *models.Node) (map[string]interface{}, error) {
	url := c.getNodeURL(node, "/config")
	resp, err := c.doRequest("GET", url, nil, node.APIToken)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения конфига: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ошибка: %s", string(body))
	}

	var config map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("ошибка парсинга конфига: %w", err)
	}

	return config, nil
}
