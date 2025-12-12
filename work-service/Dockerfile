FROM golang:1.21-alpine AS builder

WORKDIR /app

# Устанавливаем зависимости для сборки
RUN apk add --no-cache git ca-certificates

# Копируем файлы модулей
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o work-service .

# Финальный образ
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Копируем бинарник из builder
COPY --from=builder /app/work-service .
COPY --from=builder /app/config/config.yaml ./config/config.yaml
COPY --from=builder /app/migrations ./migrations

# Экспонируем порт
EXPOSE 8081

# Ждем готовности зависимостей и запускаем миграции
COPY --from=builder /app/scripts/wait-for-it.sh .
RUN chmod +x wait-for-it.sh

# Запускаем приложение
CMD ["./wait-for-it.sh", "postgres-work:5432", "--", "./work-service"]