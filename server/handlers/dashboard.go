package handlers

import (
	"time"

	"zen-admin/models"
	"zen-admin/services"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// DashboardHandler обрабатывает запросы дашборда
type DashboardHandler struct {
	db         *gorm.DB
	nodeClient *services.NodeClient
}

// NewDashboardHandler создаёт новый обработчик дашборда
func NewDashboardHandler(db *gorm.DB) *DashboardHandler {
	return &DashboardHandler{
		db:         db,
		nodeClient: services.NewNodeClient(),
	}
}

// DashboardSummary - сводка для дашборда
type DashboardSummary struct {
	Users       UsersSummary       `json:"users"`
	Nodes       NodesSummary       `json:"nodes"`
	Traffic     TrafficSummary     `json:"traffic"`
	RecentUsers []RecentUser       `json:"recent_users"`
	NodeStatus  []NodeStatusInfo   `json:"node_status"`
}

// UsersSummary - сводка по пользователям
type UsersSummary struct {
	Total    int64 `json:"total"`
	Active   int64 `json:"active"`
	Disabled int64 `json:"disabled"`
	Expired  int64 `json:"expired"`
}

// NodesSummary - сводка по нодам
type NodesSummary struct {
	Total    int64 `json:"total"`
	Online   int64 `json:"online"`
	Offline  int64 `json:"offline"`
	Disabled int64 `json:"disabled"`
}

// TrafficSummary - сводка по трафику
type TrafficSummary struct {
	TotalUpload     int64              `json:"total_upload"`
	TotalDownload   int64              `json:"total_download"`
	TodayUpload     int64              `json:"today_upload"`
	TodayDownload   int64              `json:"today_download"`
	WeeklyHistory   []DailyTraffic     `json:"weekly_history"`
}

// DailyTraffic - дневной трафик
type DailyTraffic struct {
	Date     string `json:"date"`
	Upload   int64  `json:"upload"`
	Download int64  `json:"download"`
}

// RecentUser - недавно добавленный пользователь
type RecentUser struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

// NodeStatusInfo - статус ноды для дашборда
type NodeStatusInfo struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	Address      string `json:"address"`
	Online       bool   `json:"online"`
	InboundCount int    `json:"inbound_count"`
	UserCount    int64  `json:"user_count"`
}

