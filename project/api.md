# API Documentation

> Последнее обновление: 2026-03-26  
> Статус: **Реализовано** ✅

---

## Base URL

```
http://localhost:8080
```

В production: `https://yourdomain.com`

---

## Публичные эндпоинты

### POST /api/leads

Приём заявки с лендинга. Rate limit: 10 запросов в минуту с одного IP.

**Content-Type:** `application/x-www-form-urlencoded` или `application/json`

**Request Body:**

| Поле | Тип | Обязательное | Ограничения |
|---|---|---|---|
| `name` | string | Да | 1-255 символов |
| `phone` | string | Да | 1-20 символов |
| `email` | string | Нет | валидный email, до 255 символов |
| `comment` | string | Нет | до 1000 символов |

**Responses:**

`200 OK` — заявка принята
```json
{
  "status": "success",
  "message": "Lead saved"
}
```

`400 Bad Request` — ошибка валидации
```json
{
  "status": "error",
  "errors": [
    {"field": "name", "message": "Name is required"},
    {"field": "email", "message": "Invalid email format"}
  ]
}
```

`429 Too Many Requests` — превышен rate limit
```json
{
  "status": "error",
  "message": "Too many requests, please try again later"
}
```

`500 Internal Server Error`
```json
{
  "status": "error",
  "message": "Internal server error"
}
```

**Пример запроса (форма):**
```html
<form action="https://cms.yourdomain.com/api/leads" method="POST">
  <input name="name" required>
  <input name="phone" required>
  <input name="email">
  <textarea name="comment"></textarea>
  <!-- Honeypot: скрыть через CSS, не через type="hidden" -->
  <input name="website" style="display:none" tabindex="-1">
  <button type="submit">Отправить</button>
</form>
```

**Пример запроса (JSON):**
```bash
curl -X POST http://localhost:8080/api/leads \
  -H "Content-Type: application/json" \
  -d '{"name":"Иван","phone":"+7 999 123-45-67","email":"ivan@example.com"}'
```

---

### GET /api/news

Список новостей с пагинацией.

**Query Parameters:**

| Параметр | Тип | Default | Описание |
|---|---|---|---|
| `page` | int | 1 | Номер страницы (с 1) |
| `limit` | int | 10 | Записей на странице (max 50) |

**Response: `200 OK`**
```json
{
  "data": [
    {
      "id": 1,
      "date": "2026-03-26T00:00:00Z",
      "title": "Заголовок новости",
      "announce": "Краткий анонс",
      "image": "/uploads/news-123.jpg",
      "created_at": "2026-03-26T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 10,
    "total": 42,
    "pages": 5
  }
}
```

**Примечания:**
- Поле `description` (Markdown) не включается в список — только в деталях
- `image` — путь относительно хоста CMS, может быть пустым `""`
- Сортировка: по `date DESC`

---

### GET /api/news/{id}

Конкретная новость с полным описанием.

**Path Parameters:**

| Параметр | Тип | Описание |
|---|---|---|
| `id` | int | ID новости |

**Response: `200 OK`**
```json
{
  "id": 1,
  "date": "2026-03-26T00:00:00Z",
  "title": "Заголовок новости",
  "announce": "Краткий анонс",
  "description": "# Заголовок\n\nПолный текст в **Markdown**",
  "description_html": "<h1>Заголовок</h1><p>Полный текст в <strong>Markdown</strong></p>",
  "image": "/uploads/news-123.jpg",
  "created_at": "2026-03-26T10:00:00Z",
  "updated_at": "2026-03-26T11:00:00Z"
}
```

**Response: `404 Not Found`**
```json
{
  "status": "error",
  "message": "News not found"
}
```

---

### GET /uploads/{filename}

Статические файлы (изображения новостей). Отдаются напрямую.

**Response:** Файл с корректным `Content-Type`

---

## Административный API

> Все эндпоинты требуют авторизации (cookie `session`).
> При отсутствии сессии — redirect на `/admin/login`.

### Аутентификация

#### POST /admin/login

