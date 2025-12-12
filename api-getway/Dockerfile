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
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o api-gateway .

# Финальный образ
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

# Копируем бинарник из builder
COPY --from=builder /app/api-gateway .
COPY --from=builder /app/config/config.yaml ./config/config.yaml

# Экспонируем порт
EXPOSE 8080

# Запускаем приложение
CMD ["./api-gateway"]