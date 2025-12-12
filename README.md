## Plagiarism Checker (microservices, Go + PostgreSQL + RabbitMQ + MinIO)

### Запуск (Docker)

Требуется установленный `docker` + `docker compose`.

```bash
docker compose up --build -d
```

Проверка:

- **API Gateway**: `http://localhost:8080/health`
- **RabbitMQ UI**: `http://localhost:15672` (логин/пароль по умолчанию: `guest` / `guest`)
- **MinIO Console**: `http://localhost:9001` (по умолчанию: `minioadmin` / `minioadmin`)

Чтобы остановить и удалить volumes:

```bash
docker compose down -v
```

### REST API (через Gateway)

Базовый URL: `http://localhost:8080/api/v1`

- **Работы**:
  - `POST /works` (JSON) — создать работу
  - `POST /works` (multipart/form-data) — загрузить файл + создать работу
  - `GET /works/{id}`
  - `GET /works/{id}/reports`
  - `PUT /works/{id}/status`
- **Задания**:
  - `POST /assignments`
  - `GET /assignments`
  - `GET /assignments/{id}`
  - `GET /assignments/{id}/works`
- **Студенты**:
  - `POST /students`
  - `GET /students`
  - `GET /students/{id}`
  - `GET /students/{id}/works`
- **Файлы**:
  - `POST /files/upload`
  - `GET /files/{id}`
  - `GET /files/{id}/info`
  - `GET /files/{id}/url`
  - `DELETE /files/{id}`
- **Отчёты** (analysis-service):
  - `GET /reports/{report_id}`
  - `GET /reports/work/{work_id}`

### Переменные окружения (опционально)

`docker-compose.yml` содержит dev-дефолты. Если нужно переопределить — задайте переменные окружения перед запуском, например:

```bash
set WORK_DB_PASSWORD=work_password
docker compose up --build -d
```

Также можно использовать шаблон `env.example`:

- Вариант 1 (рекомендуется): скопируйте его в `.env` (файл `.env` в репозитории игнорируется) и отредактируйте значения под себя.
- Вариант 2: запуск с явным env-файлом:

```bash
docker compose --env-file env.example up --build -d
```


