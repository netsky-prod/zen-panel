package handlers

import (
	"strconv"
	"time"

	"zen-admin/models"
	"zen-admin/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserHandler обрабатывает запросы пользователей VPN
type UserHandler struct {
	db        *gorm.DB
	configGen *services.ConfigGenerator
}

// NewUserHandler создаёт новый обработчик пользователей
func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{
		db:        db,
		configGen: services.NewConfigGenerator(),
	}
}

// CreateUserRequest - запрос на создание пользователя
type CreateUserRequest struct {
	Name       string     `json:"name" validate:"required"`
	Enabled    *bool      `json:"enabled"`
	DataLimit  int64      `json:"data_limit"`
	ExpiresAt  *time.Time `json:"expires_at"`
	InboundIDs []uint     `json:"inbound_ids"`
}

// UpdateUserRequest - запрос на обновление пользователя
type UpdateUserRequest struct {
	Name       string     `json:"name"`
	Enabled    *bool      `json:"enabled"`
	DataLimit  int64      `json:"data_limit"`
	ExpiresAt  *time.Time `json:"expires_at"`
	InboundIDs []uint     `json:"inbound_ids"`
}

// List - GET /api/users
// Список всех пользователей
func (h *UserHandler) List(c *fiber.Ctx) error {
	var users []models.User

	query := h.db.Model(&models.User{})

	// Фильтрация по статусу
	if enabled := c.Query("enabled"); enabled != "" {
		query = query.Where("enabled = ?", enabled == "true")
	}

	// Поиск по имени
	if search := c.Query("search"); search != "" {
		query = query.Where("name ILIKE ?", "%"+search+"%")
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

	// Получаем пользователей с инбаундами
	if err := query.Preload("Inbounds").Offset(offset).Limit(limit).Order("created_at DESC").Find(&users).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка получения пользователей",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    users,
		"meta": fiber.Map{
			"total":  total,
			"page":   page,
			"limit":  limit,
			"pages":  (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// Get - GET /api/users/:id
// Получение пользователя по ID
func (h *UserHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный ID пользователя",
		})
	}

	var user models.User
	if err := h.db.Preload("Inbounds.Node").First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"error":   "Пользователь не найден",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка получения пользователя",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    user,
	})
}

// Create - POST /api/users
// Создание нового пользователя
func (h *UserHandler) Create(c *fiber.Ctx) error {
	var req CreateUserRequest
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
			"error":   "Имя пользователя обязательно",
		})
	}

	// Проверка уникальности имени
	var existing models.User
	if err := h.db.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"success": false,
			"error":   "Пользователь с таким именем уже существует",
		})
	}

	// Создаём пользователя
	user := models.User{
		Name:      req.Name,
		UUID:      uuid.New(),
		Enabled:   true,
		DataLimit: req.DataLimit,
		DataUsed:  0,
		ExpiresAt: req.ExpiresAt,
	}

	if req.Enabled != nil {
		user.Enabled = *req.Enabled
	}

	if err := h.db.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка создания пользователя",
		})
	}

	// Привязываем инбаунды, если указаны
	if len(req.InboundIDs) > 0 {
		var inbounds []models.Inbound
		if err := h.db.Where("id IN ?", req.InboundIDs).Find(&inbounds).Error; err == nil {
			h.db.Model(&user).Association("Inbounds").Replace(inbounds)
		}
	}

	// Загружаем инбаунды для ответа
	h.db.Preload("Inbounds").First(&user, user.ID)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    user,
	})
}

// Update - PUT /api/users/:id
// Обновление пользователя
func (h *UserHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный ID пользователя",
		})
	}

	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"error":   "Пользователь не найден",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка получения пользователя",
		})
	}

	var req UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный формат запроса",
		})
	}

	// Обновляем поля
	if req.Name != "" && req.Name != user.Name {
		// Проверка уникальности нового имени
		var existing models.User
		if err := h.db.Where("name = ? AND id != ?", req.Name, id).First(&existing).Error; err == nil {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"success": false,
				"error":   "Пользователь с таким именем уже существует",
			})
		}
		user.Name = req.Name
	}

	if req.Enabled != nil {
		user.Enabled = *req.Enabled
	}

	if req.DataLimit >= 0 {
		user.DataLimit = req.DataLimit
	}

	if req.ExpiresAt != nil {
		user.ExpiresAt = req.ExpiresAt
	}

	if err := h.db.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка обновления пользователя",
		})
	}

	// Обновляем инбаунды, если указаны
	if req.InboundIDs != nil {
		var inbounds []models.Inbound
		if len(req.InboundIDs) > 0 {
			h.db.Where("id IN ?", req.InboundIDs).Find(&inbounds)
		}
		h.db.Model(&user).Association("Inbounds").Replace(inbounds)
	}

	// Загружаем инбаунды для ответа
	h.db.Preload("Inbounds").First(&user, user.ID)

	return c.JSON(fiber.Map{
		"success": true,
		"data":    user,
	})
}

