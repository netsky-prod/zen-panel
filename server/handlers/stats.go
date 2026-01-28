package handlers

import (
	"strconv"
	"time"

	"zen-admin/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// StatsHandler обрабатывает запросы статистики
type StatsHandler struct {
	db *gorm.DB
}

// NewStatsHandler создаёт новый обработчик статистики
func NewStatsHandler(db *gorm.DB) *StatsHandler {
	return &StatsHandler{db: db}
}

// OverallStats - общая статистика системы
type OverallStats struct {
	TotalUsers       int64 `json:"total_users"`
	ActiveUsers      int64 `json:"active_users"`
	TotalNodes       int64 `json:"total_nodes"`
	ActiveNodes      int64 `json:"active_nodes"`
	TotalInbounds    int64 `json:"total_inbounds"`
	TotalUpload      int64 `json:"total_upload"`
	TotalDownload    int64 `json:"total_download"`
	TodayUpload      int64 `json:"today_upload"`
	TodayDownload    int64 `json:"today_download"`
}

// UserTrafficHistory - история трафика пользователя
type UserTrafficHistory struct {
	Date     string `json:"date"`
	Upload   int64  `json:"upload"`
	Download int64  `json:"download"`
}

// NodeTrafficStats - статистика трафика ноды
type NodeTrafficStats struct {
	InboundID   uint   `json:"inbound_id"`
	InboundName string `json:"inbound_name"`
	Protocol    string `json:"protocol"`
	Upload      int64  `json:"upload"`
	Download    int64  `json:"download"`
	UserCount   int64  `json:"user_count"`
}

// GetOverall - GET /api/stats
// Общая статистика системы
func (h *StatsHandler) GetOverall(c *fiber.Ctx) error {
	var stats OverallStats

	// Общее количество пользователей
	h.db.Model(&models.User{}).Count(&stats.TotalUsers)

	// Активные пользователи (enabled = true и не истёк срок)
	h.db.Model(&models.User{}).Where("enabled = ? AND (expires_at IS NULL OR expires_at > ?)", true, time.Now()).Count(&stats.ActiveUsers)

	// Общее количество нод
	h.db.Model(&models.Node{}).Count(&stats.TotalNodes)

	// Активные ноды
	h.db.Model(&models.Node{}).Where("enabled = ?", true).Count(&stats.ActiveNodes)

	// Общее количество инбаундов
	h.db.Model(&models.Inbound{}).Count(&stats.TotalInbounds)

	// Суммарный трафик за всё время
	var totalTraffic struct {
		Upload   int64
		Download int64
	}
	h.db.Model(&models.TrafficStats{}).Select("COALESCE(SUM(upload), 0) as upload, COALESCE(SUM(download), 0) as download").Scan(&totalTraffic)
	stats.TotalUpload = totalTraffic.Upload
	stats.TotalDownload = totalTraffic.Download

	// Трафик за сегодня
	today := time.Now().Truncate(24 * time.Hour)
	var todayTraffic struct {
		Upload   int64
		Download int64
	}
	h.db.Model(&models.TrafficStats{}).
		Where("recorded_at >= ?", today).
		Select("COALESCE(SUM(upload), 0) as upload, COALESCE(SUM(download), 0) as download").
		Scan(&todayTraffic)
	stats.TodayUpload = todayTraffic.Upload
	stats.TodayDownload = todayTraffic.Download

	return c.JSON(fiber.Map{
		"success": true,
		"data":    stats,
	})
}

// GetUserStats - GET /api/stats/users/:id
// Статистика трафика пользователя
func (h *StatsHandler) GetUserStats(c *fiber.Ctx) error {
	userID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Неверный ID пользователя",
		})
	}

	// Проверяем существование пользователя
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
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

	// Период статистики (по умолчанию 30 дней)
	days, _ := strconv.Atoi(c.Query("days", "30"))
	if days < 1 || days > 365 {
		days = 30
	}

	startDate := time.Now().AddDate(0, 0, -days).Truncate(24 * time.Hour)

	// Получаем историю трафика по дням
	var history []struct {
		Date     time.Time
		Upload   int64
		Download int64
	}

	h.db.Model(&models.TrafficStats{}).
		Select("DATE(recorded_at) as date, SUM(upload) as upload, SUM(download) as download").
		Where("user_id = ? AND recorded_at >= ?", userID, startDate).
		Group("DATE(recorded_at)").
		Order("date ASC").
		Scan(&history)

	// Форматируем результат
	result := make([]UserTrafficHistory, len(history))
	for i, h := range history {
		result[i] = UserTrafficHistory{
			Date:     h.Date.Format("2006-01-02"),
			Upload:   h.Upload,
			Download: h.Download,
		}
	}

	// Суммарная статистика
	var totalStats struct {
		Upload   int64
		Download int64
	}
	h.db.Model(&models.TrafficStats{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(upload), 0) as upload, COALESCE(SUM(download), 0) as download").
		Scan(&totalStats)

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"user": fiber.Map{
				"id":         user.ID,
				"name":       user.Name,
				"data_used":  user.DataUsed,
				"data_limit": user.DataLimit,
			},
			"total": fiber.Map{
				"upload":   totalStats.Upload,
				"download": totalStats.Download,
			},
			"history": result,
		},
	})
}

