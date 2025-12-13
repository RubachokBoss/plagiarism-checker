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

### Архитектура

- API Gateway (`api-getway`) маршрутизирует все клиентские запросы и проксирует их в бизнес-сервисы.
- Work Service (`work-service`) хранит студентов, задания и работы, принимает загрузку работы и публикует событие `work.created` в RabbitMQ.
- File Service (`file-service`) принимает и отдаёт бинарные файлы, хранит хэши и метаданные в PostgreSQL, сами файлы — в MinIO.
- Analysis Service (`analysis-service`) читает события из очереди, тянет файл/метаданные из File Service, предыдущие работы из Work Service и сохраняет отчёты в свою БД.
- Инфраструктура: PostgreSQL на каждый сервис, RabbitMQ для событий, MinIO для файлов. Всё поднимается одной командой `docker compose up --build`.

### Пользовательский сценарий

1. Студент вызывает `POST /works` (multipart) через Gateway: файл уходит в File Service, работа сохраняется в Work Service.
2. Work Service после сохранения публикует событие `work.created` в RabbitMQ.
3. Analysis Service читает событие, тянет хэш загруженного файла из File Service, получает предыдущие работы по тому же заданию из Work Service и запускает проверку.
4. Результат проверки сохраняется как отчёт в БД analysis-service; статус работы обновляется в Work Service.
5. Преподаватель запрашивает `GET /works/{id}/reports` (через Gateway) и получает сводку по статусу и флагу плагиата.

### Алгоритм определения плагиата (MVP)

1. Для новой работы берётся устойчивый хэш файла (SHA-256) и размер из File Service.
2. Из Work Service забираются все предыдущие работы по тому же `assignment_id` (без текущей) с их `file_id`; для каждой работы запрашивается хэш файла в File Service.
3. Хэши сравниваются:
   - если найдено точное совпадение (100%) с работой другого студента — ставится `plagiarism_flag = true`, в отчёт сохраняется `original_work_id`.
   - если совпадений нет — `plagiarism_flag = false`, `match_percentage = 0`.
4. В отчёт пишутся: исходный хэш, список сравнённых работ, процент совпадения и технические метаданные (время, алгоритм, порог).
5. Порог сходства задаётся в конфиге (`analysis.similarity_threshold`, по умолчанию 100 для точного совпадения хэшей). При необходимости можно снизить порог или включить глубокий анализ содержимого.

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


