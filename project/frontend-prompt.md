# Frontend Developer Prompt

> Используй этот промпт как контекст при работе с шаблонами и стилями.

---

## Проект

**LightHeadless CMS** — административная панель на Go templates + HTMX.
Нет Node.js, нет сборки, нет React. Всё embed в бинарник.

Перед началом работы прочитай:
- `project/overview.md` — что это и зачем
- `project/architecture.md` — структура шаблонов, роутинг
- `project/status.md` — что уже сделано

---

## Твоя роль

Ты работаешь с:
- HTML шаблоны в `cms/internal/templates/` (Go `html/template`)
- CSS в `cms/static/css/style.css`
- HTMX и Alpine.js для интерактивности
- Без кастомной сборки JS — только то что уже в файлах

---

## Ключевые принципы

### Стек и ограничения
- **Go html/template** — XSS-safe по умолчанию, синтаксис `{{.Field}}`, `{{range}}`, `{{template}}`
- **HTMX** — для динамики без JS: `hx-get`, `hx-post`, `hx-target`, `hx-swap`
- **Alpine.js** — для локального состояния (модалки, toggle, форма preview)
- **Никакого кастомного JS** кроме точечного Alpine.js кода
- **CSS без фреймворков** — чистый CSS, переменные, flexbox/grid
- Всё должно работать без JavaScript (базовая функциональность)

### UI принципы
- Простой и чистый интерфейс — не перегружать
- Адаптивный дизайн (desktop + mobile)
- Подтверждение перед удалением (Alpine.js `x-confirm` или `<dialog>`)
- Flash сообщения (success/error) через partial шаблон
- Пагинация — числовая, с предыдущей/следующей страницей

---

## Структура шаблонов

```
internal/templates/
├── layouts/
│   └── base.html           # <!DOCTYPE html>, head, nav, flash, footer
├── partials/
│   ├── nav.html            # Навигация с активным пунктом
│   ├── flash.html          # Сообщения success/error
│   ├── pagination.html     # Компонент пагинации
│   └── confirm_modal.html  # Модалка подтверждения удаления
├── admin/
│   ├── login.html          # Страница входа
│   ├── dashboard.html      # Главная с кратким статистикой
│   ├── leads.html          # Список заявок + фильтры + поиск
│   ├── lead_detail.html    # Детали заявки
│   ├── news.html           # Список новостей
│   ├── news_form.html      # Форма создания/редактирования
│   └── settings.html       # Страница настроек
└── errors/
    ├── 404.html
    └── 500.html
```

---

## Паттерны HTMX

### Фильтрация без перезагрузки (leads list)
```html
<form hx-get="/admin/leads" hx-target="#leads-table" hx-trigger="change, input delay:500ms">
    <select name="status">
        <option value="">Все статусы</option>
        <option value="new">Новые</option>
        <option value="sent">Отправлены</option>
        <option value="error">Ошибка</option>
    </select>
    <input type="text" name="search" placeholder="Поиск...">
</form>
<div id="leads-table">
    {{template "leads_table" .}}
</div>
```

### Удаление с подтверждением
```html
<button
    hx-delete="/admin/leads/{{.ID}}"
    hx-confirm="Удалить заявку #{{.ID}}?"
    hx-target="closest tr"
    hx-swap="outerHTML swap:0.5s">
    Удалить
</button>
```

### Повторная отправка в Битрикс24
```html
<button
    hx-post="/admin/leads/{{.ID}}/resend"
    hx-target="#lead-status"
    hx-swap="outerHTML">
    Отправить в Битрикс24
</button>
```

### Markdown preview (Alpine.js)
```html
<div x-data="{ preview: false }">
    <button @click="preview = !preview" type="button">
        <span x-text="preview ? 'Редактировать' : 'Предпросмотр'"></span>
    </button>
    <textarea x-show="!preview" name="description">{{.Description}}</textarea>
    <div x-show="preview" x-html="$el.previousElementSibling.value">
        <!-- JS preview через marked.js или серверный endpoint -->
    </div>
</div>
```

---

## Шаблон base.html (структура)

```html
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.PageTitle}} — {{.SiteName}}</title>
    <link rel="stylesheet" href="/static/css/style.css">
</head>
<body>
    {{template "nav" .}}

    {{if .Flash}}
    {{template "flash" .Flash}}
    {{end}}

    <main class="container">
        {{block "content" .}}{{end}}
    </main>

    <script src="/static/js/htmx.min.js"></script>
    <script src="/static/js/alpine.min.js" defer></script>
</body>
</html>
```

---

## CSS система

Использовать CSS переменные для единообразия:
```css
:root {
    --color-primary: #2563eb;
    --color-danger: #dc2626;
    --color-success: #16a34a;
    --color-warning: #d97706;
    --color-text: #1f2937;
    --color-bg: #f9fafb;
    --color-border: #e5e7eb;
    --border-radius: 6px;
    --shadow: 0 1px 3px rgba(0,0,0,0.1);
}
```

Компоненты: `.btn`, `.btn-primary`, `.btn-danger`, `.btn-sm`, `.card`, `.table`, `.form-group`, `.badge`, `.badge-new`, `.badge-sent`, `.badge-error`, `.alert`, `.alert-success`, `.alert-error`.

---

## Данные из Go

Handler передаёт в шаблон `struct` с полями:
```go
type PageData struct {
    SiteName  string
    PageTitle string
    Flash     *Flash      // nil если нет сообщения
    // + специфичные для страницы данные
}
type Flash struct {
    Type    string // "success" | "error"
    Message string
}
```

---

## Правила

- Проверяй что шаблоны компилируются: `go build ./cmd/server`
- Никогда не inline JS в шаблонах кроме Alpine.js директив (`x-data`, `x-show`, `@click`)
- Все формы в /admin обязаны иметь CSRF поле: `<input type="hidden" name="_csrf" value="{{.CSRFToken}}">`
- Для HTMX DELETE/PUT использовать `hx-method` или скрытое поле `<input name="_method" value="DELETE">`

---

## Обновление статуса

После изменений в шаблонах или CSS — обновить `project/status.md` со ссылкой на что изменилось.
