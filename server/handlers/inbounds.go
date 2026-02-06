package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"

	"zen-admin/models"
	"zen-admin/services"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// InboundHandler обрабатывает запросы для инбаундов
type InboundHandler struct {
	db         *gorm.DB
	nodeClient *services.NodeClient
}

// NewInboundHandler создаёт новый обработчик инбаундов
func NewInboundHandler(db *gorm.DB) *InboundHandler {
	return &InboundHandler{
		db:         db,
		nodeClient: services.NewNodeClient(),
	}
}

// CreateInboundRequest - запрос на создание инбаунда
type CreateInboundRequest struct {
	Name         string          `json:"name" validate:"required"`
	Protocol     models.Protocol `json:"protocol" validate:"required"`
	ListenPort   int             `json:"listen_port"`
	SNI          string          `json:"sni"`
	FallbackAddr string          `json:"fallback_addr"`
	FallbackPort int             `json:"fallback_port"`
	PrivateKey   string          `json:"private_key"`
	PublicKey    string          `json:"public_key"`
	ShortID      string          `json:"short_id"`
	UpMbps       int             `json:"up_mbps"`
	DownMbps     int             `json:"down_mbps"`
	WSPath       string          `json:"ws_path"`
	CertPath     string          `json:"cert_path"`
	KeyPath      string          `json:"key_path"`
	Fingerprint  string          `json:"fingerprint"`
	Enabled      *bool           `json:"enabled"`
}

// UpdateInboundRequest - запрос на обновление инбаунда
type UpdateInboundRequest struct {
	Name         string          `json:"name"`
	Protocol     models.Protocol `json:"protocol"`
	ListenPort   int             `json:"listen_port"`
	SNI          string          `json:"sni"`
	FallbackAddr string          `json:"fallback_addr"`
	FallbackPort int             `json:"fallback_port"`
	PrivateKey   string          `json:"private_key"`
	PublicKey    string          `json:"public_key"`
	ShortID      string          `json:"short_id"`
	UpMbps       int             `json:"up_mbps"`
	DownMbps     int             `json:"down_mbps"`
	WSPath       string          `json:"ws_path"`
	CertPath     string          `json:"cert_path"`
	KeyPath      string          `json:"key_path"`
	Fingerprint  string          `json:"fingerprint"`
	Enabled      *bool           `json:"enabled"`
}

// ListByNode - GET /api/nodes/:id/inbounds
// Список инбаундов для конкретной ноды
func (h *InboundHandler) ListByNode(c *fiber.Ctx) error {
	nodeID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный ID ноды",
		})
	}

	// Проверяем существование ноды
	var node models.Node
	if err := h.db.First(&node, nodeID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"error":   "Нода не найдена",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка получения ноды",
		})
	}

	var inbounds []models.Inbound
	if err := h.db.Where("node_id = ?", nodeID).Find(&inbounds).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка получения инбаундов",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    inbounds,
	})
}

// Get - GET /api/inbounds/:id
// Получение инбаунда по ID
func (h *InboundHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный ID инбаунда",
		})
	}

	var inbound models.Inbound
	if err := h.db.Preload("Node").Preload("Users").First(&inbound, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"error":   "Инбаунд не найден",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка получения инбаунда",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    inbound,
	})
}

// Create - POST /api/nodes/:id/inbounds
// Создание нового инбаунда для ноды
func (h *InboundHandler) Create(c *fiber.Ctx) error {
	nodeID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный ID ноды",
		})
	}

	// Проверяем существование ноды
	var node models.Node
	if err := h.db.First(&node, nodeID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"error":   "Нода не найдена",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка получения ноды",
		})
	}

	var req CreateInboundRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный формат запроса",
		})
	}

	// Валидация
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Имя инбаунда обязательно",
		})
	}

	if req.Protocol == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Протокол обязателен",
		})
	}

	// Проверка валидности протокола
	switch req.Protocol {
	case models.ProtocolReality, models.ProtocolWSTLS, models.ProtocolHysteria2:
		// OK
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неподдерживаемый протокол. Используйте: reality, ws-tls, hysteria2",
		})
	}

	// Создаём инбаунд
	inbound := models.Inbound{
		NodeID:       uint(nodeID),
		Name:         req.Name,
		Protocol:     req.Protocol,
		ListenPort:   req.ListenPort,
		SNI:          req.SNI,
		FallbackAddr: req.FallbackAddr,
		FallbackPort: req.FallbackPort,
		PrivateKey:   req.PrivateKey,
		PublicKey:    req.PublicKey,
		ShortID:      req.ShortID,
		UpMbps:       req.UpMbps,
		DownMbps:     req.DownMbps,
		WSPath:       req.WSPath,
		CertPath:     req.CertPath,
		KeyPath:      req.KeyPath,
		Fingerprint:  req.Fingerprint,
		Enabled:      true,
	}

	// Значения по умолчанию
	if inbound.ListenPort == 0 {
		inbound.ListenPort = 443
	}
	if inbound.FallbackAddr == "" {
		inbound.FallbackAddr = "127.0.0.1"
	}
	if inbound.FallbackPort == 0 {
		inbound.FallbackPort = 8443
	}
	if inbound.Fingerprint == "" {
		inbound.Fingerprint = "chrome"
	}
	if inbound.UpMbps == 0 {
		inbound.UpMbps = 100
	}
	if inbound.DownMbps == 0 {
		inbound.DownMbps = 100
	}
	if inbound.WSPath == "" {
		inbound.WSPath = "/ws"
	}
	// Дефолтные пути для TLS сертификатов (Let's Encrypt стандартные пути)
	if inbound.CertPath == "" && (inbound.Protocol == models.ProtocolWSTLS || inbound.Protocol == models.ProtocolHysteria2) {
		inbound.CertPath = "/etc/ssl/certs/cert.pem"
	}
	if inbound.KeyPath == "" && (inbound.Protocol == models.ProtocolWSTLS || inbound.Protocol == models.ProtocolHysteria2) {
		inbound.KeyPath = "/etc/ssl/private/key.pem"
	}

	if req.Enabled != nil {
		inbound.Enabled = *req.Enabled
	}

	if err := h.db.Create(&inbound).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка создания инбаунда",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    inbound,
	})
}

