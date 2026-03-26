# Architecture & Technical Plan

## Стек (финальный)

| Компонент | Пакет | Версия |
|---|---|---|
| Go | — | 1.21+ |
| SQLite driver | `modernc.org/sqlite` | latest (pure Go, no CGO) |
| Markdown | `github.com/yuin/goldmark` | latest |
| Crypto | `golang.org/x/crypto` (bcrypt) | latest |
| Embed | `embed` (stdlib) | — |
| HTTP | `net/http` (stdlib) | — |
| Templates | `html/template` (stdlib) | — |
| HTMX | CDN или embed | 1.9.x |
| Alpine.js | CDN или embed | 3.x |

**Нет:** фреймворков (gin/echo/fiber), ORMs, Redis, CGO.

---

## Структура директорий

```
cms/
├── cmd/
│   └── server/
│       └── main.go                 # Точка входа: parse flags, init DB, start server
├── internal/
│   ├── config/
│   │   └── config.go               # CLI flags + env vars, валидация при старте
│   ├── db/
│   │   ├── database.go             # Открытие соединения, WAL mode, PRAGMA, migrate
│   │   ├── migrate.go              # Создание таблиц, индексов, seed admin
│   │   ├── leads.go                # CRUD: leads
│   │   ├── news.go                 # CRUD: news
│   │   ├── settings.go             # CRUD: settings (single row UPSERT)
│   │   └── sessions.go             # CRUD: sessions
│   ├── models/
│   │   ├── lead.go                 # struct Lead + статусы (new/sent/error)
│   │   ├── news.go                 # struct News
│   │   ├── settings.go             # struct Settings
│   │   └── session.go              # struct Session
│   ├── handlers/
│   │   ├── api_leads.go            # POST /api/leads (публичный)
│   │   ├── api_news.go             # GET /api/news, GET /api/news/{id}
│   │   ├── auth.go                 # GET/POST /admin/login, POST /admin/logout
│   │   ├── admin_dashboard.go      # GET /admin/
│   │   ├── admin_leads.go          # CRUD + resend + CSV export
│   │   ├── admin_news.go           # CRUD + upload
│   │   └── admin_settings.go       # GET/POST /admin/settings + test-bitrix
│   ├── middleware/
│   │   ├── auth.go                 # Проверка сессии, redirect на /admin/login
│   │   ├── ratelimit.go            # IP rate limit для /api/leads
│   │   ├── csrf.go                 # CSRF токен для форм админки
│   │   └── logger.go               # Request logging
│   ├── services/
│   │   ├── bitrix24.go             # Interface + HTTP client + worker pool
│   │   └── upload.go               # Валидация MIME, сохранение файлов
│   ├── validate/
│   │   └── lead.go                 # Валидация + санитизация полей заявки
│   └── templates/                  # Embed: //go:embed templates
│       ├── layouts/
│       │   └── base.html
│       ├── partials/
│       │   ├── nav.html
│       │   └── flash.html
│       ├── admin/
│       │   ├── login.html
│       │   ├── dashboard.html
│       │   ├── leads.html
│       │   ├── lead_detail.html
│       │   ├── news.html
│       │   ├── news_form.html
│       │   └── settings.html
│       └── errors/
│           ├── 404.html
│           └── 500.html
├── static/                         # Embed: //go:embed static
│   ├── css/
│   │   └── style.css
│   └── js/
│       ├── htmx.min.js
│       └── alpine.min.js
├── uploads/                        # Runtime: создаётся при старте, не в embed
├── go.mod
├── go.sum
└── Makefile
```

---

## Слои и их ответственность

```
HTTP Request
     │
     ▼
middleware/logger.go        — логирование каждого запроса
     │
     ▼
middleware/ratelimit.go     — только для /api/leads
     │
     ▼
middleware/auth.go          — только для /admin/*
     │
     ▼
middleware/csrf.go          — только для POST форм в /admin/*
     │
     ▼
handlers/*.go               — парсинг запроса, вызов validate/db/services
     │
     ├── validate/           — валидация + санитизация входных данных
     ├── db/                 — SQL запросы, CRUD операции
     └── services/           — бизнес-логика (Bitrix24, upload)
          │
          ▼
     models/                 — структуры данных (без логики)
          │
          ▼
     SQLite (WAL mode)
```

