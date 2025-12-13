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
  - `GET /reports` (поиск; фильтры query: `work_id`, `assignment_id`, `student_id`, `status`, `plagiarism_flag`, `page`, `limit`)
  - `GET /reports/{report_id}`
  - `GET /reports/work/{work_id}`
  - `GET /reports/assignment/{assignment_id}` (аналитика по заданию)
  - `GET /reports/student/{student_id}` (аналитика по студенту)
  - `GET /reports/export?format=json|csv` (экспорт)
- **Облако слов** (analysis-service, quickchart):
  - `GET /wordcloud/work/{work_id}` (PNG)

### Архитектура

- API Gateway (`api-getway`) маршрутизирует все клиентские запросы и проксирует их в бизнес-сервисы.
- Work Service (`work-service`) хранит студентов, задания и работы, принимает загрузку работы и публикует событие `work.created` в RabbitMQ.
- File Service (`file-service`) принимает и отдаёт бинарные файлы, хранит хэши и метаданные в PostgreSQL, сами файлы — в MinIO.
- Analysis Service (`analysis-service`) читает события из очереди, тянет файл/метаданные из File Service, предыдущие работы из Work Service и сохраняет отчёты в свою БД.
- Инфраструктура: PostgreSQL на каждый сервис, RabbitMQ для событий, MinIO для файлов. Всё поднимается одной командой `docker compose up --build`.
  Если целевой микросервис недоступен, gateway возвращает `503 Service Unavailable` с JSON-ошибкой.

### Пользовательский сценарий

1. Студент вызывает `POST /works` (multipart) через Gateway: файл уходит в File Service, работа сохраняется в Work Service.
2. Work Service после сохранения публикует событие `work.created` в RabbitMQ.
3. Analysis Service читает событие, тянет хэш загруженного файла из File Service, получает предыдущие работы по тому же заданию из Work Service и запускает проверку.
4. Результат проверки сохраняется как отчёт в БД analysis-service; статус работы обновляется в Work Service.
5. Преподаватель запрашивает `GET /works/{id}/reports` (через Gateway) и получает сводку по статусу и флагу плагиата.
   Для общей аналитики по заданию используйте `GET /reports/assignment/{assignment_id}`; для списка всех отчётов по заданию — `GET /reports?assignment_id=...` (с пагинацией).

### Алгоритм определения плагиата (MVP)

1. Для новой работы берётся устойчивый хэш файла (SHA-256) и размер из File Service.
2. Из Work Service забираются все предыдущие работы по тому же `assignment_id` (без текущей) с их `file_id`; для каждой работы запрашивается хэш файла в File Service.
3. Хэши сравниваются:
   - если найдено точное совпадение (100%) с работой другого студента — ставится `plagiarism_flag = true`, в отчёт сохраняется `original_work_id`.
   - если совпадений нет — `plagiarism_flag = false`, `match_percentage = 0`.
4. В отчёт пишутся: исходный хэш, список сравнённых работ, процент совпадения и технические метаданные (время, алгоритм, порог).
5. Порог сходства задаётся в конфиге (`analysis.similarity_threshold`, по умолчанию 100 для точного совпадения хэшей). При необходимости можно снизить порог или включить глубокий анализ содержимого.

### Облако слов (10/10)

В `analysis-service` реализован эндпоинт генерации облака слов через QuickChart Word Cloud API.

- `GET /api/v1/wordcloud/work/{work_id}` возвращает `image/png`.
- параметры (query):
  - `width` (по умолчанию 800)
  - `height` (по умолчанию 600)
  - `max_words` (по умолчанию 200)
  - `min_len` (по умолчанию 2)
  - `remove_stopwords` (по умолчанию false)
  - `lang` (по умолчанию ru)

Пример:

```bash
curl -o wordcloud.png "http://localhost:8080/api/v1/wordcloud/work/<work_id>?remove_stopwords=true&lang=ru"
```

### Postman коллекция / Swagger

Готовая Postman коллекция лежит в `postman/plagiarism-checker.postman_collection.json` и демонстрирует основной функционал через API Gateway.

Важно: загрузка файлов в **Postman Web** может быть нестабильной (файл “сбрасывается” и запрос уходит без `multipart` части `file`). Для проверки рекомендуется:

- **вариант A (рекомендуется преподавателю)**: использовать **Postman Desktop App** и импортировать коллекцию
- **вариант B**: использовать консольные команды ниже (PowerShell), они воспроизводимы и не зависят от Postman

### Быстрый e2e тест (PowerShell, без Postman)

В репозитории лежат 4 тестовых файла:

- `file1.txt` и `file2.txt` — **идентичные** (в текущем MVP это будет считаться плагиатом, т.к. совпадает SHA-256).
- `плагиат11.txt` и `плагиат22.txt` — **перефраз** (в текущем MVP *не* считается плагиатом, т.к. файл отличается байт-в-байт).

Пример прогона двух файлов (сравнение второй работы с первой по тому же assignment):

```powershell
$ErrorActionPreference = "Stop"
$api = "http://localhost:8080/api/v1"

$file1 = "C:\Users\water\plagiarism-checker\file1.txt"
$file2 = "C:\Users\water\plagiarism-checker\file2.txt"

$ts = Get-Date -Format "yyyyMMdd-HHmmss"
$student1 = (Invoke-RestMethod -Method Post -Uri "$api/students" -ContentType "application/json; charset=utf-8" -Body (@{name="s1"; email="s1_$ts@example.com"} | ConvertTo-Json -Compress)).data.id
$student2 = (Invoke-RestMethod -Method Post -Uri "$api/students" -ContentType "application/json; charset=utf-8" -Body (@{name="s2"; email="s2_$ts@example.com"} | ConvertTo-Json -Compress)).data.id
$assignmentId = (Invoke-RestMethod -Method Post -Uri "$api/assignments" -ContentType "application/json; charset=utf-8" -Body (@{title="HW $ts"; description="e2e"} | ConvertTo-Json -Compress)).data.id

$work1 = ((curl.exe -s -S -X POST -F "student_id=$student1" -F "assignment_id=$assignmentId" -F "file=@$file1" "$api/works") | ConvertFrom-Json).data.id
$work2 = ((curl.exe -s -S -X POST -F "student_id=$student2" -F "assignment_id=$assignmentId" -F "file=@$file2" "$api/works") | ConvertFrom-Json).data.id

for($i=0;$i -lt 90;$i++){
  try { $r = Invoke-RestMethod -Method Get -Uri "$api/reports/work/$work2"; if($r.data.status -eq "completed"){ break } } catch {}
  Start-Sleep 1
}

Invoke-RestMethod -Method Get -Uri "$api/reports/work/$work2" | ConvertTo-Json -Depth 20
curl.exe -f -L -o .\wordcloud.png "$api/wordcloud/work/${work2}?remove_stopwords=true&lang=ru" | Out-Null
```

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