// Delete - DELETE /api/users/:id
// Удаление пользователя
func (h *UserHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный ID пользователя",
		})
	}

	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"error":   "Пользователь не найден",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка получения пользователя",
		})
	}

	// Удаляем связи с инбаундами
	h.db.Model(&user).Association("Inbounds").Clear()

	// Удаляем пользователя (soft delete)
	if err := h.db.Delete(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка удаления пользователя",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Пользователь успешно удалён",
	})
}

// GetConfig - GET /api/users/:id/config
// Генерация клиентского конфига
// Query params: format=json|url|qr|subscription
func (h *UserHandler) GetConfig(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный ID пользователя",
		})
	}

	var user models.User
	if err := h.db.Preload("Inbounds.Node").First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"error":   "Пользователь не найден",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка получения пользователя",
		})
	}

	if len(user.Inbounds) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "У пользователя нет привязанных инбаундов",
		})
	}

	format := c.Query("format", "json")

	switch format {
	case "json":
		// Полный sing-box JSON конфиг
		config, err := h.configGen.GenerateSingboxConfig(&user, user.Inbounds)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   "Ошибка генерации конфига",
			})
		}
		return c.JSON(fiber.Map{
			"success": true,
			"data":    config,
		})

	case "url":
		// Share URLs для всех инбаундов
		urls, err := h.configGen.GenerateAllShareURLs(&user, user.Inbounds)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   "Ошибка генерации URL",
			})
		}
		return c.JSON(fiber.Map{
			"success": true,
			"data":    urls,
		})

	case "qr":
		// QR-коды для всех инбаундов (base64)
		var qrCodes []map[string]string
		for _, inbound := range user.Inbounds {
			if !inbound.Enabled {
				continue
			}
			shareURL, err := h.configGen.GenerateShareURL(&user, &inbound)
			if err != nil {
				continue
			}
			qrBase64, err := h.configGen.GenerateQRCodeBase64(shareURL)
			if err != nil {
				continue
			}
			qrCodes = append(qrCodes, map[string]string{
				"name": inbound.Name,
				"url":  shareURL,
				"qr":   qrBase64,
			})
		}
		return c.JSON(fiber.Map{
			"success": true,
			"data":    qrCodes,
		})

	case "subscription":
		// Base64-encoded subscription
		subscription, err := h.configGen.GenerateSubscription(&user, user.Inbounds)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   "Ошибка генерации подписки",
			})
		}
		// Возвращаем как plain text для прямого использования
		c.Set("Content-Type", "text/plain")
		return c.SendString(subscription)

	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неподдерживаемый формат. Используйте: json, url, qr, subscription",
		})
	}
}

// ResetUUID - POST /api/users/:id/reset-uuid
// Сброс UUID пользователя
func (h *UserHandler) ResetUUID(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный ID пользователя",
		})
	}

	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"error":   "Пользователь не найден",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка получения пользователя",
		})
	}

	// Генерируем новый UUID
	user.UUID = uuid.New()

	if err := h.db.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка обновления UUID",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"uuid": user.UUID.String(),
		},
		"message": "UUID успешно сброшен. Не забудьте синхронизировать конфиги на нодах.",
	})
}

// ResetTraffic - POST /api/users/:id/reset-traffic
// Сброс счётчика трафика пользователя
func (h *UserHandler) ResetTraffic(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный ID пользователя",
		})
	}

	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"error":   "Пользователь не найден",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка получения пользователя",
		})
	}

	oldUsed := user.DataUsed
	user.DataUsed = 0

	if err := h.db.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка сброса трафика",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"previous_used": oldUsed,
			"current_used":  0,
		},
		"message": "Счётчик трафика успешно сброшен",
	})
}
