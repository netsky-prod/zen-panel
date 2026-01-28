package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Admin - администратор панели
type Admin struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Username  string         `gorm:"uniqueIndex;size:255;not null" json:"username"`
	Password  string         `gorm:"size:255;not null" json:"-"` // Хэш пароля, не отдаём в JSON
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// User - пользователь VPN
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"uniqueIndex;size:255;not null" json:"name"`
	UUID      uuid.UUID      `gorm:"type:uuid;uniqueIndex;not null" json:"uuid"`
	Enabled   bool           `gorm:"default:true" json:"enabled"`
	DataLimit int64          `gorm:"default:0" json:"data_limit"`  // Лимит в байтах, 0 = безлимит
	DataUsed  int64          `gorm:"default:0" json:"data_used"`   // Использовано байт
	ExpiresAt *time.Time     `json:"expires_at,omitempty"`         // Срок действия
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Связи
	Inbounds []Inbound `gorm:"many2many:user_inbounds;" json:"inbounds,omitempty"`
}

// BeforeCreate генерирует UUID для нового пользователя
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.UUID == uuid.Nil {
		u.UUID = uuid.New()
	}
	return nil
}

// Node - VPN нода (сервер)
type Node struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:255;not null" json:"name"`
	Address   string         `gorm:"size:255;not null" json:"address"`   // IP или домен сервера
	APIPort   int            `gorm:"default:9090" json:"api_port"`       // Порт агента
	APIToken  string         `gorm:"size:255" json:"api_token,omitempty"` // Токен для связи с агентом
	Enabled   bool           `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Связи
	Inbounds []Inbound `gorm:"foreignKey:NodeID" json:"inbounds,omitempty"`
}

// Protocol - тип протокола инбаунда
type Protocol string

const (
	ProtocolReality   Protocol = "reality"
	ProtocolWSTLS     Protocol = "ws-tls"
	ProtocolHysteria2 Protocol = "hysteria2"
)

// Inbound - точка входа на ноде
type Inbound struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	NodeID     uint           `gorm:"index;not null" json:"node_id"`
	Name       string         `gorm:"size:255;not null" json:"name"`
	Protocol   Protocol       `gorm:"size:50;not null" json:"protocol"` // reality, ws-tls, hysteria2
	ListenPort int            `gorm:"default:443" json:"listen_port"`

	// TLS/REALITY settings
	SNI          string `gorm:"size:255" json:"sni,omitempty"`           // Домен для SNI
	FallbackAddr string `gorm:"size:255;default:'127.0.0.1'" json:"fallback_addr,omitempty"`
	FallbackPort int    `gorm:"default:8443" json:"fallback_port,omitempty"`

	// REALITY keys
	PrivateKey string `gorm:"size:255" json:"private_key,omitempty"`
	PublicKey  string `gorm:"size:255" json:"public_key,omitempty"`
	ShortID    string `gorm:"size:16" json:"short_id,omitempty"`

	// Hysteria2 settings
	UpMbps   int `gorm:"default:100" json:"up_mbps,omitempty"`
	DownMbps int `gorm:"default:100" json:"down_mbps,omitempty"`

	// WS settings
	WSPath string `gorm:"size:255" json:"ws_path,omitempty"`

	// TLS certificate paths (для ws-tls и hysteria2)
	CertPath string `gorm:"size:255" json:"cert_path,omitempty"`
	KeyPath  string `gorm:"size:255" json:"key_path,omitempty"`

	// Fingerprint для uTLS
	Fingerprint string `gorm:"size:50;default:'chrome'" json:"fingerprint"`

	Enabled   bool           `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Связи
	Node  Node   `gorm:"foreignKey:NodeID" json:"node,omitempty"`
	Users []User `gorm:"many2many:user_inbounds;" json:"users,omitempty"`
}

// UserInbound - связь пользователя с инбаундом
type UserInbound struct {
	UserID    uint `gorm:"primaryKey"`
	InboundID uint `gorm:"primaryKey"`
}

// TrafficStats - статистика трафика
type TrafficStats struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"index:idx_traffic_user" json:"user_id"`
	InboundID  uint      `gorm:"index:idx_traffic_inbound" json:"inbound_id"`
	Upload     int64     `gorm:"default:0" json:"upload"`
	Download   int64     `gorm:"default:0" json:"download"`
	RecordedAt time.Time `gorm:"index:idx_traffic_recorded;default:CURRENT_TIMESTAMP" json:"recorded_at"`

	// Связи
	User    User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Inbound Inbound `gorm:"foreignKey:InboundID" json:"inbound,omitempty"`
}

// AutoMigrate выполняет автоматическую миграцию всех моделей
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Admin{},
		&User{},
		&Node{},
		&Inbound{},
		&UserInbound{},
		&TrafficStats{},
	)
}
