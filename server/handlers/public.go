package handlers

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"os"

	"zen-admin/models"
	"zen-admin/services"

	"github.com/gofiber/fiber/v2"
	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

// PublicHandler обрабатывает публичные страницы для юзеров
type PublicHandler struct {
	db          *gorm.DB
	configGen   *services.ConfigGenerator
	publicURL   string
	subPassword string
}

// NewPublicHandler создаёт новый обработчик
func NewPublicHandler(db *gorm.DB) *PublicHandler {
	return &PublicHandler{
		db:          db,
		configGen:   services.NewConfigGenerator(),
		publicURL:   os.Getenv("PUBLIC_URL"),
		subPassword: os.Getenv("SUB_PASSWORD"),
	}
}

// checkSubPassword проверяет пароль подписки
func (h *PublicHandler) checkSubPassword(c *fiber.Ctx) bool {
	if h.subPassword == "" {
		return true // пароль не установлен - доступ открыт
	}
	key := c.Query("key")
	return key == h.subPassword
}

// UserConfigPage - GET /sub/:uuid
// Публичная страница с конфигом пользователя (замаскирована под буддизм)
func (h *PublicHandler) UserConfigPage(c *fiber.Ctx) error {
	userUUID := c.Params("uuid")
	if userUUID == "" {
		return c.Status(fiber.StatusNotFound).SendString("Page not found")
	}

	var user models.User
	if err := h.db.Preload("Inbounds.Node").Where("uuid = ?", userUUID).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Page not found")
	}

	if !user.Enabled {
		return c.Status(fiber.StatusNotFound).SendString("Page not found")
	}

	// Если пароль не совпадает - показываем страницу ввода пароля
	if !h.checkSubPassword(c) {
		return h.renderPasswordPage(c, userUUID)
	}

	// Пароль верный - показываем конфиги
	return h.renderConfigPage(c, &user, userUUID)
}

// renderPasswordPage показывает замаскированную страницу ввода пароля
func (h *PublicHandler) renderPasswordPage(c *fiber.Ctx, uuid string) error {
	html := `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Дзен-буддизм — Личный кабинет практикующего</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: 'Georgia', 'Times New Roman', serif;
            background: #1a1612;
            min-height: 100vh;
            color: #d4c5a9;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .container {
            max-width: 460px;
            width: 100%;
            padding: 20px;
        }
        .enso {
            text-align: center;
            font-size: 80px;
            margin-bottom: 20px;
            opacity: 0.7;
        }
        .title {
            text-align: center;
            font-size: 22px;
            font-weight: 400;
            color: #c4a265;
            margin-bottom: 8px;
            letter-spacing: 2px;
        }
        .subtitle {
            text-align: center;
            font-size: 13px;
            color: #8a7a60;
            margin-bottom: 40px;
            font-style: italic;
        }
        .form-card {
            background: rgba(255,255,255,0.03);
            border: 1px solid rgba(196,162,101,0.15);
            border-radius: 12px;
            padding: 30px;
        }
        .form-label {
            font-size: 13px;
            color: #8a7a60;
            margin-bottom: 10px;
            display: block;
        }
        .form-input {
            width: 100%;
            padding: 14px 16px;
            background: rgba(0,0,0,0.3);
            border: 1px solid rgba(196,162,101,0.2);
            border-radius: 8px;
            color: #d4c5a9;
            font-size: 16px;
            font-family: inherit;
            outline: none;
            transition: border-color 0.3s;
        }
        .form-input:focus {
            border-color: rgba(196,162,101,0.5);
        }
        .form-input::placeholder {
            color: #5a4f3f;
        }
        .form-btn {
            width: 100%;
            padding: 14px;
            background: linear-gradient(135deg, #8b6914, #c4a265);
            border: none;
            border-radius: 8px;
            color: #1a1612;
            font-size: 15px;
            font-weight: 600;
            font-family: inherit;
            cursor: pointer;
            margin-top: 16px;
            transition: opacity 0.3s;
            letter-spacing: 1px;
        }
        .form-btn:hover { opacity: 0.85; }
        .error-msg {
            color: #a05a5a;
            font-size: 13px;
            margin-top: 12px;
            text-align: center;
            display: none;
        }
        .quote {
            text-align: center;
            font-size: 12px;
            color: #5a4f3f;
            margin-top: 30px;
            font-style: italic;
            line-height: 1.6;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="enso">&#9775;</div>
        <div class="title">Личный кабинет</div>
        <div class="subtitle">Портал практикующего дзен-буддизм</div>

        <div class="form-card">
            <label class="form-label">Введите код доступа к материалам</label>
            <input type="password" id="passInput" class="form-input" placeholder="Код практикующего" autofocus>
            <button class="form-btn" id="enterBtn" onclick="enter()">Войти</button>
            <div class="error-msg" id="errMsg">Неверный код. Обратитесь к наставнику.</div>
        </div>

        <div class="quote">
            «Путь в тысячу ли начинается с одного шага»<br>— Лао-цзы
        </div>
    </div>

    <script>
        var uuid = '` + template.JSEscapeString(uuid) + `';
        document.getElementById('passInput').addEventListener('keydown', function(e) {
            if (e.key === 'Enter') enter();
        });
        function enter() {
            var key = document.getElementById('passInput').value;
            if (!key) return;
            window.location.href = window.location.pathname + '?key=' + encodeURIComponent(key);
        }
    </script>
</body>
</html>`

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html)
}

