package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/skip2/go-qrcode"
	"zen-admin/models"
)

// ConfigGenerator - генератор клиентских конфигов
type ConfigGenerator struct{}

// NewConfigGenerator создаёт новый генератор конфигов
func NewConfigGenerator() *ConfigGenerator {
	return &ConfigGenerator{}
}

// SingboxClientConfig - структура клиентского конфига sing-box
type SingboxClientConfig struct {
	Log       map[string]interface{}   `json:"log"`
	DNS       map[string]interface{}   `json:"dns"`
	Inbounds  []map[string]interface{} `json:"inbounds"`
	Outbounds []map[string]interface{} `json:"outbounds"`
	Route     map[string]interface{}   `json:"route"`
}

// GenerateSingboxConfig генерирует полный sing-box клиентский конфиг
func (g *ConfigGenerator) GenerateSingboxConfig(user *models.User, inbounds []models.Inbound) (*SingboxClientConfig, error) {
	config := &SingboxClientConfig{
		Log: map[string]interface{}{
			"level":     "info",
			"timestamp": true,
		},
		DNS: map[string]interface{}{
			"servers": []map[string]interface{}{
				{
					"tag":     "proxy-dns",
					"address": "8.8.8.8",
					"detour":  "proxy",
				},
				{
					"tag":     "direct-dns",
					"address": "8.8.8.8",
					"detour":  "direct",
				},
			},
			"rules": []map[string]interface{}{
				{
					"outbound": "any",
					"server":   "direct-dns",
				},
			},
			"final": "proxy-dns",
			"strategy": "prefer_ipv4",
		},
		Inbounds: []map[string]interface{}{
			{
				"type":                       "tun",
				"tag":                        "tun-in",
				"interface_name":             "tun0",
				"inet4_address":              "172.19.0.1/30",
				"mtu":                        9000,
				"auto_route":                 true,
				"strict_route":               true,
				"stack":                      "system",
				"sniff":                      true,
				"sniff_override_destination": true,
			},
		},
		Route: map[string]interface{}{
			"auto_detect_interface": true,
			"final":                 "proxy",
			"rules": []map[string]interface{}{
				{
					"protocol": "dns",
					"outbound": "dns-out",
				},
				{
					"geoip":    []string{"private"},
					"outbound": "direct",
				},
				{
					"geosite":  []string{"category-ads-all"},
					"outbound": "block",
				},
			},
		},
	}

	// Генерируем outbounds для каждого инбаунда
	outbounds := []map[string]interface{}{}

	for _, inbound := range inbounds {
		if !inbound.Enabled {
			continue
		}

		outbound, err := g.generateOutbound(user, &inbound)
		if err != nil {
			continue
		}
		outbounds = append(outbounds, outbound)
	}

	// Добавляем селектор если несколько серверов
	if len(outbounds) > 1 {
		serverTags := make([]string, len(outbounds))
		for i, ob := range outbounds {
			serverTags[i] = ob["tag"].(string)
		}
		selector := map[string]interface{}{
			"type":      "selector",
			"tag":       "proxy",
			"outbounds": serverTags,
			"default":   serverTags[0],
		}
		outbounds = append([]map[string]interface{}{selector}, outbounds...)
	} else if len(outbounds) == 1 {
		// Если один сервер, делаем его тегом proxy
		outbounds[0]["tag"] = "proxy"
	}

	// Добавляем стандартные outbounds
	outbounds = append(outbounds,
		map[string]interface{}{
			"type": "direct",
			"tag":  "direct",
		},
		map[string]interface{}{
			"type": "block",
			"tag":  "block",
		},
		map[string]interface{}{
			"type": "dns",
			"tag":  "dns-out",
		},
	)

	config.Outbounds = outbounds

	return config, nil
}

