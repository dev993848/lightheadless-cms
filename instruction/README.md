# LightHeadless CMS Instructions / Инструкции

> Version / Версия: 1.0  
> Date / Дата: 2026-03-26

---

## 📚 Documentation / Документация

### For Users / Для пользователей

**[USER_GUIDE.md](./USER_GUIDE.md)** — Complete user guide / Полное руководство пользователя

**Contents / Содержание:**
- Quick start and CMS launch / Быстрый старт и запуск CMS
- Admin panel login and password change / Вход в админку и смена пароля
- Lead management (view, filters, export) / Управление заявками (просмотр, фильтры, экспорт)
- News management (create, edit, Markdown) / Управление новостями (создание, редактирование, Markdown)
- Bitrix24 integration setup / Настройка интеграции с Битрикс24
- Website form integration / Интеграция формы с сайтом
- FAQ / Частые вопросы

**Target Audience:** Website owners, marketers, CMS administrators

**Целевая аудитория:** владельцы сайтов, маркетологи, администраторы CMS

---

### For AI Agents and Developers / Для AI агентов и разработчиков

**[AI_INTEGRATION_GUIDE.md](./AI_INTEGRATION_GUIDE.md)** — Frontend integration guide / Руководство по интеграции фронтенда

**Contents / Содержание:**
- Backend architecture (layers, data models) / Архитектура бекенда (слои, модели данных)
- Public API (endpoints, requests, responses) / Публичный API (эндпоинты, запросы, ответы)
- Admin UI (HTMX patterns, Alpine.js) / Админский UI (HTMX паттерны, Alpine.js)
- Step-by-step integration guide / Пошаговое руководство по интеграции
- Code examples (React, Vue, Next.js) / Примеры кода (React, Vue, Next.js)
- Error handling (classes, examples) / Обработка ошибок (классы, примеры)
- Best Practices / Лучшие практики
- Integration checklist / Чеклист интеграции

**Target Audience:** AI agents, frontend developers, integrators

**Целевая аудитория:** AI агенты, фронтенд-разработчики, интеграторы

---

## 📂 File Structure / Структура файлов

```
instruction/
├── README.md                   # This file / Этот файл (навигация)
├── USER_GUIDE.md               # For users / Для пользователей
└── AI_INTEGRATION_GUIDE.md     # For AI agents and developers / Для AI и разработчиков
```

---

## 🚀 Quick Start / Быстрый старт

### 1. Start CMS / Запуск CMS

```bash
cd cms
go run ./cmd/server
```

### 2. Login to Admin / Вход в админку

Open / Откройте: **http://localhost:8080/admin/**

- **Email:** `admin@example.com`
- **Password:** shown in console on first run / выводится в консоль при первом запуске

### 3. Integrate with Website / Интеграция с сайтом

Add form to your website / Добавьте форму на сайт:

```html
<form action="http://localhost:8080/api/leads" method="POST">
  <input name="name" required>
  <input name="phone" required>
  <input name="email" type="email">
  <textarea name="comment"></textarea>
  <button type="submit">Submit</button>
</form>
```

---

## 📋 Checklist / Чеклист

### For Users / Для пользователей

- [ ] CMS started / CMS запущена
- [ ] Admin password changed / Пароль администратора изменён
- [ ] Form placed on website / Форма на сайте размещена
- [ ] Bitrix24 configured (if needed) / Битрикс24 настроен (если нужно)
- [ ] Database backup configured / Резервное копирование БД настроено

### For Developers / Для разработчиков

- [ ] API documentation reviewed / API документация изучена
- [ ] CORS configured (if different domains) / CORS настроен (если домены разные)
- [ ] Frontend validation implemented / Валидация на фронтенде реализована
- [ ] API error handling configured / Обработка ошибок API настроена
- [ ] HTTPS enabled in production / HTTPS включён в production

---

## 🎯 Next Steps / Следующие шаги

**For Users / Пользователям:**
1. Read [USER_GUIDE.md](./USER_GUIDE.md) / Прочитать руководство
2. Login to admin panel / Войти в админку
3. Create first news / Создать первую новость
4. Test lead form / Протестировать форму заявки

**For Developers / Разработчикам:**
1. Read [AI_INTEGRATION_GUIDE.md](./AI_INTEGRATION_GUIDE.md) / Прочитать руководство по интеграции
2. Study API endpoints / Изучить API endpoints
3. Choose framework (React/Vue/Next.js) / Выбрать фреймворк
4. Integrate lead form / Интегрировать форму заявки
5. Integrate news list / Интегрировать список новостей

---

## 📞 Support / Поддержка

**Documentation / Документация:**
- `instruction/USER_GUIDE.md` — User guide / Руководство пользователя
- `instruction/AI_INTEGRATION_GUIDE.md` — Frontend integration / Интеграция фронтенда
- `README.md` — Project overview / Общая информация о проекте

**Debugging / Отладка:**
- Check CMS console logs / Проверьте логи CMS в консоли
- Use `CMS_DEBUG=1` for debugging / Используйте для отладки
- Check CORS headers in DevTools / Проверьте CORS заголовки в DevTools

---

**LightHeadless CMS** — Lightweight CMS for lead collection and news management.

**LightHeadless CMS** — Легковесная CMS для сбора заявок и управления новостями.