// renderConfigPage показывает конфиги (замаскировано под буддийские материалы)
func (h *PublicHandler) renderConfigPage(c *fiber.Ctx, user *models.User, userUUID string) error {
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

		shareURL, err := h.configGen.GenerateShareURL(user, &inbound)
		if err != nil {
			continue
		}

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

	baseURL := h.publicURL
	if baseURL == "" {
		baseURL = c.BaseURL()
	}
	key := c.Query("key")
	subscriptionURL := fmt.Sprintf("%s/api/sub/%s/raw", baseURL, userUUID)
	if key != "" {
		subscriptionURL += "?key=" + key
	}

	dataLimit := "Unlimited"
	if user.DataLimit > 0 {
		dataLimit = formatBytes(user.DataLimit)
	}
	dataUsed := formatBytes(user.DataUsed)

	html := `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Дзен-буддизм — Материалы практикующего</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: 'Georgia', 'Times New Roman', serif;
            background: #1a1612;
            min-height: 100vh;
            color: #d4c5a9;
            padding: 20px;
        }
        .container { max-width: 600px; margin: 0 auto; }
        .header {
            text-align: center;
            padding: 25px 0 20px;
        }
        .enso { font-size: 48px; opacity: 0.6; margin-bottom: 10px; }
        .header h1 {
            font-size: 22px;
            font-weight: 400;
            color: #c4a265;
            letter-spacing: 2px;
        }
        .header .practitioner {
            color: #6a5f4f;
            font-size: 13px;
            margin-top: 6px;
            font-style: italic;
        }
        .stats {
            display: flex;
            gap: 12px;
            margin-bottom: 20px;
        }
        .stat {
            flex: 1;
            background: rgba(196,162,101,0.05);
            border: 1px solid rgba(196,162,101,0.1);
            border-radius: 10px;
            padding: 14px;
            text-align: center;
        }
        .stat-value { font-size: 18px; font-weight: 600; color: #c4a265; }
        .stat-label { font-size: 11px; color: #6a5f4f; margin-top: 4px; }
        .card {
            background: rgba(255,255,255,0.02);
            border-radius: 12px;
            padding: 20px;
            margin-bottom: 16px;
            border: 1px solid rgba(196,162,101,0.1);
        }
        .card-title {
            font-size: 16px;
            margin-bottom: 14px;
            display: flex;
            align-items: center;
            gap: 10px;
            color: #c4a265;
        }
        .badge {
            font-size: 10px;
            padding: 3px 8px;
            border-radius: 4px;
            background: rgba(196,162,101,0.15);
            color: #c4a265;
            text-transform: uppercase;
            letter-spacing: 1px;
            font-family: -apple-system, sans-serif;
        }
        .qr-container {
            text-align: center;
            margin: 12px 0;
        }
        .qr-container img {
            background: #fff;
            padding: 8px;
            border-radius: 10px;
            max-width: 260px;
            width: 100%;
        }
        .url-box {
            background: rgba(0,0,0,0.3);
            border-radius: 8px;
            padding: 12px 50px 12px 12px;
            font-family: 'Courier New', monospace;
            font-size: 10px;
            word-break: break-all;
            color: #7a6f5f;
            margin: 10px 0;
            position: relative;
            line-height: 1.5;
        }
        .copy-btn {
            position: absolute;
            right: 6px;
            top: 6px;
            background: linear-gradient(135deg, #8b6914, #c4a265);
            border: none;
            color: #1a1612;
            padding: 5px 10px;
            border-radius: 5px;
            cursor: pointer;
            font-size: 11px;
            font-weight: 600;
            font-family: -apple-system, sans-serif;
        }
        .copy-btn:hover { opacity: 0.85; }
        .copy-btn.copied { background: #5a8a5a; color: #fff; }
        .sub-section {
            background: rgba(196,162,101,0.04);
            border: 1px solid rgba(196,162,101,0.15);
        }
        .hint {
            font-size: 11px;
            color: #5a4f3f;
            margin-top: 8px;
            font-style: italic;
        }
        .apps {
            display: flex;
            gap: 8px;
            flex-wrap: wrap;
            margin-top: 12px;
        }
        .app {
            background: rgba(196,162,101,0.08);
            border: 1px solid rgba(196,162,101,0.1);
            padding: 5px 10px;
            border-radius: 6px;
            font-size: 11px;
            color: #8a7a60;
            font-family: -apple-system, sans-serif;
        }
        .tabs {
            display: flex;
            gap: 4px;
            margin-bottom: 12px;
        }
        .tab {
            flex: 1;
            padding: 8px;
            background: rgba(196,162,101,0.05);
            border: 1px solid rgba(196,162,101,0.08);
            color: #6a5f4f;
            border-radius: 6px;
            cursor: pointer;
            font-size: 12px;
            font-family: inherit;
        }
        .tab.active {
            background: rgba(196,162,101,0.12);
            color: #c4a265;
            border-color: rgba(196,162,101,0.2);
        }
        .tab-content { display: none; }
        .tab-content.active { display: block; }
        .footer {
            text-align: center;
            margin-top: 25px;
            padding: 15px;
            font-size: 11px;
            color: #3a352d;
            font-style: italic;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="enso">&#9775;</div>
            <h1>Материалы практикующего</h1>
            <div class="practitioner">{{.UserName}}</div>
        </div>

        <div class="stats">
            <div class="stat">
                <div class="stat-value">{{.DataUsed}}</div>
                <div class="stat-label">Использовано</div>
            </div>
            <div class="stat">
                <div class="stat-value">{{.DataLimit}}</div>
                <div class="stat-label">Доступно</div>
            </div>
        </div>

        <div class="card sub-section">
            <div class="card-title">Ссылка для приложений</div>
            <p class="hint">Добавьте эту ссылку в приложение для автоматического обновления:</p>
            <div class="url-box">
                <span id="sub-url">{{.SubscriptionURL}}</span>
                <button class="copy-btn" onclick="copyText('sub-url', this)">Copy</button>
            </div>
            <div class="apps">
                <span class="app">v2rayNG</span>
                <span class="app">Shadowrocket</span>
                <span class="app">Hiddify</span>
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
                <button class="tab active" onclick="showTab({{$i}}, 'qr', this)">QR-код</button>
                <button class="tab" onclick="showTab({{$i}}, 'url', this)">Ссылка</button>
            </div>

            <div id="tab-{{$i}}-qr" class="tab-content active">
                <div class="qr-container">
                    <img src="data:image/png;base64,{{$inbound.QRCode}}" alt="QR">
                </div>
                <p class="hint" style="text-align:center">Отсканируйте камерой или из приложения</p>
            </div>

            <div id="tab-{{$i}}-url" class="tab-content">
                <div class="url-box">
                    <span id="url-{{$i}}">{{$inbound.URL}}</span>
                    <button class="copy-btn" onclick="copyText('url-{{$i}}', this)">Copy</button>
                </div>
            </div>
        </div>
        {{end}}

        <div class="footer">
            zen-buddhism.ru
        </div>
    </div>

    <script>
        function copyText(id, btn) {
            var text = document.getElementById(id).innerText;
            navigator.clipboard.writeText(text).then(function() {
                btn.innerText = 'Copied!';
                btn.classList.add('copied');
                setTimeout(function() {
                    btn.innerText = 'Copy';
                    btn.classList.remove('copied');
                }, 2000);
            });
        }

        function showTab(cardIndex, tabName, tabBtn) {
            var cards = document.querySelectorAll('.card');
            var card = cards[cardIndex + 1];
            card.querySelectorAll('.tab').forEach(function(t) { t.classList.remove('active'); });
            card.querySelectorAll('.tab-content').forEach(function(c) { c.classList.remove('active'); });
            tabBtn.classList.add('active');
            document.getElementById('tab-' + cardIndex + '-' + tabName).classList.add('active');
        }
    </script>
</body>
</html>`

	tmpl, err := template.New("config").Parse(html)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Page not found")
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
// Raw subscription для импорта в приложения (защищён паролем)
func (h *PublicHandler) RawSubscription(c *fiber.Ctx) error {
	if !h.checkSubPassword(c) {
		return c.Status(fiber.StatusForbidden).SendString("Access denied")
	}

	userUUID := c.Params("uuid")
	if userUUID == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid request")
	}

	var user models.User
	if err := h.db.Preload("Inbounds.Node").Where("uuid = ?", userUUID).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Not found")
	}

	if !user.Enabled {
		return c.Status(fiber.StatusForbidden).SendString("Disabled")
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