// generateOutbound генерирует outbound для конкретного инбаунда
func (g *ConfigGenerator) generateOutbound(user *models.User, inbound *models.Inbound) (map[string]interface{}, error) {
	tag := fmt.Sprintf("%s-%s", inbound.Node.Name, inbound.Name)

	switch inbound.Protocol {
	case models.ProtocolReality:
		return g.generateVLESSRealityOutbound(user, inbound, tag)
	case models.ProtocolWSTLS:
		return g.generateVLESSWSOutbound(user, inbound, tag)
	case models.ProtocolHysteria2:
		return g.generateHysteria2Outbound(user, inbound, tag)
	default:
		return nil, fmt.Errorf("неподдерживаемый протокол: %s", inbound.Protocol)
	}
}

// generateVLESSRealityOutbound генерирует VLESS+REALITY outbound
func (g *ConfigGenerator) generateVLESSRealityOutbound(user *models.User, inbound *models.Inbound, tag string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"type":        "vless",
		"tag":         tag,
		"server":      inbound.Node.Address,
		"server_port": inbound.ListenPort,
		"uuid":        user.UUID.String(),
		"flow":        "xtls-rprx-vision",
		"tls": map[string]interface{}{
			"enabled":     true,
			"server_name": inbound.SNI,
			"utls": map[string]interface{}{
				"enabled":     true,
				"fingerprint": inbound.Fingerprint,
			},
			"reality": map[string]interface{}{
				"enabled":    true,
				"public_key": inbound.PublicKey,
				"short_id":   inbound.ShortID,
			},
		},
	}, nil
}

// generateVLESSWSOutbound генерирует VLESS+WS+TLS outbound
func (g *ConfigGenerator) generateVLESSWSOutbound(user *models.User, inbound *models.Inbound, tag string) (map[string]interface{}, error) {
	wsPath := inbound.WSPath
	if wsPath == "" {
		wsPath = "/ws"
	}

	return map[string]interface{}{
		"type":        "vless",
		"tag":         tag,
		"server":      inbound.Node.Address,
		"server_port": inbound.ListenPort,
		"uuid":        user.UUID.String(),
		"tls": map[string]interface{}{
			"enabled":     true,
			"server_name": inbound.SNI,
			"utls": map[string]interface{}{
				"enabled":     true,
				"fingerprint": inbound.Fingerprint,
			},
		},
		"transport": map[string]interface{}{
			"type": "ws",
			"path": wsPath,
			"headers": map[string]interface{}{
				"Host": inbound.SNI,
			},
		},
	}, nil
}

// generateHysteria2Outbound генерирует Hysteria2 outbound
func (g *ConfigGenerator) generateHysteria2Outbound(user *models.User, inbound *models.Inbound, tag string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"type":        "hysteria2",
		"tag":         tag,
		"server":      inbound.Node.Address,
		"server_port": inbound.ListenPort,
		"password":    user.UUID.String(),
		"up_mbps":     inbound.UpMbps,
		"down_mbps":   inbound.DownMbps,
		"tls": map[string]interface{}{
			"enabled":     true,
			"server_name": inbound.SNI,
			"insecure":    false,
		},
	}, nil
}

// GenerateShareURL генерирует URL для шаринга (vless://, hysteria2://)
func (g *ConfigGenerator) GenerateShareURL(user *models.User, inbound *models.Inbound) (string, error) {
	switch inbound.Protocol {
	case models.ProtocolReality:
		return g.generateVLESSRealityURL(user, inbound)
	case models.ProtocolWSTLS:
		return g.generateVLESSWSURL(user, inbound)
	case models.ProtocolHysteria2:
		return g.generateHysteria2URL(user, inbound)
	default:
		return "", fmt.Errorf("неподдерживаемый протокол: %s", inbound.Protocol)
	}
}

