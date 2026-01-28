package handlers

import (
	"encoding/base64"
	"fmt"
	"html/template"

	"zen-admin/models"
	"zen-admin/services"

	"github.com/gofiber/fiber/v2"
	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

// PublicHandler обрабатывает публичные страницы для юзеров
type PublicHandler struct {
	db        *gorm.DB
	configGen *services.ConfigGenerator
}

// NewPublicHandler создаёт новый обработчик
func NewPublicHandler(db *gorm.DB) *PublicHandler {
	return &PublicHandler{
		db:        db,
		configGen: services.NewConfigGenerator(),
	}
}

// UserConfigPage - GET /sub/:uuid
// Публичная страница с конфигом пользователя
func (h *PublicHandler) UserConfigPage(c *fiber.Ctx) error {
	userUUID := c.Params("uuid")
	if userUUID == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid UUID")
	}

	var user models.User
	if err := h.db.Preload("Inbounds.Node").Where("uuid = ?", userUUID).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).SendString("User not found")
	}

	if !user.Enabled {
		return c.Status(fiber.StatusForbidden).SendString("Account disabled")
	}

	if len(user.Inbounds) == 0 {
		return c.Status(fiber.StatusNotFound).SendString("No servers configured")
	}

	// Генерируем данные для каждого inbound
	type InboundData struct {
		Name     string
		NodeName string
		Protocol string
		URL      string
		QRCode   string
	}

	var inbounds []InboundData
	for _, inbound := range user.Inbounds {
		if !inbound.Enabled {
			continue
		}

		shareURL, err := h.configGen.GenerateShareURL(&user, &inbound)
		if err != nil {
			continue
		}

		// Генерируем QR код
		qr, err := qrcode.New(shareURL, qrcode.Medium)
		if err != nil {
			continue
		}
		png, err := qr.PNG(300)
		if err != nil {
			continue
		}
		qrBase64 := base64.StdEncoding.EncodeToString(png)

		inbounds = append(inbounds, InboundData{
			Name:     inbound.Name,
			NodeName: inbound.Node.Name,
			Protocol: string(inbound.Protocol),
			URL:      shareURL,
			QRCode:   qrBase64,
		})
	}

	// Subscription URL
	subscriptionURL := fmt.Sprintf("%s/sub/%s/raw", c.BaseURL(), userUUID)

	// Форматируем лимит трафика
	dataLimit := "Unlimited"
	if user.DataLimit > 0 {
		dataLimit = formatBytes(user.DataLimit)
	}
	dataUsed := formatBytes(user.DataUsed)

	// HTML страница
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>VPN Config - {{.UserName}}</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
            min-height: 100vh;
            color: #fff;
            padding: 20px;
        }
        .container { max-width: 600px; margin: 0 auto; }
        .header {
            text-align: center;
            padding: 30px 0;
        }
        .header h1 {
            font-size: 28px;
            margin-bottom: 10px;
            background: linear-gradient(90deg, #00d4ff, #7b2cbf);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        .header .user-name {
            color: #888;
            font-size: 14px;
        }
        .stats {
            display: flex;
            gap: 15px;
            margin-bottom: 25px;
        }
        .stat {
            flex: 1;
            background: rgba(255,255,255,0.05);
            border-radius: 12px;
            padding: 15px;
            text-align: center;
        }
        .stat-value { font-size: 20px; font-weight: 600; color: #00d4ff; }
        .stat-label { font-size: 12px; color: #888; margin-top: 5px; }
        .card {
            background: rgba(255,255,255,0.05);
            border-radius: 16px;
            padding: 20px;
            margin-bottom: 20px;
            border: 1px solid rgba(255,255,255,0.1);
        }
        .card-title {
            font-size: 18px;
            margin-bottom: 15px;
            display: flex;
            align-items: center;
            gap: 10px;
        }
        .badge {
            font-size: 11px;
            padding: 4px 8px;
            border-radius: 6px;
            background: #7b2cbf;
            text-transform: uppercase;
        }
        .qr-container {
            text-align: center;
            margin: 15px 0;
        }
        .qr-container img {
            background: #fff;
            padding: 10px;
            border-radius: 12px;
            max-width: 100%;
        }
        .url-box {
            background: rgba(0,0,0,0.3);
            border-radius: 8px;
            padding: 12px;
            font-family: monospace;
            font-size: 11px;
            word-break: break-all;
            color: #aaa;
            margin: 10px 0;
            position: relative;
        }
        .copy-btn {
            position: absolute;
            right: 8px;
            top: 8px;
            background: #00d4ff;
            border: none;
            color: #000;
            padding: 6px 12px;
            border-radius: 6px;
            cursor: pointer;
            font-size: 12px;
            font-weight: 600;
        }
        .copy-btn:hover { background: #00b8e6; }
        .copy-btn.copied { background: #4ade80; }
        .subscription-section {
            background: linear-gradient(135deg, rgba(0,212,255,0.1), rgba(123,44,191,0.1));
            border: 1px solid rgba(0,212,255,0.3);
        }
        .subscription-section .card-title { color: #00d4ff; }
        .help-text {
            font-size: 12px;
            color: #888;
            margin-top: 10px;
        }
        .apps {
            display: flex;
            gap: 10px;
            flex-wrap: wrap;
            margin-top: 15px;
        }
        .app {
            background: rgba(255,255,255,0.1);
            padding: 8px 12px;
            border-radius: 8px;
            font-size: 12px;
        }
        .tabs {
            display: flex;
            gap: 5px;
            margin-bottom: 15px;
        }
        .tab {
            flex: 1;
            padding: 10px;
            background: rgba(255,255,255,0.05);
            border: none;
            color: #888;
            border-radius: 8px;
            cursor: pointer;
            font-size: 13px;
        }
        .tab.active {
            background: rgba(0,212,255,0.2);
            color: #00d4ff;
        }
        .tab-content { display: none; }
        .tab-content.active { display: block; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔐 Zen VPN</h1>
            <div class="user-name">{{.UserName}}</div>
        </div>

        <div class="stats">
            <div class="stat">
                <div class="stat-value">{{.DataUsed}}</div>
                <div class="stat-label">Used</div>
            </div>
            <div class="stat">
                <div class="stat-value">{{.DataLimit}}</div>
                <div class="stat-label">Limit</div>
            </div>
        </div>

        <div class="card subscription-section">
            <div class="card-title">📡 Subscription URL</div>
            <p class="help-text">Add this URL to your VPN app for auto-updates:</p>
            <div class="url-box">
                <span id="sub-url">{{.SubscriptionURL}}</span>
                <button class="copy-btn" onclick="copyText('sub-url', this)">Copy</button>
            </div>
            <div class="apps">
                <span class="app">v2rayNG</span>
                <span class="app">Shadowrocket</span>
                <span class="app">Clash</span>
                <span class="app">NekoBox</span>
            </div>
        </div>

        {{range $i, $inbound := .Inbounds}}
        <div class="card">
            <div class="card-title">
                {{$inbound.NodeName}}
                <span class="badge">{{$inbound.Protocol}}</span>
            </div>

            <div class="tabs">
                <button class="tab active" onclick="showTab({{$i}}, 'qr')">QR Code</button>
                <button class="tab" onclick="showTab({{$i}}, 'url')">URL</button>
            </div>

            <div id="tab-{{$i}}-qr" class="tab-content active">
                <div class="qr-container">
                    <img src="data:image/png;base64,{{$inbound.QRCode}}" alt="QR Code">
                </div>
                <p class="help-text" style="text-align:center">Scan with your phone camera or VPN app</p>
            </div>

            <div id="tab-{{$i}}-url" class="tab-content">
                <div class="url-box">
                    <span id="url-{{$i}}">{{$inbound.URL}}</span>
                    <button class="copy-btn" onclick="copyText('url-{{$i}}', this)">Copy</button>
                </div>
            </div>
        </div>
        {{end}}
    </div>

    <script>
        function copyText(id, btn) {
            const text = document.getElementById(id).innerText;
            navigator.clipboard.writeText(text).then(() => {
                btn.innerText = 'Copied!';
                btn.classList.add('copied');
                setTimeout(() => {
                    btn.innerText = 'Copy';
                    btn.classList.remove('copied');
                }, 2000);
            });
        }

        function showTab(cardIndex, tabName) {
            const card = document.querySelectorAll('.card')[cardIndex + 1];
            card.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
            card.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));

            event.target.classList.add('active');
            document.getElementById('tab-' + cardIndex + '-' + tabName).classList.add('active');
        }
    </script>
</body>
</html>`

	tmpl, err := template.New("config").Parse(html)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Template error")
	}

	data := struct {
		UserName        string
		DataUsed        string
		DataLimit       string
		SubscriptionURL string
		Inbounds        []InboundData
	}{
		UserName:        user.Name,
		DataUsed:        dataUsed,
		DataLimit:       dataLimit,
		SubscriptionURL: subscriptionURL,
		Inbounds:        inbounds,
	}

	c.Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.Execute(c.Response().BodyWriter(), data)
}

// RawSubscription - GET /sub/:uuid/raw
// Raw subscription для импорта в приложения
func (h *PublicHandler) RawSubscription(c *fiber.Ctx) error {
	userUUID := c.Params("uuid")
	if userUUID == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid UUID")
	}

	var user models.User
	if err := h.db.Preload("Inbounds.Node").Where("uuid = ?", userUUID).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).SendString("User not found")
	}

	if !user.Enabled {
		return c.Status(fiber.StatusForbidden).SendString("User disabled")
	}

	subscription, err := h.configGen.GenerateSubscription(&user, user.Inbounds)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error")
	}

	c.Set("Content-Type", "text/plain; charset=utf-8")
	c.Set("Profile-Update-Interval", "12")
	c.Set("Subscription-Userinfo", fmt.Sprintf("upload=0; download=%d; total=%d", user.DataUsed, user.DataLimit))
	return c.SendString(subscription)
}

func formatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
