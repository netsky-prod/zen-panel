package handlers

import (
	"zen-admin/middleware"
	"zen-admin/models"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthHandler обрабатывает запросы аутентификации
type AuthHandler struct {
	db *gorm.DB
}

// NewAuthHandler создаёт новый обработчик аутентификации
func NewAuthHandler(db *gorm.DB) *AuthHandler {
	return &AuthHandler{db: db}
}

// LoginRequest - запрос на логин
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse - ответ на логин
type LoginResponse struct {
	Token string       `json:"token"`
	Admin AdminInfo    `json:"admin"`
}

// AdminInfo - информация об администраторе
type AdminInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

// Login - POST /api/auth/login
// Аутентификация администратора
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный формат запроса",
		})
	}

	// Валидация
	if req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Логин и пароль обязательны",
		})
	}

	// Поиск администратора
	var admin models.Admin
	if err := h.db.Where("username = ?", req.Username).First(&admin).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Неверный логин или пароль",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка базы данных",
		})
	}

	// Проверка пароля
	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный логин или пароль",
		})
	}

	// Генерация JWT токена
	token, err := middleware.GenerateToken(admin.ID, admin.Username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка генерации токена",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": LoginResponse{
			Token: token,
			Admin: AdminInfo{
				ID:       admin.ID,
				Username: admin.Username,
			},
		},
	})
}

// Logout - POST /api/auth/logout
// Выход из системы (client-side: просто удалить токен)
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	// JWT токены stateless, поэтому logout на сервере - просто подтверждение
	// Клиент должен удалить токен из localStorage
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Выход выполнен успешно",
	})
}

// Me - GET /api/auth/me
// Информация о текущем администраторе
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	adminID := c.Locals("admin_id").(uint)
	username := c.Locals("username").(string)

	return c.JSON(fiber.Map{
		"success": true,
		"data": AdminInfo{
			ID:       adminID,
			Username: username,
		},
	})
}

// ChangePasswordRequest - запрос на смену пароля
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6"`
}

// ChangePassword - POST /api/auth/change-password
// Смена пароля администратора
func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	adminID := c.Locals("admin_id").(uint)

	var req ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный формат запроса",
		})
	}

	// Валидация
	if req.OldPassword == "" || req.NewPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Старый и новый пароли обязательны",
		})
	}

	if len(req.NewPassword) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Новый пароль должен быть не менее 6 символов",
		})
	}

	// Получаем администратора
	var admin models.Admin
	if err := h.db.First(&admin, adminID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка базы данных",
		})
	}

	// Проверяем старый пароль
	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(req.OldPassword)); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный текущий пароль",
		})
	}

	// Хэшируем новый пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка хэширования пароля",
		})
	}

	// Обновляем пароль
	if err := h.db.Model(&admin).Update("password", string(hashedPassword)).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка обновления пароля",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Пароль успешно изменён",
	})
}