// Update - PUT /api/inbounds/:id
// Обновление инбаунда
func (h *InboundHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный ID инбаунда",
		})
	}

	var inbound models.Inbound
	if err := h.db.First(&inbound, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"error":   "Инбаунд не найден",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка получения инбаунда",
		})
	}

	var req UpdateInboundRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный формат запроса",
		})
	}

	// Обновляем поля
	if req.Name != "" {
		inbound.Name = req.Name
	}
	if req.Protocol != "" {
		// Проверка валидности протокола
		switch req.Protocol {
		case models.ProtocolReality, models.ProtocolWSTLS, models.ProtocolHysteria2:
			inbound.Protocol = req.Protocol
		default:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   "Неподдерживаемый протокол",
			})
		}
	}
	if req.ListenPort > 0 {
		inbound.ListenPort = req.ListenPort
	}
	if req.SNI != "" {
		inbound.SNI = req.SNI
	}
	if req.FallbackAddr != "" {
		inbound.FallbackAddr = req.FallbackAddr
	}
	if req.FallbackPort > 0 {
		inbound.FallbackPort = req.FallbackPort
	}
	if req.PrivateKey != "" {
		inbound.PrivateKey = req.PrivateKey
	}
	if req.PublicKey != "" {
		inbound.PublicKey = req.PublicKey
	}
	if req.ShortID != "" {
		inbound.ShortID = req.ShortID
	}
	if req.UpMbps > 0 {
		inbound.UpMbps = req.UpMbps
	}
	if req.DownMbps > 0 {
		inbound.DownMbps = req.DownMbps
	}
	if req.WSPath != "" {
		inbound.WSPath = req.WSPath
	}
	if req.CertPath != "" {
		inbound.CertPath = req.CertPath
	}
	if req.KeyPath != "" {
		inbound.KeyPath = req.KeyPath
	}
	if req.Fingerprint != "" {
		inbound.Fingerprint = req.Fingerprint
	}
	if req.Enabled != nil {
		inbound.Enabled = *req.Enabled
	}

	if err := h.db.Save(&inbound).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка обновления инбаунда",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    inbound,
	})
}

// Delete - DELETE /api/inbounds/:id
// Удаление инбаунда
func (h *InboundHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный ID инбаунда",
		})
	}

	var inbound models.Inbound
	if err := h.db.First(&inbound, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"error":   "Инбаунд не найден",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка получения инбаунда",
		})
	}

	// Удаляем связи с пользователями
	h.db.Model(&inbound).Association("Users").Clear()

	// Удаляем инбаунд (soft delete)
	if err := h.db.Delete(&inbound).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка удаления инбаунда",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Инбаунд успешно удалён",
	})
}

// GenerateKeys - POST /api/inbounds/:id/generate-keys
// Генерация REALITY keypair
func (h *InboundHandler) GenerateKeys(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный ID инбаунда",
		})
	}

	var inbound models.Inbound
	if err := h.db.Preload("Node").First(&inbound, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"error":   "Инбаунд не найден",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка получения инбаунда",
		})
	}

	// Проверяем, что это REALITY протокол
	if inbound.Protocol != models.ProtocolReality {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Генерация ключей доступна только для REALITY протокола",
		})
	}

	// Пытаемся сгенерировать ключи через агент на ноде
	keys, err := h.nodeClient.GenerateKeys(&inbound.Node)
	if err != nil {
		// Если агент недоступен, генерируем short_id локально
		// Примечание: для полноценной генерации REALITY ключей нужен sing-box
		shortID, _ := generateShortID()
		return c.JSON(fiber.Map{
			"success": true,
			"data": fiber.Map{
				"short_id": shortID,
			},
			"message": "Агент недоступен. Сгенерирован только short_id. Для генерации REALITY ключей используйте sing-box на ноде.",
		})
	}

	// Обновляем инбаунд с новыми ключами
	inbound.PrivateKey = keys.PrivateKey
	inbound.PublicKey = keys.PublicKey
	inbound.ShortID = keys.ShortID

	if err := h.db.Save(&inbound).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка сохранения ключей",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"private_key": keys.PrivateKey,
			"public_key":  keys.PublicKey,
			"short_id":    keys.ShortID,
		},
		"message": "Ключи успешно сгенерированы и сохранены",
	})
}

// generateShortID генерирует случайный short_id для REALITY
func generateShortID() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:8], nil
}

// List - GET /api/inbounds
// Список всех инбаундов
func (h *InboundHandler) List(c *fiber.Ctx) error {
	var inbounds []models.Inbound

	query := h.db.Model(&models.Inbound{}).Preload("Node")

	// Фильтрация по протоколу
	if protocol := c.Query("protocol"); protocol != "" {
		query = query.Where("protocol = ?", protocol)
	}

	// Фильтрация по статусу
	if enabled := c.Query("enabled"); enabled != "" {
		query = query.Where("enabled = ?", enabled == "true")
	}

	// Пагинация
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	// Получаем общее количество
	var total int64
	query.Count(&total)

	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&inbounds).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка получения инбаундов",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    inbounds,
		"meta": fiber.Map{
			"total": total,
			"page":  page,
			"limit": limit,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}