// GetNodeStats - GET /api/stats/nodes/:id
// Статистика трафика ноды
func (h *StatsHandler) GetNodeStats(c *fiber.Ctx) error {
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

	// Получаем инбаунды ноды
	var inbounds []models.Inbound
	h.db.Where("node_id = ?", nodeID).Find(&inbounds)

	if len(inbounds) == 0 {
		return c.JSON(fiber.Map{
			"success": true,
			"data": fiber.Map{
				"node":     node,
				"inbounds": []NodeTrafficStats{},
				"total": fiber.Map{
					"upload":   0,
					"download": 0,
				},
			},
		})
	}

	// Собираем ID инбаундов
	inboundIDs := make([]uint, len(inbounds))
	for i, ib := range inbounds {
		inboundIDs[i] = ib.ID
	}

	// Статистика по инбаундам
	var inboundStats []struct {
		InboundID uint
		Upload    int64
		Download  int64
		UserCount int64
	}

	h.db.Model(&models.TrafficStats{}).
		Select("inbound_id, SUM(upload) as upload, SUM(download) as download, COUNT(DISTINCT user_id) as user_count").
		Where("inbound_id IN ?", inboundIDs).
		Group("inbound_id").
		Scan(&inboundStats)

	// Создаём map для быстрого доступа
	statsMap := make(map[uint]struct {
		Upload    int64
		Download  int64
		UserCount int64
	})
	for _, s := range inboundStats {
		statsMap[s.InboundID] = struct {
			Upload    int64
			Download  int64
			UserCount int64
		}{s.Upload, s.Download, s.UserCount}
	}

	// Формируем результат
	result := make([]NodeTrafficStats, len(inbounds))
	var totalUpload, totalDownload int64

	for i, ib := range inbounds {
		stats := statsMap[ib.ID]
		result[i] = NodeTrafficStats{
			InboundID:   ib.ID,
			InboundName: ib.Name,
			Protocol:    string(ib.Protocol),
			Upload:      stats.Upload,
			Download:    stats.Download,
			UserCount:   stats.UserCount,
		}
		totalUpload += stats.Upload
		totalDownload += stats.Download
	}

	// История трафика по дням
	days, _ := strconv.Atoi(c.Query("days", "30"))
	if days < 1 || days > 365 {
		days = 30
	}
	startDate := time.Now().AddDate(0, 0, -days).Truncate(24 * time.Hour)

	var history []struct {
		Date     time.Time
		Upload   int64
		Download int64
	}

	h.db.Model(&models.TrafficStats{}).
		Select("DATE(recorded_at) as date, SUM(upload) as upload, SUM(download) as download").
		Where("inbound_id IN ? AND recorded_at >= ?", inboundIDs, startDate).
		Group("DATE(recorded_at)").
		Order("date ASC").
		Scan(&history)

	historyResult := make([]UserTrafficHistory, len(history))
	for i, h := range history {
		historyResult[i] = UserTrafficHistory{
			Date:     h.Date.Format("2006-01-02"),
			Upload:   h.Upload,
			Download: h.Download,
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"node":     node,
			"inbounds": result,
			"total": fiber.Map{
				"upload":   totalUpload,
				"download": totalDownload,
			},
			"history": historyResult,
		},
	})
}

// RecordTraffic - внутренний метод для записи статистики
// Используется sync сервисом для сохранения данных с нод
func (h *StatsHandler) RecordTraffic(userID, inboundID uint, upload, download int64) error {
	stats := models.TrafficStats{
		UserID:     userID,
		InboundID:  inboundID,
		Upload:     upload,
		Download:   download,
		RecordedAt: time.Now(),
	}

	if err := h.db.Create(&stats).Error; err != nil {
		return err
	}

	// Обновляем data_used у пользователя
	h.db.Model(&models.User{}).Where("id = ?", userID).
		UpdateColumn("data_used", gorm.Expr("data_used + ?", upload+download))

	return nil
}

// GetTopUsers - GET /api/stats/top-users
// Топ пользователей по трафику
func (h *StatsHandler) GetTopUsers(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit < 1 || limit > 100 {
		limit = 10
	}

	var topUsers []struct {
		UserID   uint
		UserName string
		Upload   int64
		Download int64
		Total    int64
	}

	h.db.Model(&models.TrafficStats{}).
		Select("traffic_stats.user_id, users.name as user_name, SUM(traffic_stats.upload) as upload, SUM(traffic_stats.download) as download, SUM(traffic_stats.upload + traffic_stats.download) as total").
		Joins("JOIN users ON users.id = traffic_stats.user_id").
		Group("traffic_stats.user_id, users.name").
		Order("total DESC").
		Limit(limit).
		Scan(&topUsers)

	return c.JSON(fiber.Map{
		"success": true,
		"data":    topUsers,
	})
}
