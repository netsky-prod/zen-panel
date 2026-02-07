package singbox

import (
	"encoding/json"
	"fmt"

	"zen-admin/models"
)

// ServerConfig - конфигурация sing-box сервера
type ServerConfig struct {
	Log       LogConfig        `json:"log"`
	Inbounds  []interface{}    `json:"inbounds"`
	Outbounds []OutboundConfig `json:"outbounds"`
	Route     *RouteConfig     `json:"route,omitempty"`
}

// RouteConfig - настройки маршрутизации
type RouteConfig struct {
	Final string `json:"final"`
}

// LogConfig - настройки логирования
type LogConfig struct {
	Level     string `json:"level"`
	Timestamp bool   `json:"timestamp"`
}

// OutboundConfig - базовый outbound
type OutboundConfig struct {
	Type string `json:"type"`
	Tag  string `json:"tag"`
}

// VLESSUser - пользователь VLESS
type VLESSUser struct {
	UUID string `json:"uuid"`
	Flow string `json:"flow,omitempty"`
}

// Hysteria2User - пользователь Hysteria2
type Hysteria2User struct {
	Password string `json:"password"`
}

// VLESSRealityInbound - VLESS + REALITY inbound конфиг
type VLESSRealityInbound struct {
	Type   string      `json:"type"`
	Tag    string      `json:"tag"`
	Listen string      `json:"listen"`
	Port   int         `json:"listen_port"`
	Users  []VLESSUser `json:"users"`
	TLS    RealityTLS  `json:"tls"`
}

// RealityTLS - TLS настройки для REALITY
type RealityTLS struct {
	Enabled     bool           `json:"enabled"`
	ServerName  string         `json:"server_name"`
	Reality     RealityConfig  `json:"reality"`
}

// RealityConfig - REALITY специфичные настройки
type RealityConfig struct {
	Enabled    bool           `json:"enabled"`
	Handshake  HandshakeConfig `json:"handshake"`
	PrivateKey string         `json:"private_key"`
	ShortID    []string       `json:"short_id"`
}

// HandshakeConfig - настройки handshake для REALITY fallback
type HandshakeConfig struct {
	Server     string `json:"server"`
	ServerPort int    `json:"server_port"`
}

// VLESSWSInbound - VLESS + WS + TLS inbound конфиг
type VLESSWSInbound struct {
	Type      string       `json:"type"`
	Tag       string       `json:"tag"`
	Listen    string       `json:"listen"`
	Port      int          `json:"listen_port"`
	Users     []VLESSUser  `json:"users"`
	TLS       *StandardTLS `json:"tls,omitempty"`
	Transport WSTransport  `json:"transport"`
}

// StandardTLS - стандартные TLS настройки
type StandardTLS struct {
	Enabled    bool   `json:"enabled"`
	ServerName string `json:"server_name"`
	Certificate string `json:"certificate_path,omitempty"`
	Key         string `json:"key_path,omitempty"`
}

// WSTransport - WebSocket транспорт
type WSTransport struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

// Hysteria2Inbound - Hysteria2 inbound конфиг
type Hysteria2Inbound struct {
	Type    string          `json:"type"`
	Tag     string          `json:"tag"`
	Listen  string          `json:"listen"`
	Port    int             `json:"listen_port"`
	UpMbps  int             `json:"up_mbps"`
	DownMbps int            `json:"down_mbps"`
	Users   []Hysteria2User `json:"users"`
	TLS     StandardTLS     `json:"tls"`
}

// TemplateGenerator генерирует серверные конфиги sing-box
type TemplateGenerator struct{}

// NewTemplateGenerator создаёт новый генератор шаблонов
func NewTemplateGenerator() *TemplateGenerator {
	return &TemplateGenerator{}
}

// GenerateVLESSRealityInbound генерирует VLESS+REALITY inbound
func (g *TemplateGenerator) GenerateVLESSRealityInbound(inbound *models.Inbound, users []models.User) *VLESSRealityInbound {
	vlessUsers := make([]VLESSUser, len(users))
	for i, user := range users {
		vlessUsers[i] = VLESSUser{
			UUID: user.UUID.String(),
			Flow: "xtls-rprx-vision",
		}
	}

	// REALITY handshake: fallback сервер для non-VPN TLS клиентов
	// Если задан FallbackAddr - используем его (напр. 127.0.0.1 для локального nginx)
	// Иначе используем SNI домен напрямую
	handshakeServer := inbound.SNI
	handshakePort := 443

	if inbound.FallbackAddr != "" {
		handshakeServer = inbound.FallbackAddr
	}
	if inbound.FallbackPort != 0 {
		handshakePort = inbound.FallbackPort
	}

	return &VLESSRealityInbound{
		Type:   "vless",
		Tag:    fmt.Sprintf("vless-reality-%d", inbound.ID),
		Listen: "::",
		Port:   inbound.ListenPort,
		Users:  vlessUsers,
		TLS: RealityTLS{
			Enabled:    true,
			ServerName: inbound.SNI,
			Reality: RealityConfig{
				Enabled: true,
				Handshake: HandshakeConfig{
					Server:     handshakeServer,
					ServerPort: handshakePort,
				},
				PrivateKey: inbound.PrivateKey,
				ShortID:    []string{inbound.ShortID},
			},
		},
	}
}

