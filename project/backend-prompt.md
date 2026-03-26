# Backend Developer Prompt

> Используй этот промпт как контекст при старте каждой сессии разработки бекенда.

---

## Проект

**LightHeadless CMS** — легковесная Headless CMS на Go.
Единый бинарный файл, SQLite, без внешних зависимостей, без CGO.
Код находится в папке `cms/` относительно корня репозитория.

Перед началом работы прочитай:
- `project/overview.md` — что это и зачем
- `project/architecture.md` — стек, структура, паттерны, этапы
- `project/status.md` — что уже сделано, что делается, что следующее
- `project/api.md` — контракт публичного и административного API

---

## Твоя роль

Ты пишешь весь серверный Go код:
- HTTP handlers (публичный API и админка)
- Middleware (auth, rate limit, CSRF, logger)
- Слой работы с БД (db/)
- Бизнес-логика (services/)
- Валидация (validate/)
- Конфигурация (config/)
- Точка входа (cmd/server/main.go)
- Инициализация БД, миграции, first run

Фронтенд (шаблоны, CSS, JS) — в `frontend-prompt.md`.

---

## Принципы

### Обязательные
1. **No CGO** — использовать `modernc.org/sqlite`, не `mattn/go-sqlite3`
2. **WAL mode** — всегда включать при открытии SQLite: `PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;`
3. **Параметризованные запросы** — никогда не конкатенировать SQL вручную
4. **Streaming CSV** — экспорт через `sql.Rows`, не загружать всё в память
5. **Worker pool** — Bitrix24 через 10 воркеров, buffered channel 100, drain при shutdown
6. **Graceful shutdown** — перехватить SIGINT/SIGTERM, дождаться drain очереди (таймаут 30s)
7. **Random password** — при первом запуске генерировать через `crypto/rand`, не хардкодить

### Качество кода
- Обрабатывай все ошибки явно — нет `_` для ошибок кроме `defer`
- Возвращай ошибки наверх, логируй только на уровне handler
- Логируй структурированно: `[ERROR] context: message`
- Используй `context` для отмены операций при shutdown
- Таблица-driven tests для каждой функции валидации и каждого CRUD метода

### Что не делать
- Не использовать сторонние HTTP фреймворки (gin, echo, fiber, chi)
- Не использовать ORM
- Не паниковать в production коде (`panic` только при инициализации если БД не открылась)
- Не хранить секреты в коде (пароли, токены)

---

## Структура пакетов и их контракт

### `internal/config`
```go
type Config struct {
    Port       int    // -port flag, CMS_PORT env, default 8080
    DBPath     string // -db flag, CMS_DB_PATH env, default ./cms.db
    UploadPath string // -upload flag, CMS_UPLOAD_PATH env, default ./uploads
}
func Load() (*Config, error) // parse flags + env, validate
```

### `internal/db`
- `database.go`: `Open(path string) (*sql.DB, error)` — открыть с PRAGMA
- `migrate.go`: `Migrate(db *sql.DB) error` — создать таблицы/индексы, вызвать SeedAdmin если нужно
- `leads.go`: `Create`, `GetByID`, `List` (с фильтром/пагинацией), `UpdateStatus`, `Delete`, `UpdateBitrix`
- `news.go`: `Create`, `GetByID`, `List`, `Update`, `Delete`
- `settings.go`: `Get`, `Upsert`
- `sessions.go`: `Create`, `GetByID`, `Delete`, `DeleteExpired`

### `internal/validate`
```go
type LeadInput struct { Name, Phone, Email, Comment string }
type ValidationError struct { Field, Message string }
func ValidateLead(input LeadInput) []ValidationError
// Санитизация: strings.TrimSpace + html.EscapeString для текстовых полей
// Email: regexp для базовой проверки формата
// Длины: name<=255, phone<=20, email<=255, comment<=1000
```

### `internal/services/bitrix24`
```go
type Client interface {
    SendLead(ctx context.Context, lead models.Lead) error
}
type WorkerPool struct { ... }
func NewWorkerPool(client Client, workers int, queueSize int) *WorkerPool
func (p *WorkerPool) Submit(lead models.Lead)
func (p *WorkerPool) Shutdown(timeout time.Duration) // drain + close
```

### `internal/middleware`
- `auth.go`: читает cookie `session`, проверяет через `db.sessions.GetByID`, redirect на login при отсутствии
- `ratelimit.go`: 10 req/min per IP для `/api/leads`, HTTP 429 при превышении
- `csrf.go`: генерирует токен в cookie + hidden field, проверяет при POST в /admin
- `logger.go`: логирует method, path, status, duration

---

## Паттерн handler'а

```go
func (h *Handler) HandleSomething(w http.ResponseWriter, r *http.Request) {
    // 1. Parse input
    // 2. Validate (вернуть 400 с деталями при ошибке)
    // 3. Call db/service
    // 4. Handle error (500 с логом)
    // 5. Respond (JSON для API, redirect или render для admin)
}
```

Для API handlers: всегда отвечать JSON, `Content-Type: application/json`.
Для admin handlers: рендерить шаблон или HTMX partial.

---

## Тестирование

### Что тестировать обязательно
- `internal/validate` — 100% coverage, все edge cases
- `internal/db` — integration tests с `:memory:` SQLite
- `internal/services/bitrix24` — через mock `Client` interface
- `internal/middleware/ratelimit` — проверить 429 при превышении
- `internal/handlers/api_leads` — happy path + validation errors + rate limit

### Как тестировать handlers
```go
// Использовать httptest.NewRecorder() + httptest.NewRequest()
// Поднимать тестовую SQLite in-memory БД для каждого теста
// Моки только для внешних сервисов (Bitrix24)
```

### Что не нужно тестировать
- main.go (только интеграционный тест при необходимости)
- Шаблоны (тестируется вручную)

---

## Обновление статуса

После завершения каждого этапа или значимой задачи:
1. Обновить `project/status.md` — отметить выполненное, добавить заметки
2. Если изменился API контракт — обновить `project/api.md`
3. Если изменилась архитектура — обновить `project/architecture.md`

Правила обновления статусов — в `project/agent-rules.md`.
