# Инструкция для AI агента: Интеграция фронтенда с бекендом LightHeadless CMS

> Версия: 1.0  
> Дата: 2026-03-26  
> Статус: Готово к интеграции

---

## Оглавление

1. [Общая информация](#общая-информация)
2. [Архитектура бекенда](#архитектура-бекенда)
3. [Публичный API](#публичный-api)
4. [Админский UI](#админский-ui)
5. [Интеграция: Пошаговое руководство](#интеграция-пошаговое-руководство)
6. [Примеры кода](#примеры-кода)
7. [Обработка ошибок](#обработка-ошибок)
8. [Best Practices](#best-practices)

---

## Общая информация

### Стек бекенда

| Компонент | Технология |
|---|---|
| Язык | Go 1.22+ |
| HTTP сервер | `net/http` (stdlib) |
| База данных | SQLite (`modernc.org/sqlite`) |
| Шаблоны | `html/template` + HTMX + Alpine.js |
| CSS | Tailwind CSS (CDN) |
| Markdown | `goldmark` |

### Режимы работы

**Бекенд работает в двух режимах:**

1. **Headless CMS** — только API для внешнего фронтенда
2. **Встроенная админка** — готовый UI для управления

### URL структура

```
http://localhost:8080/
├── /api/              # Публичный API (JSON)
│   ├── /leads         # Приём заявок
│   └── /news          # Новости
├── /admin/            # Админка (HTML)
├── /uploads/          # Загруженные файлы
└── /static/           # Статика (JS, CSS)
```

---

## Архитектура бекенда

### Слои приложения

```
HTTP Request
     │
     ▼
middleware/logger      — логирование запросов
     │
     ▼
middleware/ratelimit   — rate limit для /api/leads (10/min)
     │
     ▼
middleware/auth        — проверка сессии для /admin/*
     │
     ▼
middleware/csrf        — CSRF токены для POST в админке
     │
     ▼
handlers/              — парсинг запроса, валидация
     │
     ├── validate/     — валидация + санитизация
     ├── db/           — CRUD операции (SQLite)
     └── services/     — бизнес-логика (Bitrix24, upload)
          │
          ▼
     models/           — структуры данных
```

### Модели данных

#### Lead (Заявка)

```go
type Lead struct {
    ID             int
    Name           string    // 1-255 символов
    Phone          string    // 1-20 символов
    Email          string    // опционально, валидация email
    Comment        string    // опционально, до 1000 символов
    Status         string    // "new" | "sent" | "error"
    BitrixResponse string    // ответ от Битрикс24
    BitrixSentAt   *time.Time
    CreatedAt      time.Time
}
```

#### News (Новости)

```go
type News struct {
    ID          int
    Date        time.Time
    Title       string    // до 255 символов
    Image       string    // путь к файлу /uploads/{filename}
    Announce    string    // до 500 символов
    Description string    // Markdown
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

---

## Публичный API

### POST /api/leads

Приём заявки с лендинга.

**Request:**

```http
POST /api/leads
Content-Type: application/json | application/x-www-form-urlencoded
```

**Body (JSON):**

```json
{
  "name": "Иван",
  "phone": "+7 999 123-45-67",
  "email": "ivan@example.com",
  "comment": "Заявка с сайта"
}
```

**Body (Form):**

```
name=Иван&phone=%2B7+999+123-45-67&email=ivan%40example.com&comment=Заявка
```

**Response 200:**

```json
{
  "status": "success",
  "message": "Lead saved"
}
```

**Response 400 (Validation Error):**

```json
{
  "status": "error",
  "errors": [
    {"field": "name", "message": "Имя обязательно для заполнения"},
    {"field": "email", "message": "Некорректный формат email"}
  ]
}
```

**Response 429 (Rate Limit):**

```json
{
  "status": "error",
  "message": "Too many requests. Please try again later.",
  "Retry-After": "60"
}
```

**CORS Headers:**

```
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, OPTIONS
Access-Control-Allow-Headers: Content-Type
```

---

### GET /api/news

Список новостей с пагинацией.

**Request:**

```http
GET /api/news?page=1&limit=10
```

**Query Parameters:**

| Параметр | Тип | Default | Описание |
|---|---|---|---|
| `page` | int | 1 | Номер страницы |
| `limit` | int | 10 | Записей на странице (max 100) |
| `search` | string | — | Поиск по заголовку/анонсу |

**Response 200:**

```json
{
  "data": [
    {
      "id": 1,
      "date": "2026-03-26",
      "title": "Заголовок новости",
      "announce": "Краткий анонс",
      "image": "/uploads/news-abc123.jpg",
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

---

### GET /api/news/{id}

Детальная новость.

**Request:**

```http
GET /api/news/1
```

**Response 200:**

```json
{
  "id": 1,
  "date": "2026-03-26",
  "title": "Заголовок новости",
  "announce": "Краткий анонс",
  "description": "# Заголовок\n\nПолный текст в **Markdown**",
  "description_html": "<h1>Заголовок</h1><p>Полный текст в <strong>Markdown</strong></p>",
  "image": "/uploads/news-abc123.jpg",
  "created_at": "2026-03-26T10:00:00Z",
  "updated_at": "2026-03-26T11:00:00Z"
}
```

**Response 404:**

```json
{
  "status": "error",
  "message": "News not found"
}
```

---

### GET /uploads/{filename}

Получение загруженного файла.

**Request:**

```http
GET /uploads/news-abc123.jpg
```

**Response:** Файл с корректным `Content-Type`

---

## Админский UI

### Готовые страницы

Бекенд предоставляет готовый UI для управления:

| URL | Описание |
|---|---|
| `/admin/login` | Вход в систему |
| `/admin/` | Dashboard |
| `/admin/leads` | Список заявок |
| `/admin/leads/{id}` | Детали заявки |
| `/admin/news` | Список новостей |
| `/admin/news/new` | Создание новости |
| `/admin/news/{id}/edit` | Редактирование новости |
| `/admin/settings` | Настройки |

### Технологии UI

- **Tailwind CSS** (CDN) — стилизация
- **HTMX** — динамические запросы без перезагрузки
- **Alpine.js** — реактивность (состояния, модальные окна)

### HTMX паттерны в админке

**Фильтрация заявок:**

```html
<form hx-get="/admin/leads"
      hx-target="#leads-table-wrap"
      hx-trigger="change, input delay:400ms">
  <select name="status">...</select>
  <input name="search">
</form>
```

**Удаление заявки:**

```html
<button hx-delete="/admin/leads/123"
        hx-target="#lead-row-123"
        hx-swap="outerHTML swap:0.5s"
        hx-confirm="Удалить заявку?">
  Удалить
</button>
```

**Предпросмотр Markdown:**

```html
<textarea name="description"
          hx-post="/admin/news/preview"
          hx-target="#md-preview"
          hx-trigger="click">
</textarea>
<div id="md-preview"></div>
```

---

## Интеграция: Пошаговое руководство

### Шаг 1: Настройка CORS (если фронтенд на другом домене)

Бекенд уже отправляет CORS заголовки для `/api/*`:

```go
// В handlers/api_leads.go и api_news.go
func setCORSHeaders(w http.ResponseWriter) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}
```

**Для production:** замените `*` на конкретный домен фронтенда.

### Шаг 2: Интеграция формы заявки

**Вариант A: Прямая отправка формы (без JS)**

```html
<form action="https://cms.yourdomain.com/api/leads" method="POST">
  <input name="name" required>
  <input name="phone" required>
  <input name="email" type="email">
  <textarea name="comment"></textarea>
  <button type="submit">Отправить</button>
</form>
```

**Вариант B: AJAX отправка (React/Vue/Vanilla JS)**

```javascript
async function submitLead(data) {
  const response = await fetch('https://cms.yourdomain.com/api/leads', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data)
  });
  
  const result = await response.json();
  
  if (response.ok) {
    console.log('Заявка отправлена:', result);
  } else {
    console.error('Ошибка:', result.errors);
  }
  
  return result;
}
```

### Шаг 3: Получение новостей

**Вариант A: Fetch API**

```javascript
async function getNews(page = 1, limit = 10) {
  const response = await fetch(
    `https://cms.yourdomain.com/api/news?page=${page}&limit=${limit}`
  );
  const data = await response.json();
  return data; // { data: [...], pagination: {...} }
}
```

**Вариант B: React Hook**

```jsx
function useNews(page, limit) {
  const [news, setNews] = useState([]);
  const [pagination, setPagination] = useState({});
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function fetchNews() {
      setLoading(true);
      const res = await fetch(
        `/api/news?page=${page}&limit=${limit}`
      );
      const data = await res.json();
      setNews(data.data);
      setPagination(data.pagination);
      setLoading(false);
    }
    fetchNews();
  }, [page, limit]);

  return { news, pagination, loading };
}
```

### Шаг 4: Отображение изображений

Бекенд отдаёт изображения по пути `/uploads/{filename}`:

```html
<img src="https://cms.yourdomain.com/uploads/news-abc123.jpg" alt="News image">
```

**Важно:** Если фронтенд на другом домене, используйте абсолютный URL.

### Шаг 5: Рендеринг Markdown

Бекенд возвращает готовый HTML в поле `description_html`:

```javascript
// React
function NewsDetail({ news }) {
  return (
    <article>
      <h1>{news.title}</h1>
      <div dangerouslySetInnerHTML={{ __html: news.description_html }} />
    </article>
  );
}
```

**Или используйте Markdown библиотеку на фронтенде:**

```bash
npm install react-markdown
```

```jsx
import ReactMarkdown from 'react-markdown';

<ReactMarkdown>{news.description}</ReactMarkdown>
```

---

## Примеры кода

### React: Форма заявки

```jsx
import { useState } from 'react';

function LeadForm() {
  const [formData, setFormData] = useState({
    name: '',
    phone: '',
    email: '',
    comment: ''
  });
  const [status, setStatus] = useState('idle'); // idle | submitting | success | error
  const [errors, setErrors] = useState({});

  const handleSubmit = async (e) => {
    e.preventDefault();
    setStatus('submitting');
    
    try {
      const res = await fetch('https://cms.yourdomain.com/api/leads', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(formData)
      });
      
      const data = await res.json();
      
      if (res.ok) {
        setStatus('success');
        setFormData({ name: '', phone: '', email: '', comment: '' });
      } else {
        setStatus('error');
        setErrors(data.errors || {});
      }
    } catch (err) {
      setStatus('error');
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      {status === 'success' && (
        <div className="success">Заявка отправлена!</div>
      )}
      
      {status === 'error' && (
        <div className="error">Ошибка отправки</div>
      )}
      
      <input
        name="name"
        value={formData.name}
        onChange={(e) => setFormData({...formData, name: e.target.value})}
        placeholder="Имя"
        required
      />
      {errors.name && <span className="error">{errors.name}</span>}
      
      <input
        name="phone"
        value={formData.phone}
        onChange={(e) => setFormData({...formData, phone: e.target.value})}
        placeholder="Телефон"
        required
      />
      {errors.phone && <span className="error">{errors.phone}</span>}
      
      <input
        name="email"
        type="email"
        value={formData.email}
        onChange={(e) => setFormData({...formData, email: e.target.value})}
        placeholder="Email"
      />
      {errors.email && <span className="error">{errors.email}</span>}
      
      <textarea
        name="comment"
        value={formData.comment}
        onChange={(e) => setFormData({...formData, comment: e.target.value})}
        placeholder="Комментарий"
      />
      
      <button type="submit" disabled={status === 'submitting'}>
        {status === 'submitting' ? 'Отправка...' : 'Отправить'}
      </button>
    </form>
  );
}
```

### Vue: Список новостей

```vue
<template>
  <div class="news-list">
    <div v-if="loading">Загрузка...</div>
    
    <div v-else>
      <article v-for="news in newsList" :key="news.id" class="news-item">
        <img v-if="news.image" :src="news.image" :alt="news.title">
        <h2>{{ news.title }}</h2>
        <p class="date">{{ formatDate(news.date) }}</p>
        <p class="announce">{{ news.announce }}</p>
        <router-link :to="`/news/${news.id}`">Читать далее</router-link>
      </article>
      
      <div class="pagination">
        <button @click="prevPage" :disabled="page === 1">Назад</button>
        <span>Страница {{ page }} из {{ pagination.pages }}</span>
        <button @click="nextPage" :disabled="page >= pagination.pages">Вперёд</button>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  data() {
    return {
      newsList: [],
      pagination: {},
      page: 1,
      limit: 10,
      loading: false
    };
  },
  created() {
    this.fetchNews();
  },
  methods: {
    async fetchNews() {
      this.loading = true;
      const res = await fetch(`/api/news?page=${this.page}&limit=${this.limit}`);
      const data = await res.json();
      this.newsList = data.data;
      this.pagination = data.pagination;
      this.loading = false;
    },
    nextPage() {
      this.page++;
      this.fetchNews();
    },
    prevPage() {
      this.page--;
      this.fetchNews();
    },
    formatDate(dateStr) {
      return new Date(dateStr).toLocaleDateString('ru-RU');
    }
  }
};
</script>
```

### Next.js: Статическая генерация новостей

```jsx
// pages/news/index.js
export default function NewsPage({ newsList, pagination }) {
  return (
    <div>
      <h1>Новости</h1>
      {newsList.map(news => (
        <article key={news.id}>
          <h2>{news.title}</h2>
          <p>{news.announce}</p>
        </article>
      ))}
    </div>
  );
}

export async function getStaticProps() {
  const res = await fetch('https://cms.yourdomain.com/api/news?limit=100');
  const data = await res.json();
  
  return {
    props: {
      newsList: data.data,
      pagination: data.pagination
    },
    revalidate: 60 // ISR: перестраивать каждые 60 секунд
  };
}
```

### Next.js: Детальная страница новости

```jsx
// pages/news/[id].js
export default function NewsDetail({ news }) {
  return (
    <article>
      <h1>{news.title}</h1>
      {news.image && <img src={news.image} alt={news.title} />}
      <div dangerouslySetInnerHTML={{ __html: news.description_html }} />
    </article>
  );
}

export async function getStaticPaths() {
  const res = await fetch('https://cms.yourdomain.com/api/news');
  const data = await res.json();
  
  const paths = data.data.map(news => ({
    params: { id: news.id.toString() }
  }));
  
  return { paths, fallback: 'blocking' };
}

export async function getStaticProps({ params }) {
  const res = await fetch(
    `https://cms.yourdomain.com/api/news/${params.id}`
  );
  const news = await res.json();
  
  return { props: { news }, revalidate: 60 };
}
```

---

## Обработка ошибок

### Типы ошибок API

| Код | Описание | Действие |
|---|---|---|
| 400 | Ошибка валидации | Показать ошибки по полям |
| 404 | Ресурс не найден | Показать страницу 404 |
| 429 | Rate limit exceeded | Показать «Попробуйте позже» |
| 500 | Внутренняя ошибка | Показать «Ошибка сервера» |

### Пример обработки (JavaScript)

```javascript
async function apiCall(url, options = {}) {
  try {
    const response = await fetch(url, options);
    const data = await response.json();
    
    if (!response.ok) {
      switch (response.status) {
        case 400:
          throw new ValidationError(data.errors);
        case 404:
          throw new NotFoundError(data.message);
        case 429:
          throw new RateLimitError(data.message);
        case 500:
          throw new ServerError(data.message);
        default:
          throw new ApiError(data.message);
      }
    }
    
    return data;
  } catch (error) {
    if (error instanceof TypeError) {
      // Network error
      throw new NetworkError('Нет соединения с сервером');
    }
    throw error;
  }
}

// Классы ошибок
class ApiError extends Error {
  constructor(message) {
    super(message);
    this.name = 'ApiError';
  }
}

class ValidationError extends ApiError {
  constructor(errors) {
    super('Ошибка валидации');
    this.name = 'ValidationError';
    this.errors = errors;
  }
}

class NotFoundError extends ApiError {
  constructor(message) {
    super(message);
    this.name = 'NotFoundError';
  }
}

class RateLimitError extends ApiError {
  constructor(message) {
    super(message);
    this.name = 'RateLimitError';
  }
}

class ServerError extends ApiError {
  constructor(message) {
    super(message);
    this.name = 'ServerError';
  }
}

class NetworkError extends ApiError {
  constructor(message) {
    super(message);
    this.name = 'NetworkError';
  }
}
```

---

## Best Practices

### 1. Rate Limiting

API `/api/leads` имеет ограничение **10 запросов в минуту** с одного IP.

**Рекомендации:**
- Debounce ввод пользователя (400ms)
- Отключайте кнопку отправки во время запроса
- Показывайте понятное сообщение при 429 ошибке

### 2. Валидация на фронтенде

Дублируйте валидацию бекенда на фронтенде:

```javascript
const validateLead = (data) => {
  const errors = {};
  
  if (!data.name || data.name.trim() === '') {
    errors.name = 'Имя обязательно';
  } else if (data.name.length > 255) {
    errors.name = 'Имя не должно превышать 255 символов';
  }
  
  if (!data.phone || data.phone.trim() === '') {
    errors.phone = 'Телефон обязателен';
  } else if (data.phone.length > 20) {
    errors.phone = 'Телефон не должен превышать 20 символов';
  }
  
  if (data.email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(data.email)) {
    errors.email = 'Некорректный email';
  }
  
  if (data.comment && data.comment.length > 1000) {
    errors.comment = 'Комментарий не должен превышать 1000 символов';
  }
  
  return errors;
};
```

### 3. Оптимизация изображений

Бекенд сохраняет изображения без сжатия.

**Рекомендации:**
- Сжимайте изображения на фронтенде перед отправкой
- Используйте `sharp` (Node.js) или браузерный Canvas
- Максимальный размер: 1920x1080 для десктопа

### 4. Кэширование новостей

Новости редко меняются — кэшируйте их:

```javascript
// React Query
const { data } = useQuery(
  ['news', page],
  () => fetchNews(page),
  { staleTime: 5 * 60 * 1000 } // 5 минут
);

// SWR
const { data } = useSWR(
  `/api/news?page=${page}`,
  fetcher,
  { dedupingInterval: 60000 } // 1 минута
);
```

### 5. SSR/SSG для новостей

Для SEO используйте серверный рендеринг:

- **Next.js:** `getStaticProps` + ISR
- **Nuxt:** `useFetch` + `generate`
- **Remix:** `loader` функции

### 6. Honeypot для форм

Добавьте скрытое поле для защиты от спама:

```html
<input name="website" style="display:none" tabindex="-1" autocomplete="off">
```

Бекенд отклоняет заявки с заполненным полем `website`.

### 7. HTTPS в production

Всегда используйте HTTPS для передачи данных:

```javascript
// Принудительный редирект на HTTPS (Node.js middleware)
app.use((req, res, next) => {
  if (req.headers['x-forwarded-proto'] !== 'https') {
    return res.redirect(`https://${req.host}${req.url}`);
  }
  next();
});
```

---

## Чеклист интеграции

### Фронтенд

- [ ] Форма отправки заявок работает
- [ ] Валидация на фронтенде реализована
- [ ] Обработка ошибок API настроена
- [ ] Список новостей отображается
- [ ] Детальная страница новости работает
- [ ] Изображения загружаются корректно
- [ ] Markdown рендерится (HTML или библиотека)
- [ ] Пагинация новостей работает
- [ ] CORS настроен (если домены разные)
- [ ] HTTPS включён в production

### Бекенд

- [ ] CMS запущена и доступна
- [ ] База данных создана
- [ ] Пароль администратора изменён
- [ ] Битрикс24 настроен (если нужно)
- [ ] Резервное копирование БД настроено
- [ ] Логирование включено
- [ ] Rate limiting работает
- [ ] Мониторинг настроен

---

## Поддержка

**Документация:**
- `README.md` — общая информация
- `project/api.md` — полная API документация
- `instruction/USER_GUIDE.md` — руководство пользователя

**Отладка:**
- Проверьте логи бекенда в консоли
- Используйте `CMS_DEBUG=1` для отладки
- Проверьте CORS заголовки в Network tab DevTools

---

## Примеры интеграции

### Полная интеграция с React

```bash
# Структура проекта
my-app/
├── src/
│   ├── components/
│   │   ├── LeadForm.jsx
│   │   ├── NewsList.jsx
│   │   └── NewsDetail.jsx
│   ├── api/
│   │   └── cms.js
│   └── App.jsx
└── package.json
```

**src/api/cms.js:**

```javascript
const CMS_URL = 'https://cms.yourdomain.com';

export const cmsApi = {
  async submitLead(data) {
    const res = await fetch(`${CMS_URL}/api/leads`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    });
    return res.json();
  },
  
  async getNews(page = 1, limit = 10) {
    const res = await fetch(
      `${CMS_URL}/api/news?page=${page}&limit=${limit}`
    );
    return res.json();
  },
  
  async getNewsById(id) {
    const res = await fetch(`${CMS_URL}/api/news/${id}`);
    return res.json();
  }
};
```

**src/components/LeadForm.jsx:**

```jsx
import { useState } from 'react';
import { cmsApi } from '../api/cms';

export function LeadForm() {
  const [status, setStatus] = useState('idle');
  const [formData, setFormData] = useState({
    name: '', phone: '', email: '', comment: ''
  });

  const handleSubmit = async (e) => {
    e.preventDefault();
    setStatus('submitting');
    
    try {
      await cmsApi.submitLead(formData);
      setStatus('success');
      setFormData({ name: '', phone: '', email: '', comment: '' });
    } catch (error) {
      setStatus('error');
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      {/* ... поля формы ... */}
    </form>
  );
}
```

---

**Готово!** Теперь вы можете интегрировать фронтенд с бекендом LightHeadless CMS.