**Content-Type:** `application/x-www-form-urlencoded`

| Поле | Тип | Описание |
|---|---|---|
| `email` | string | Email администратора |
| `password` | string | Пароль |
| `_csrf` | string | CSRF токен |

**Response:**
- `302 Found` → `/admin/` при успехе
- `200 OK` с формой + ошибкой при неверных данных

#### POST /admin/logout

**Response:** `302 Found` → `/admin/login`

---

### Управление заявками

#### GET /admin/leads

Список заявок.

**Query Parameters:**

| Параметр | Default | Описание |
|---|---|---|
| `status` | (все) | Фильтр: `new`, `sent`, `error` |
| `search` | (нет) | Поиск по name, phone, email |
| `page` | 1 | Страница |
| `limit` | 20 | Записей на странице |

**Response:** HTML страница или HTMX partial (если `HX-Request: true`)

---

#### GET /admin/leads/{id}

Детали заявки. **Response:** HTML страница

---

#### POST /admin/leads/{id}/resend

Повторная отправка заявки в Битрикс24.

**Response (HTMX):** HTML partial с новым статусом заявки

---

#### DELETE /admin/leads/{id}

Удаление заявки.

**Response (HTMX):** пустой ответ, строка удаляется через `hx-swap="outerHTML swap:0.5s"`

---

#### GET /admin/leads/export.csv

Экспорт всех заявок (с учётом текущего фильтра) в CSV.

**Query Parameters:** те же что у GET /admin/leads (status, search)

**Response:**
```
Content-Type: text/csv; charset=utf-8
Content-Disposition: attachment; filename="leads-2026-03-26.csv"
```

CSV поля: `id,name,phone,email,comment,status,created_at,bitrix_sent_at`

---

### Управление новостями

#### GET /admin/news — список новостей (HTML)
#### GET /admin/news/new — форма создания (HTML)
#### POST /admin/news — создание новости

**Content-Type:** `multipart/form-data`

| Поле | Тип | Обязательное |
|---|---|---|
| `date` | string (YYYY-MM-DD) | Да |
| `title` | string | Да |
| `announce` | string | Нет |
| `description` | string (Markdown) | Нет |
| `image` | file (JPEG/PNG/GIF/WebP, max 5MB) | Нет |
| `_csrf` | string | Да |

**Response:** `302 Found` → `/admin/news` при успехе, форма с ошибками при ошибке

#### GET /admin/news/{id}/edit — форма редактирования (HTML)
#### POST /admin/news/{id} — обновление (с `_method=PUT` в теле)
#### DELETE /admin/news/{id} — удаление

---

### Настройки

#### GET /admin/settings — страница настроек (HTML)

#### POST /admin/settings

| Поле | Тип | Описание |
|---|---|---|
| `site_name` | string | Название сайта |
| `admin_email` | string | Email для входа |
| `admin_password` | string | Новый пароль (пусто = не менять) |
| `admin_password_confirm` | string | Подтверждение |
| `bitrix24_webhook` | string | URL вебхука |
| `bitrix24_enabled` | checkbox (1/0) | Включить интеграцию |
| `_csrf` | string | CSRF токен |

**Response:** `302 Found` → `/admin/settings` с flash сообщением

#### POST /admin/settings/test-bitrix

Тестовая отправка в Битрикс24.

**Response (HTMX):** HTML partial с результатом теста (success/error + детали)

---

## CORS

Публичные API эндпоинты (`/api/*`) поддерживают CORS:
```
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, OPTIONS
Access-Control-Allow-Headers: Content-Type
```

Можно ограничить конкретным доменом через настройки (после MVP).

---

## Формат ошибок (публичный API)

Все ошибки возвращают JSON:
```json
{
  "status": "error",
  "message": "Human-readable error message",
  "errors": [...]  // только для 400 с полями
}
```

---

## Changelog

| Дата | Версия | Изменение |
|---|---|---|
| 2026-03-26 | 1.0 | Финальная реализация всех endpoints ✅ |
| 2026-03-26 | 0.1 | Начальная документация (планирование) |