// generateVLESSRealityURL генерирует vless:// URL для REALITY
func (g *ConfigGenerator) generateVLESSRealityURL(user *models.User, inbound *models.Inbound) (string, error) {
	// Формат: vless://uuid@server:port?params#name
	params := url.Values{}
	params.Set("type", "tcp")
	params.Set("security", "reality")
	params.Set("sni", inbound.SNI)
	params.Set("fp", inbound.Fingerprint)
	params.Set("pbk", inbound.PublicKey)
	params.Set("sid", inbound.ShortID)
	params.Set("flow", "xtls-rprx-vision")

	name := url.QueryEscape(fmt.Sprintf("%s - %s", inbound.Node.Name, inbound.Name))
	shareURL := fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		user.UUID.String(),
		inbound.Node.Address,
		inbound.ListenPort,
		params.Encode(),
		name,
	)

	return shareURL, nil
}

// generateVLESSWSURL генерирует vless:// URL для WS+TLS
func (g *ConfigGenerator) generateVLESSWSURL(user *models.User, inbound *models.Inbound) (string, error) {
	wsPath := inbound.WSPath
	if wsPath == "" {
		wsPath = "/ws"
	}

	params := url.Values{}
	params.Set("type", "ws")
	params.Set("security", "tls")
	params.Set("sni", inbound.SNI)
	params.Set("host", inbound.SNI)
	params.Set("path", wsPath)
	params.Set("fp", inbound.Fingerprint)

	name := url.QueryEscape(fmt.Sprintf("%s - %s", inbound.Node.Name, inbound.Name))
	shareURL := fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		user.UUID.String(),
		inbound.Node.Address,
		inbound.ListenPort,
		params.Encode(),
		name,
	)

	return shareURL, nil
}

// generateHysteria2URL генерирует hysteria2:// URL
func (g *ConfigGenerator) generateHysteria2URL(user *models.User, inbound *models.Inbound) (string, error) {
	// Формат: hysteria2://password@server:port?sni=xxx#name
	params := url.Values{}
	params.Set("sni", inbound.SNI)

	name := url.QueryEscape(fmt.Sprintf("%s - %s", inbound.Node.Name, inbound.Name))
	shareURL := fmt.Sprintf("hysteria2://%s@%s:%d?%s#%s",
		user.UUID.String(),
		inbound.Node.Address,
		inbound.ListenPort,
		params.Encode(),
		name,
	)

	return shareURL, nil
}

// GenerateQRCode генерирует QR-код PNG из URL
func (g *ConfigGenerator) GenerateQRCode(content string) ([]byte, error) {
	qr, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания QR-кода: %w", err)
	}

	// Генерируем PNG
	return qr.PNG(256)
}

// GenerateQRCodeBase64 генерирует QR-код в формате base64
func (g *ConfigGenerator) GenerateQRCodeBase64(content string) (string, error) {
	pngData, err := g.GenerateQRCode(content)
	if err != nil {
		return "", err
	}

	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngData), nil
}

// GenerateAllShareURLs генерирует все URL для пользователя
func (g *ConfigGenerator) GenerateAllShareURLs(user *models.User, inbounds []models.Inbound) ([]string, error) {
	var urls []string

	for _, inbound := range inbounds {
		if !inbound.Enabled {
			continue
		}

		shareURL, err := g.GenerateShareURL(user, &inbound)
		if err != nil {
			continue
		}
		urls = append(urls, shareURL)
	}

	return urls, nil
}

// GenerateSubscription генерирует subscription URL (base64 encoded URLs)
func (g *ConfigGenerator) GenerateSubscription(user *models.User, inbounds []models.Inbound) (string, error) {
	urls, err := g.GenerateAllShareURLs(user, inbounds)
	if err != nil {
		return "", err
	}

	// Объединяем URL через перенос строки и кодируем в base64
	combined := strings.Join(urls, "\n")
	return base64.StdEncoding.EncodeToString([]byte(combined)), nil
}

// SerializeConfig сериализует конфиг в JSON
func (g *ConfigGenerator) SerializeConfig(config *SingboxClientConfig) (string, error) {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("ошибка сериализации конфига: %w", err)
	}
	return string(data), nil
}
