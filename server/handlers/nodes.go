package handlers

import (
	"log"
	"strconv"

	"zen-admin/models"
	"zen-admin/services"
	"zen-admin/singbox"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// NodeHandler обрабатывает запросы для VPN нод
type NodeHandler struct {
	db          *gorm.DB
	nodeClient  *services.NodeClient
	templateGen *singbox.TemplateGenerator
}

// NewNodeHandler создаёт новый обработчик нод
func NewNodeHandler(db *gorm.DB) *NodeHandler {
	return &NodeHandler{
		db:          db,
		nodeClient:  services.NewNodeClient(),
		templateGen: singbox.NewTemplateGenerator(),
	}
}

// CreateNodeRequest - запрос на создание ноды
type CreateNodeRequest struct {
	Name     string `json:"name" validate:"required"`
	Address  string `json:"address" validate:"required"`
	APIPort  int    `json:"api_port"`
	APIToken string `json:"api_token"`
	Enabled  *bool  `json:"enabled"`
}

// UpdateNodeRequest - запрос на обновление ноды
type UpdateNodeRequest struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	APIPort  int    `json:"api_port"`
	APIToken string `json:"api_token"`
	Enabled  *bool  `json:"enabled"`
}

// List - GET /api/nodes
// Список всех нод
func (h *NodeHandler) List(c *fiber.Ctx) error {
	var nodes []models.Node

	query := h.db.Model(&models.Node{})

	// Фильтрация по статусу
	if enabled := c.Query("enabled"); enabled != "" {
		query = query.Where("enabled = ?", enabled == "true")
	}

	// Поиск по имени или адресу
	if search := c.Query("search"); search != "" {
		query = query.Where("name ILIKE ? OR address ILIKE ?", "%"+search+"%", "%"+search+"%")
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

	// Получаем ноды с количеством инбаундов
	if err := query.Preload("Inbounds").Offset(offset).Limit(limit).Order("created_at DESC").Find(&nodes).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка получения нод",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    nodes,
		"meta": fiber.Map{
			"total": total,
			"page":  page,
			"limit": limit,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// Get - GET /api/nodes/:id
// Получение ноды по ID
func (h *NodeHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный ID ноды",
		})
	}

	var node models.Node
	if err := h.db.Preload("Inbounds").First(&node, id).Error; err != nil {
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

	return c.JSON(fiber.Map{
		"success": true,
		"data":    node,
	})
}

// Create - POST /api/nodes
// Создание новой ноды
func (h *NodeHandler) Create(c *fiber.Ctx) error {
	var req CreateNodeRequest
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
			"error":   "Имя ноды обязательно",
		})
	}

	if req.Address == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Адрес ноды обязателен",
		})
	}

	// Создаём ноду
	node := models.Node{
		Name:     req.Name,
		Address:  req.Address,
		APIPort:  req.APIPort,
		APIToken: req.APIToken,
		Enabled:  true,
	}

	if node.APIPort == 0 {
		node.APIPort = 9090 // Порт по умолчанию
	}

	if req.Enabled != nil {
		node.Enabled = *req.Enabled
	}

	if err := h.db.Create(&node).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка создания ноды",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    node,
	})
}

// Update - PUT /api/nodes/:id
// Обновление ноды
func (h *NodeHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный ID ноды",
		})
	}

	var node models.Node
	if err := h.db.First(&node, id).Error; err != nil {
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

	var req UpdateNodeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный формат запроса",
		})
	}

	// Обновляем поля
	if req.Name != "" {
		node.Name = req.Name
	}
	if req.Address != "" {
		node.Address = req.Address
	}
	if req.APIPort > 0 {
		node.APIPort = req.APIPort
	}
	if req.APIToken != "" {
		node.APIToken = req.APIToken
	}
	if req.Enabled != nil {
		node.Enabled = *req.Enabled
	}

	if err := h.db.Save(&node).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка обновления ноды",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    node,
	})
}

// Delete - DELETE /api/nodes/:id
// Удаление ноды
func (h *NodeHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный ID ноды",
		})
	}

	var node models.Node
	if err := h.db.First(&node, id).Error; err != nil {
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

	// Удаляем ноду (soft delete, каскадно удалит инбаунды)
	if err := h.db.Delete(&node).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка удаления ноды",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Нода успешно удалена",
	})
}

// GetStatus - GET /api/nodes/:id/status
// Проверка статуса ноды (online/offline)
func (h *NodeHandler) GetStatus(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный ID ноды",
		})
	}

	var node models.Node
	if err := h.db.First(&node, id).Error; err != nil {
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

	// Пингуем агент на ноде
	status, err := h.nodeClient.GetStatus(&node)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка проверки статуса",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    status,
	})
}

// Sync - POST /api/nodes/:id/sync
// Синхронизация конфига на ноду
func (h *NodeHandler) Sync(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный ID ноды",
		})
	}

	var node models.Node
	if err := h.db.Preload("Inbounds").First(&node, id).Error; err != nil {
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

	// Собираем пользователей для каждого инбаунда
	usersByInbound := make(map[uint][]models.User)
	for _, inbound := range node.Inbounds {
		var users []models.User
		h.db.Model(&inbound).Association("Users").Find(&users)
		usersByInbound[inbound.ID] = users
	}

	// Генерируем серверный конфиг
	config, err := h.templateGen.GenerateServerConfig(node.Inbounds, usersByInbound)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка генерации конфига",
		})
	}

	// Отправляем конфиг на ноду
	if err := h.nodeClient.PushConfig(&node, config); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка отправки конфига: " + err.Error(),
		})
	}

	// Перезапускаем sing-box асинхронно (чтобы не обрывать HTTP ответ,
	// т.к. запрос может идти через REALITY, и рестарт обрежет соединение)
	nodeCopy := node
	go func() {
		if err := h.nodeClient.RestartSingbox(&nodeCopy); err != nil {
			log.Printf("Sync: ошибка перезапуска sing-box на ноде %s: %v", nodeCopy.Name, err)
		} else {
			log.Printf("Sync: sing-box перезапущен на ноде %s", nodeCopy.Name)
		}
	}()

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Конфиг синхронизирован, sing-box перезапускается",
	})
}

// GetAllStatuses - GET /api/nodes/statuses
// Получение статусов всех нод одним запросом
func (h *NodeHandler) GetAllStatuses(c *fiber.Ctx) error {
	var nodes []models.Node
	if err := h.db.Where("enabled = ?", true).Find(&nodes).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка получения нод",
		})
	}

	type NodeStatusResult struct {
		NodeID      uint   `json:"node_id"`
		NodeName    string `json:"node_name"`
		Online      bool   `json:"online"`
		SingboxUp   bool   `json:"singbox_up"`
		Version     string `json:"version,omitempty"`
	}

	results := make([]NodeStatusResult, len(nodes))

	// Параллельная проверка статусов (в реальности лучше использовать goroutines)
	for i, node := range nodes {
		status, _ := h.nodeClient.GetStatus(&node)
		results[i] = NodeStatusResult{
			NodeID:    node.ID,
			NodeName:  node.Name,
			Online:    status.Online,
			SingboxUp: status.SingboxUp,
			Version:   status.Version,
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    results,
	})
}