// Get - GET /api/dashboard
// Сводка для дашборда
func (h *DashboardHandler) Get(c *fiber.Ctx) error {
	summary := DashboardSummary{}

	// === Пользователи ===
	h.db.Model(&models.User{}).Count(&summary.Users.Total)

	// Активные (enabled и срок не истёк)
	h.db.Model(&models.User{}).
		Where("enabled = ? AND (expires_at IS NULL OR expires_at > ?)", true, time.Now()).
		Count(&summary.Users.Active)

	// Отключённые
	h.db.Model(&models.User{}).
		Where("enabled = ?", false).
		Count(&summary.Users.Disabled)

	// С истёкшим сроком
	h.db.Model(&models.User{}).
		Where("expires_at IS NOT NULL AND expires_at <= ?", time.Now()).
		Count(&summary.Users.Expired)

	// === Ноды ===
	var nodes []models.Node
	h.db.Preload("Inbounds").Find(&nodes)

	summary.Nodes.Total = int64(len(nodes))

	// Проверяем статус каждой ноды
	for _, node := range nodes {
		if !node.Enabled {
			summary.Nodes.Disabled++
			continue
		}

		status, _ := h.nodeClient.GetStatus(&node)
		if status != nil && status.Online {
			summary.Nodes.Online++
		} else {
			summary.Nodes.Offline++
		}
	}

	// === Трафик ===
	var totalTraffic struct {
		Upload   int64
		Download int64
	}
	h.db.Model(&models.TrafficStats{}).
		Select("COALESCE(SUM(upload), 0) as upload, COALESCE(SUM(download), 0) as download").
		Scan(&totalTraffic)
	summary.Traffic.TotalUpload = totalTraffic.Upload
	summary.Traffic.TotalDownload = totalTraffic.Download

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
	summary.Traffic.TodayUpload = todayTraffic.Upload
	summary.Traffic.TodayDownload = todayTraffic.Download

	// История за последние 7 дней
	weekAgo := time.Now().AddDate(0, 0, -7).Truncate(24 * time.Hour)
	var weeklyHistory []struct {
		Date     time.Time
		Upload   int64
		Download int64
	}
	h.db.Model(&models.TrafficStats{}).
		Select("DATE(recorded_at) as date, SUM(upload) as upload, SUM(download) as download").
		Where("recorded_at >= ?", weekAgo).
		Group("DATE(recorded_at)").
		Order("date ASC").
		Scan(&weeklyHistory)

	summary.Traffic.WeeklyHistory = make([]DailyTraffic, len(weeklyHistory))
	for i, h := range weeklyHistory {
		summary.Traffic.WeeklyHistory[i] = DailyTraffic{
			Date:     h.Date.Format("2006-01-02"),
			Upload:   h.Upload,
			Download: h.Download,
		}
	}

	// === Недавние пользователи ===
	var recentUsers []models.User
	h.db.Order("created_at DESC").Limit(5).Find(&recentUsers)

	summary.RecentUsers = make([]RecentUser, len(recentUsers))
	for i, u := range recentUsers {
		summary.RecentUsers[i] = RecentUser{
			ID:        u.ID,
			Name:      u.Name,
			Enabled:   u.Enabled,
			CreatedAt: u.CreatedAt,
		}
	}

	// === Статус нод ===
	summary.NodeStatus = make([]NodeStatusInfo, len(nodes))
	for i, node := range nodes {
		// Считаем пользователей привязанных к инбаундам этой ноды
		var userCount int64
		inboundIDs := make([]uint, len(node.Inbounds))
		for j, ib := range node.Inbounds {
			inboundIDs[j] = ib.ID
		}
		if len(inboundIDs) > 0 {
			h.db.Model(&models.UserInbound{}).
				Where("inbound_id IN ?", inboundIDs).
				Distinct("user_id").
				Count(&userCount)
		}

		online := false
		if node.Enabled {
			status, _ := h.nodeClient.GetStatus(&node)
			online = status != nil && status.Online
		}

		summary.NodeStatus[i] = NodeStatusInfo{
			ID:           node.ID,
			Name:         node.Name,
			Address:      node.Address,
			Online:       online,
			InboundCount: len(node.Inbounds),
			UserCount:    userCount,
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    summary,
	})
}

// GetQuickStats - GET /api/dashboard/quick
// Быстрая статистика (без проверки статуса нод)
func (h *DashboardHandler) GetQuickStats(c *fiber.Ctx) error {
	var stats struct {
		TotalUsers    int64 `json:"total_users"`
		ActiveUsers   int64 `json:"active_users"`
		TotalNodes    int64 `json:"total_nodes"`
		TotalInbounds int64 `json:"total_inbounds"`
		TodayTraffic  int64 `json:"today_traffic"`
	}

	h.db.Model(&models.User{}).Count(&stats.TotalUsers)
	h.db.Model(&models.User{}).
		Where("enabled = ? AND (expires_at IS NULL OR expires_at > ?)", true, time.Now()).
		Count(&stats.ActiveUsers)
	h.db.Model(&models.Node{}).Where("enabled = ?", true).Count(&stats.TotalNodes)
	h.db.Model(&models.Inbound{}).Where("enabled = ?", true).Count(&stats.TotalInbounds)

	today := time.Now().Truncate(24 * time.Hour)
	var todayTraffic struct {
		Total int64
	}
	h.db.Model(&models.TrafficStats{}).
		Where("recorded_at >= ?", today).
		Select("COALESCE(SUM(upload + download), 0) as total").
		Scan(&todayTraffic)
	stats.TodayTraffic = todayTraffic.Total

	return c.JSON(fiber.Map{
		"success": true,
		"data":    stats,
	})
}
