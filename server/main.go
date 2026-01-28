package main

import (
	"fmt"
	"log"
	"os"

	"zen-admin/handlers"
	"zen-admin/middleware"
	"zen-admin/models"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Конфигурация из переменных окружения
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "zen")
	dbPassword := getEnv("DB_PASSWORD", "zen")
	dbName := getEnv("DB_NAME", "zen_admin")
	serverPort := getEnv("SERVER_PORT", "8080")

	// Подключение к PostgreSQL
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC",
		dbHost, dbPort, dbUser, dbPassword, dbName,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}

	log.Println("Подключение к базе данных установлено")

	// Автоматическая миграция моделей
	if err := models.AutoMigrate(db); err != nil {
		log.Fatalf("Ошибка миграции базы данных: %v", err)
	}

	log.Println("Миграция базы данных выполнена")

	// Создание администратора по умолчанию, если не существует
	createDefaultAdmin(db)

	// Инициализация Fiber приложения
	app := fiber.New(fiber.Config{
		AppName:      "Zen VPN Admin API",
		ErrorHandler: customErrorHandler,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     getEnv("CORS_ORIGINS", "http://localhost:3000,http://localhost:5173"),
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
		AllowCredentials: true,
	}))

	// Инициализация обработчиков
	authHandler := handlers.NewAuthHandler(db)
	userHandler := handlers.NewUserHandler(db)
	nodeHandler := handlers.NewNodeHandler(db)
	inboundHandler := handlers.NewInboundHandler(db)
	statsHandler := handlers.NewStatsHandler(db)
	dashboardHandler := handlers.NewDashboardHandler(db)
	publicHandler := handlers.NewPublicHandler(db)

	// === Публичные маршруты ===
	api := app.Group("/api")

	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success": true,
			"message": "Zen VPN Admin API is running",
		})
	})

	// Auth routes (публичные)
	auth := api.Group("/auth")
	auth.Post("/login", authHandler.Login)

	// Public pages (для юзеров - без авторизации)
	sub := api.Group("/sub")
	sub.Get("/:uuid", publicHandler.UserConfigPage)      // Красивая HTML страница
	sub.Get("/:uuid/raw", publicHandler.RawSubscription) // Raw для приложений

	// === Защищённые маршруты ===
	protected := api.Group("", middleware.JWTMiddleware())

	// Auth (защищённые)
	protectedAuth := protected.Group("/auth")
	protectedAuth.Post("/logout", authHandler.Logout)
	protectedAuth.Get("/me", authHandler.Me)
	protectedAuth.Post("/change-password", authHandler.ChangePassword)

	// Dashboard
	protected.Get("/dashboard", dashboardHandler.Get)
	protected.Get("/dashboard/quick", dashboardHandler.GetQuickStats)

	// Users
	users := protected.Group("/users")
	users.Get("/", userHandler.List)
	users.Post("/", userHandler.Create)
	users.Get("/:id", userHandler.Get)
	users.Put("/:id", userHandler.Update)
	users.Delete("/:id", userHandler.Delete)
	users.Get("/:id/config", userHandler.GetConfig)
	users.Post("/:id/reset-uuid", userHandler.ResetUUID)
	users.Post("/:id/reset-traffic", userHandler.ResetTraffic)

	// Nodes
	nodes := protected.Group("/nodes")
	nodes.Get("/", nodeHandler.List)
	nodes.Post("/", nodeHandler.Create)
	nodes.Get("/statuses", nodeHandler.GetAllStatuses)
	nodes.Get("/:id", nodeHandler.Get)
	nodes.Put("/:id", nodeHandler.Update)
	nodes.Delete("/:id", nodeHandler.Delete)
	nodes.Get("/:id/status", nodeHandler.GetStatus)
	nodes.Post("/:id/sync", nodeHandler.Sync)
	nodes.Get("/:id/inbounds", inboundHandler.ListByNode)
	nodes.Post("/:id/inbounds", inboundHandler.Create)

	// Inbounds
	inbounds := protected.Group("/inbounds")
	inbounds.Get("/", inboundHandler.List)
	inbounds.Get("/:id", inboundHandler.Get)
	inbounds.Put("/:id", inboundHandler.Update)
	inbounds.Delete("/:id", inboundHandler.Delete)
	inbounds.Post("/:id/generate-keys", inboundHandler.GenerateKeys)

	// Stats
	stats := protected.Group("/stats")
	stats.Get("/", statsHandler.GetOverall)
	stats.Get("/users/:id", statsHandler.GetUserStats)
	stats.Get("/nodes/:id", statsHandler.GetNodeStats)
	stats.Get("/top-users", statsHandler.GetTopUsers)

	// Запуск сервера
	log.Printf("Сервер запущен на порту %s", serverPort)
	if err := app.Listen(":" + serverPort); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}

// getEnv возвращает значение переменной окружения или значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// createDefaultAdmin создаёт администратора по умолчанию, если его нет
func createDefaultAdmin(db *gorm.DB) {
	var count int64
	db.Model(&models.Admin{}).Count(&count)

	if count == 0 {
		// Пароль по умолчанию: admin
		// В продакшене ОБЯЗАТЕЛЬНО изменить!
		defaultPassword := getEnv("ADMIN_PASSWORD", "admin")
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Ошибка хэширования пароля: %v", err)
			return
		}

		admin := models.Admin{
			Username: getEnv("ADMIN_USERNAME", "admin"),
			Password: string(hashedPassword),
		}

		if err := db.Create(&admin).Error; err != nil {
			log.Printf("Ошибка создания администратора по умолчанию: %v", err)
			return
		}

		log.Printf("Создан администратор по умолчанию: %s", admin.Username)
		log.Println("ВНИМАНИЕ: Измените пароль по умолчанию в продакшене!")
	}
}

// customErrorHandler обрабатывает ошибки Fiber
func customErrorHandler(c *fiber.Ctx, err error) error {
	// Определяем код ошибки
	code := fiber.StatusInternalServerError
	message := "Внутренняя ошибка сервера"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"success": false,
		"error":   message,
	})
}