---

## База данных

### SQLite конфигурация (обязательно при открытии)
```sql
PRAGMA journal_mode=WAL;
PRAGMA busy_timeout=5000;
PRAGMA foreign_keys=ON;
PRAGMA synchronous=NORMAL;
```

### Таблицы

```sql
CREATE TABLE IF NOT EXISTS leads (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT NOT NULL,
    phone           TEXT NOT NULL,
    email           TEXT DEFAULT '',
    comment         TEXT DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'new',
    bitrix_response TEXT DEFAULT '',
    bitrix_sent_at  DATETIME,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_leads_status ON leads(status);
CREATE INDEX IF NOT EXISTS idx_leads_created ON leads(created_at DESC);

CREATE TABLE IF NOT EXISTS news (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    date        DATETIME NOT NULL,
    title       TEXT NOT NULL,
    image       TEXT DEFAULT '',
    announce    TEXT DEFAULT '',
    description TEXT DEFAULT '',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_news_date ON news(date DESC);

CREATE TABLE IF NOT EXISTS settings (
    id               INTEGER PRIMARY KEY DEFAULT 1,
    site_name        TEXT NOT NULL DEFAULT 'My CMS',
    admin_email      TEXT NOT NULL,
    admin_password   TEXT NOT NULL,
    bitrix24_webhook TEXT DEFAULT '',
    bitrix24_enabled INTEGER NOT NULL DEFAULT 0,
    CHECK(id = 1)    -- гарантирует единственность строки
);

CREATE TABLE IF NOT EXISTS sessions (
    id         TEXT PRIMARY KEY,              -- random 32 bytes hex
    admin_id   INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
```

---

## Ключевые паттерны

### 1. Bitrix24 Worker Pool

```
POST /api/leads
    │
    ├── sync: validate + save to SQLite → respond 200
    │
    └── async: bitrix24Queue channel (buffer=100)
              │
              └── 10 workers (goroutines, started at boot)
                        │
                        └── HTTP POST to Bitrix24 webhook
                                  │
                                  └── UPDATE leads SET status, bitrix_response, bitrix_sent_at
```

При graceful shutdown:
1. Закрыть HTTP сервер (перестать принимать новые запросы)
2. Дождаться drain канала (или таймаут 30 сек)
3. Закрыть SQLite соединение

### 2. Session Flow

```
Login POST → verify bcrypt → generate session ID (crypto/rand 32 bytes)
           → INSERT sessions (expires = now + 24h)
           → Set-Cookie: session=<id>; HttpOnly; SameSite=Lax

Request /admin/* → read cookie → SELECT session WHERE id=? AND expires_at > NOW()
                → если нет → redirect /admin/login
                → если есть → proceed
```

Session cleanup: при каждом открытии БД запускать `DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP`

### 3. Rate Limit (IP-based)

- 10 запросов в минуту с одного IP на `/api/leads`
- In-memory `sync.Map[string]*rateBucket`
- Cleanup старых записей каждые 5 минут (горутина при старте)
- При превышении: HTTP 429 + JSON error

### 4. First Run

```
main.go:
1. Открыть/создать SQLite файл
2. Запустить migrate (создать таблицы)
3. SELECT COUNT(*) FROM settings
4. Если 0:
   - Сгенерировать random password (12 символов, readable charset)
   - bcrypt hash
   - INSERT INTO settings (admin_email='admin@example.com', admin_password=hash, ...)
   - Вывести в stdout:
     ┌─────────────────────────────────────┐
     │  First run detected                  │
     │  Admin email:    admin@example.com   │
     │  Admin password: xK7mP2qR9nLs        │
     │  Change password in Settings!        │
     └─────────────────────────────────────┘
5. Стартовать сервер
```

---

## Роутинг

