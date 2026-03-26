# DevOps Developer Prompt

> Используй этот промпт при работе с инфраструктурой, сборкой и деплоем.

---

## Проект

**LightHeadless CMS** — Go приложение, единый бинарный файл.
Цель DevOps: сделать сборку и деплой максимально простыми для нетехнического пользователя.
Код в `cms/`, конфигурация сборки — `cms/Makefile`.

Перед началом работы прочитай:
- `project/overview.md` — что это и ключевые ограничения
- `project/architecture.md` — стек и структура

---

## Твоя роль

- Makefile для сборки, тестов, линтера
- Cross-compilation (Linux/Windows/macOS, amd64/arm64)
- GitHub Actions CI/CD пайплайн
- Docker образ (опционально, после MVP)
- Nginx конфигурация как reverse proxy
- Инструкции по деплою

---

## Принципы

1. **No CGO** — сборка без CGO обязательна (`CGO_ENABLED=0`), иначе нельзя кросс-компилировать
2. **Single binary** — бинарник должен включать все статические файлы (embed) и работать без установки
3. **Простота** — деплой должен быть: скопировать файл → запустить → готово
4. **Reproducible builds** — версия и дата сборки через ldflags

---

## Makefile

```makefile
APP     = cms
CMD     = ./cmd/server
VERSION = $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD   = $(shell date -u +%Y%m%d%H%M%S)
LDFLAGS = -ldflags "-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD)"

.PHONY: build test lint clean release

build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(APP) $(CMD)

test:
	CGO_ENABLED=0 go test ./... -v -race -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

clean:
	rm -f $(APP) coverage.out coverage.html

# Cross-compilation targets
build-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(APP)-linux-amd64 $(CMD)

build-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(APP)-linux-arm64 $(CMD)

build-windows-amd64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(APP)-windows-amd64.exe $(CMD)

build-darwin-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(APP)-darwin-amd64 $(CMD)

build-darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(APP)-darwin-arm64 $(CMD)

release: clean
	mkdir -p dist
	$(MAKE) build-linux-amd64
	$(MAKE) build-linux-arm64
	$(MAKE) build-darwin-amd64
	$(MAKE) build-darwin-arm64
	$(MAKE) build-windows-amd64
	cd dist && sha256sum * > checksums.txt

run:
	CGO_ENABLED=0 go run $(CMD) -port 8080

run-prod:
	./$(APP) -port 8080 -db /var/lib/cms/cms.db -upload /var/lib/cms/uploads
```

---

## GitHub Actions CI

Файл: `.github/workflows/ci.yml`

```yaml
name: CI

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
          cache: true

      - name: Test
        working-directory: cms
        run: CGO_ENABLED=0 go test ./... -race -coverprofile=coverage.out

      - name: Coverage check
        working-directory: cms
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | tr -d '%')
          echo "Coverage: $COVERAGE%"
          # Порог: 60% минимум
          awk "BEGIN { if ($COVERAGE < 60) { print \"Coverage below 60%\"; exit 1 } }"

  build:
    runs-on: ubuntu-latest
    needs: test
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
          cache: true

      - name: Build (Linux amd64)
        working-directory: cms
        run: CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /tmp/cms ./cmd/server

      - name: Verify binary runs
        run: /tmp/cms -port 9999 &
        # Просто проверяем что запускается без паники
```

Файл: `.github/workflows/release.yml`

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Build all targets
        working-directory: cms
        run: make release

      - name: Create GitHub Release
        uses: softprops/action-gh-release@v2
        with:
          files: cms/dist/*
          generate_release_notes: true
```

---

## Nginx конфигурация

Файл для пользователя: `deploy/nginx.conf`

```nginx
server {
    listen 80;
    server_name yourdomain.com;

    # Redirect HTTP to HTTPS (раскомментировать если есть SSL)
    # return 301 https://$host$request_uri;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Timeout для загрузки файлов
        proxy_read_timeout 60s;
        client_max_body_size 10M;
    }

    # Статические файлы — можно отдавать напрямую через nginx (опционально)
    # location /uploads/ {
    #     alias /var/lib/cms/uploads/;
    #     expires 7d;
    # }
}

# HTTPS (добавить после получения SSL сертификата через certbot)
# server {
#     listen 443 ssl;
#     server_name yourdomain.com;
#     ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
#     ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;
#     ...
# }
```

---

## Systemd сервис

Файл: `deploy/cms.service`

```ini
[Unit]
Description=LightHeadless CMS
After=network.target

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/var/lib/cms
ExecStart=/usr/local/bin/cms -port 8080 -db /var/lib/cms/cms.db -upload /var/lib/cms/uploads
Restart=on-failure
RestartSec=5s

# Security
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ReadWritePaths=/var/lib/cms

# Logs
StandardOutput=journal
StandardError=journal
SyslogIdentifier=cms

[Install]
WantedBy=multi-user.target
```

---

## Инструкция по деплою (для пользователя)

Файл: `deploy/DEPLOY.md`

```markdown
## Установка на Linux сервер

### 1. Скачать бинарник
wget https://github.com/user/lightcms/releases/latest/download/cms-linux-amd64
chmod +x cms-linux-amd64
sudo mv cms-linux-amd64 /usr/local/bin/cms

### 2. Создать директории
sudo mkdir -p /var/lib/cms/uploads
sudo chown -R www-data:www-data /var/lib/cms

### 3. Установить как сервис
sudo cp cms.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable cms
sudo systemctl start cms

### 4. Проверить запуск и получить пароль
sudo journalctl -u cms -n 50

# В логах будет:
# First run detected
# Admin email:    admin@example.com
# Admin password: xK7mP2qR9nLs

### 5. Настроить nginx (опционально)
sudo cp nginx.conf /etc/nginx/sites-available/cms
sudo ln -s /etc/nginx/sites-available/cms /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx

### 6. SSL (опционально)
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d yourdomain.com
```

---

## Проверки при CI

| Проверка | Команда | Порог |
|---|---|---|
| Тесты | `go test ./...` | Все зеленые |
| Race detector | `go test -race ./...` | 0 race conditions |
| Coverage | `go tool cover` | >= 60% overall |
| Build (no CGO) | `CGO_ENABLED=0 go build` | Без ошибок |
| Vet | `go vet ./...` | 0 предупреждений |

---

## Docker (после MVP)

Пока не в приоритете, но структура Dockerfile:
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /build
COPY cms/ .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o cms ./cmd/server

FROM scratch
COPY --from=builder /build/cms /cms
EXPOSE 8080
VOLUME ["/data"]
CMD ["/cms", "-db", "/data/cms.db", "-upload", "/data/uploads"]
```

---

## Обновление статуса

После настройки CI/CD или изменения деплоя — обновить `project/status.md`.