// GenerateVLESSWSInbound генерирует VLESS+WS+TLS inbound
// Если CertPath и KeyPath пустые - TLS отключается (nginx терминирует)
func (g *TemplateGenerator) GenerateVLESSWSInbound(inbound *models.Inbound, users []models.User) *VLESSWSInbound {
	vlessUsers := make([]VLESSUser, len(users))
	for i, user := range users {
		vlessUsers[i] = VLESSUser{
			UUID: user.UUID.String(),
		}
	}

	wsPath := inbound.WSPath
	if wsPath == "" {
		wsPath = "/ws"
	}

	result := &VLESSWSInbound{
		Type:   "vless",
		Tag:    fmt.Sprintf("vless-ws-%d", inbound.ID),
		Listen: "::",
		Port:   inbound.ListenPort,
		Users:  vlessUsers,
		Transport: WSTransport{
			Type: "ws",
			Path: wsPath,
		},
	}

	// Если есть сертификаты - включаем TLS на sing-box
	// Если нет - значит TLS терминируется на nginx/reverse proxy
	if inbound.CertPath != "" && inbound.KeyPath != "" {
		result.TLS = &StandardTLS{
			Enabled:     true,
			ServerName:  inbound.SNI,
			Certificate: inbound.CertPath,
			Key:         inbound.KeyPath,
		}
	}

	return result
}

// GenerateHysteria2Inbound генерирует Hysteria2 inbound
func (g *TemplateGenerator) GenerateHysteria2Inbound(inbound *models.Inbound, users []models.User) *Hysteria2Inbound {
	hy2Users := make([]Hysteria2User, len(users))
	for i, user := range users {
		hy2Users[i] = Hysteria2User{
			Password: user.UUID.String(),
		}
	}

	upMbps := inbound.UpMbps
	if upMbps == 0 {
		upMbps = 100
	}
	downMbps := inbound.DownMbps
	if downMbps == 0 {
		downMbps = 100
	}

	return &Hysteria2Inbound{
		Type:     "hysteria2",
		Tag:      fmt.Sprintf("hysteria2-%d", inbound.ID),
		Listen:   "::",
		Port:     inbound.ListenPort,
		UpMbps:   upMbps,
		DownMbps: downMbps,
		Users:    hy2Users,
		TLS: StandardTLS{
			Enabled:     true,
			ServerName:  inbound.SNI,
			Certificate: inbound.CertPath,
			Key:         inbound.KeyPath,
		},
	}
}

// GenerateServerConfig генерирует полный серверный конфиг для ноды
func (g *TemplateGenerator) GenerateServerConfig(inbounds []models.Inbound, usersByInbound map[uint][]models.User) (*ServerConfig, error) {
	config := &ServerConfig{
		Log: LogConfig{
			Level:     "info",
			Timestamp: true,
		},
		Outbounds: []OutboundConfig{
			{Type: "direct", Tag: "direct"},
			{Type: "block", Tag: "block"},
		},
		Route: &RouteConfig{
			Final: "direct",
		},
	}

	for _, inbound := range inbounds {
		if !inbound.Enabled {
			continue
		}

		users := usersByInbound[inbound.ID]
		if len(users) == 0 {
			continue
		}

		var inboundConfig interface{}
		switch inbound.Protocol {
		case models.ProtocolReality:
			inboundConfig = g.GenerateVLESSRealityInbound(&inbound, users)
		case models.ProtocolWSTLS:
			inboundConfig = g.GenerateVLESSWSInbound(&inbound, users)
		case models.ProtocolHysteria2:
			inboundConfig = g.GenerateHysteria2Inbound(&inbound, users)
		default:
			continue
		}

		config.Inbounds = append(config.Inbounds, inboundConfig)
	}

	return config, nil
}

// SerializeConfig сериализует конфиг в JSON строку
func (g *TemplateGenerator) SerializeConfig(config *ServerConfig) (string, error) {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("ошибка сериализации конфига: %w", err)
	}
	return string(data), nil
}

// GenerateInboundJSON генерирует JSON для одного inbound
func (g *TemplateGenerator) GenerateInboundJSON(inbound *models.Inbound, users []models.User) (string, error) {
	var inboundConfig interface{}

	switch inbound.Protocol {
	case models.ProtocolReality:
		inboundConfig = g.GenerateVLESSRealityInbound(inbound, users)
	case models.ProtocolWSTLS:
		inboundConfig = g.GenerateVLESSWSInbound(inbound, users)
	case models.ProtocolHysteria2:
		inboundConfig = g.GenerateHysteria2Inbound(inbound, users)
	default:
		return "", fmt.Errorf("неподдерживаемый протокол: %s", inbound.Protocol)
	}

	data, err := json.MarshalIndent(inboundConfig, "", "  ")
	if err != nil {
		return "", fmt.Errorf("ошибка сериализации: %w", err)
	}
	return string(data), nil
}