```go
// Public
POST   /api/leads              ← rate limited
GET    /api/news
GET    /api/news/{id}
GET    /uploads/{filename}     ← http.FileServer

// Auth
GET    /admin/login
POST   /admin/login
POST   /admin/logout

// Admin (auth required)
GET    /admin/
GET    /admin/leads
GET    /admin/leads/{id}
POST   /admin/leads/{id}/resend
DELETE /admin/leads/{id}
GET    /admin/leads/export.csv

GET    /admin/news
GET    /admin/news/new
POST   /admin/news
GET    /admin/news/{id}/edit
POST   /admin/news/{id}        ← PUT через hidden _method field (HTMX compat)
DELETE /admin/news/{id}

GET    /admin/settings
POST   /admin/settings
POST   /admin/settings/test-bitrix

// Static
GET    /static/*               ← embed.FS
```

---

## Этапы разработки (с учётом архитектурных решений)

### Этап 1: Фундамент (3-4 дня)
- [ ] `go.mod` с зависимостями (modernc/sqlite, goldmark, x/crypto)
- [ ] `internal/config` — flags + env
- [ ] `internal/db/database.go` — открытие с PRAGMA
- [ ] `internal/db/migrate.go` — все таблицы + индексы
- [ ] First run: random password generation
- [ ] Graceful shutdown каркас
- [ ] Makefile: `make build`, `make test`, `make run`

### Этап 2: Заявки (2-3 дня)
- [ ] `internal/models/lead.go`
- [ ] `internal/validate/lead.go` — валидация + санитизация
- [ ] `internal/db/leads.go` — CRUD + фильтрация
- [ ] `internal/middleware/ratelimit.go`
- [ ] `internal/handlers/api_leads.go`
- [ ] Тесты: validate, db/leads, handler/api_leads

### Этап 3: Аутентификация (1-2 дня)
- [ ] `internal/db/sessions.go`
- [ ] `internal/handlers/auth.go`
- [ ] `internal/middleware/auth.go`
- [ ] `internal/middleware/csrf.go`
- [ ] Шаблон login.html
- [ ] Тесты: auth flow, session expiry

### Этап 4: Админка заявок (2 дня)
- [ ] `internal/handlers/admin_leads.go` — list, detail, resend, delete
- [ ] CSV streaming export
- [ ] Шаблоны: leads.html, lead_detail.html
- [ ] HTMX: фильтрация, поиск без перезагрузки страницы
- [ ] Тесты: CSV export, pagination

### Этап 5: Новости (2-3 дня)
- [ ] `internal/models/news.go`, `internal/db/news.go`
- [ ] `internal/services/upload.go` — MIME validation, save
- [ ] `internal/handlers/admin_news.go` — CRUD + upload
- [ ] `internal/handlers/api_news.go`
- [ ] Шаблоны: news.html, news_form.html (с Markdown preview)
- [ ] Тесты: upload validation, markdown rendering

### Этап 6: Битрикс24 + Настройки (2 дня)
- [ ] `internal/services/bitrix24.go` — interface + impl + worker pool
- [ ] `internal/handlers/admin_settings.go`
- [ ] `internal/db/settings.go` — UPSERT
- [ ] Шаблон settings.html
- [ ] Тесты: bitrix24 mock, worker pool drain

### Этап 7: Полировка (1-2 дня)
- [ ] Dashboard с базовой статистикой
- [ ] Error pages (404, 500)
- [ ] Логирование ошибок с контекстом
- [ ] Финальный проход по тестам (цель: 70%+ на критических пакетах)
- [ ] README с инструкцией по установке и nginx конфигом

**Итого: ~14-18 дней**

---

## Нефункциональные требования (зафиксированы)

| Требование | Цель | Подход |
|---|---|---|
| Response time /api/leads | < 100ms | WAL mode, sync save, async Bitrix24 |
| Concurrent leads | 100 RPS | WAL mode + connection pool (max 1 writer) |
| Binary size | Минимальный | embed все, modernc/sqlite pure Go |
| Build | `go build ./cmd/server` без CGO | modernc/sqlite |
| Test coverage | 70%+ на db/, validate/, services/ | Table-driven tests |
