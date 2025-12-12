FROM golang:1.24-alpine AS builder

WORKDIR /app

# Устанавливаем зависимости для сборки
RUN apk add --no-cache git ca-certificates

# Копируем файлы модулей
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o analysis-service .

# Финальный образ
FROM alpine:latest

RUN apk --no-cache add ca-certificates netcat-openbsd wget

WORKDIR /root/

# Копируем бинарник из builder
COPY --from=builder /app/analysis-service .
COPY --from=builder /app/config/config.yaml ./config/config.yaml
COPY --from=builder /app/migrations ./migrations

# Экспонируем порт
EXPOSE 8083

# Ждем готовности зависимостей
COPY --from=builder /app/scripts/wait-for-it.sh .
RUN sed -i 's/\r$//' wait-for-it.sh && chmod +x wait-for-it.sh

# Запускаем приложение
CMD ["./wait-for-it.sh", "postgres-analysis:5432", "--", "./wait-for-it.sh", "rabbitmq:5672", "--", "./analysis-service"]