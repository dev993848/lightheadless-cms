# LightHeadless CMS

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Build](https://img.shields.io/badge/build-passing-brightgreen)]()
[![Tests](https://img.shields.io/badge/tests-125%20passing-brightgreen)]()

**LightHeadless CMS** is a lightweight headless CMS built with Go for collecting landing page leads and managing news.  
Single binary, SQLite, no external dependencies, no CGO.

---

## Features

- 🚀 **Fast startup** — single binary, no external dependencies
- 💾 **SQLite** — embedded DB with WAL mode for high performance
- 📊 **Lead collection** — REST API for HTML forms, rate limiting, validation
- 📰 **News management** — CRUD with Markdown support and image uploads
- 🔗 **Bitrix24** — async lead submission via webhook (worker pool)
- 🔒 **Security** — bcrypt, CSRF, session cookies, SQL injection protection
- 🎨 **Modern UI** — Tailwind CSS, HTMX, Alpine.js, responsive design
- 🧪 **Tests** — 125 tests, race detector, ~100% coverage on critical packages

---

## Quick Start

### 1. Download or Build

```bash
# Clone the repository
git clone https://github.com/yourusername/lightheadless-cms.git
cd lightheadless-cms/cms

# Build
go build -o cms.exe ./cmd/server
```

### 2. Run

```bash
./cms.exe
```

Or with custom settings:

```bash
./cms.exe -port 8080 -db ./cms.db -upload ./uploads
```

Or via environment variables:

```bash
CMS_PORT=8080 CMS_DB_PATH=./cms.db CMS_UPLOAD_PATH=./uploads ./cms.exe
```

### 3. Login to Admin Panel

On first run, CMS outputs credentials to console:

```
╔══════════════════════════════════════════════════╗
║          FIRST RUN — ADMIN CREDENTIALS           ║
╠══════════════════════════════════════════════════╣
║  Email:    admin@example.com                     ║
║  Password: xK7mP2qR9nLs                          ║
║  Change password in Settings!                    ║
╚══════════════════════════════════════════════════╝
```

Open browser: **http://localhost:8080/admin/**

---

## Capabilities

### Public API

| Endpoint | Method | Description |
|---|---|---|
| `/api/leads` | POST | Submit lead (rate limit: 10/min) |
| `/api/news` | GET | News list with pagination |
| `/api/news/{id}` | GET | Single news + HTML |
| `/uploads/{filename}` | GET | Static files |

**Example lead submission:**

```bash
curl -X POST http://localhost:8080/api/leads \
  -H "Content-Type: application/json" \
  -d '{"name":"Ivan","phone":"+7 999 123-45-67","email":"ivan@example.com","comment":"Lead from website"}'
```

**Response:**

```json
{
  "status": "success",
  "message": "Lead saved"
}
```

### Admin Panel

| Section | Features |
|---|---|
| **Dashboard** | Statistics, recent leads, quick actions |
| **Leads** | List, filters, search, details, CSV export, Bitrix24 submission |
| **News** | CRUD, Markdown editor, live preview, image upload |
| **Settings** | Admin email/password, Bitrix24 webhook, test connection |

---

## Bitrix24 Integration

1. In admin panel go to **Settings** → **Bitrix24 Integration**
2. Create incoming webhook in Bitrix24 (method `crm.lead.add`)
3. Paste webhook URL into **Webhook URL** field
4. Click **Test Connection**
5. Enable **Enable submission** checkbox
6. Save settings

Leads will be asynchronously submitted to Bitrix24 via worker pool (10 goroutines).

---

## Architecture

```
cms/
├── cmd/server/           # Entry point
├── internal/
│   ├── config/           # Configuration (flags + env)
│   ├── db/               # CRUD operations, migrations
│   ├── handlers/         # HTTP handlers
│   ├── middleware/       # Auth, CSRF, rate limit, logger
│   ├── models/           # Data structures
│   ├── services/         # Bitrix24, upload
│   └── validate/         # Validation + sanitization
├── ui/templates/         # HTML templates (embed)
├── static/               # CSS, JS (embed)
├── uploads/              # Uploaded files (runtime)
└── go.mod
```

### Technology Stack

| Component | Technology |
|---|---|
| Language | Go 1.22+ |
| Database | SQLite (modernc.org/sqlite, WAL mode) |
| Markdown | github.com/yuin/goldmark |
| Crypto | golang.org/x/crypto (bcrypt) |
| HTTP | net/http (stdlib) |
| Templates | html/template (stdlib) |
| UI | Tailwind CSS, HTMX, Alpine.js |

---

## Development

### Requirements

- Go 1.22+
- Git

### Build

```bash
cd cms
go build -o cms.exe ./cmd/server
```

### Tests

```bash
# All tests
go test ./... -v -race -count=1

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Makefile

```bash
make build         # Build
make test          # Tests
make test-cover    # Tests + coverage
make run           # Run in dev mode
make clean         # Clean
```

---

## Configuration

### CLI Flags

| Flag | Default | Description |
|---|---|---|
| `-port` | 8080 | HTTP port |
| `-db` | ./cms.db | SQLite database path |
| `-upload` | ./uploads | Upload directory path |

### Environment Variables

| Variable | Description |
|---|---|
| `CMS_PORT` | Port (overrides flag) |
| `CMS_DB_PATH` | Database path (overrides flag) |
| `CMS_UPLOAD_PATH` | Upload path (overrides flag) |

---

## Security

- ✅ **SQL injection** — parameterized queries
- ✅ **XSS** — output escaping (html.EscapeString)
- ✅ **CSRF** — HMAC-SHA256 tokens
- ✅ **Session hijacking** — cryptographically random session IDs (32 bytes)
- ✅ **Password storage** — bcrypt (cost=10)
- ✅ **Rate limiting** — 10 requests per minute per IP
- ✅ **Cookies** — HttpOnly, SameSite=Lax

---

## Performance

| Metric | Value |
|---|---|
| Response time /api/leads | < 100ms |
| Concurrent leads | 100 RPS |
| Session expiry | 24 hours |
| Rate limit | 10 req/min |
| Upload max size | 5 MB |
| Worker pool | 10 goroutines |
| Queue size | 100 leads |

---

## Demo Landing Page

A test landing page is included in the `landing/` directory:

- 📰 News display from CMS API
- 📩 Lead form with validation
- 🎨 Modern responsive design

**Run:**

```bash
# Option 1: Open file directly (may have CORS issues)
open landing/index.html

# Option 2: Use local server
cd landing
python -m http.server 3000

# Open: http://localhost:3000
```

Or access via CMS: **http://localhost:8080/static/index.html**

---

## License

MIT License — see [LICENSE](LICENSE) file for details.

---

## Changelog

### 1.0 (2026-03-26)

- ✅ Initial release
- ✅ All 7 development stages completed
- ✅ 125 tests passing
- ✅ UI/UX polished
- ✅ Documentation complete

---

## Author

**Anton Budylin**  
Email: aabudilin@gmail.com

---

## Roadmap (Optional)

- [ ] Multi-user mode with roles
- [ ] AmoCRM, HubSpot support
- [ ] Webhooks for external integrations
- [ ] API keys
- [ ] Docker image
- [ ] Notifications (Telegram, Email)
- [ ] Analytics and charts
- [ ] Excel export
- [ ] Custom form fields

---

## Support

- **Documentation:** `instruction/` directory
- **API Docs:** `project/api.md`
- **User Guide:** `instruction/USER_GUIDE.md`
- **Integration Guide:** `instruction/AI_INTEGRATION_GUIDE.md`

**LightHeadless CMS** — lightweight CMS for landing pages and small websites.
