.PHONY: build run test clean docker-build docker-run help

# Переменные
BINARY_NAME=api-gateway
DOCKER_IMAGE=api-gateway:latest

help:
	@echo "Доступные команды:"
	@echo "  build        - Собрать приложение"
	@echo "  run          - Запустить приложение локально"
	@echo "  test         - Запустить тесты"
	@echo "  clean        - Очистить скомпилированные файлы"
	@echo "  docker-build - Собрать Docker образ"
	@echo "  docker-run   - Запустить Docker контейнер"
	@echo "  lint         - Проверить код линтером"

build:
	@echo "Сборка приложения..."
	go build -o $(BINARY_NAME) main.go

run:
	@echo "Запуск приложения..."
	go run main.go

test:
	@echo "Запуск тестов..."
	go test ./... -v

clean:
	@echo "Очистка..."
	rm -f $(BINARY_NAME)
	rm -rf dist/
	go clean

docker-build:
	@echo "Сборка Docker образа..."
	docker build -t $(DOCKER_IMAGE) .

docker-run:
	@echo "Запуск Docker контейнера..."
	docker run -p 8080:8080 --env-file .env $(DOCKER_IMAGE)

lint:
	@echo "Проверка кода..."
	golangci-lint run ./...