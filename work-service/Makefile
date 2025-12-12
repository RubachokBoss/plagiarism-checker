.PHONY: build run test clean docker-build docker-run migrate-up migrate-down help

# Переменные
BINARY_NAME=work-service
DOCKER_IMAGE=work-service:latest
DB_CONTAINER=postgres-work

help:
	@echo "Доступные команды:"
	@echo "  build        - Собрать приложение"
	@echo "  run          - Запустить приложение локально"
	@echo "  test         - Запустить тесты"
	@echo "  clean        - Очистить скомпилированные файлы"
	@echo "  docker-build - Собрать Docker образ"
	@echo "  docker-run   - Запустить Docker контейнер"
	@echo "  migrate-up   - Применить миграции"
	@echo "  migrate-down - Откатить миграции"
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
	docker run -p 8081:8081 --env-file .env --network plagiarism-network $(DOCKER_IMAGE)

migrate-up:
	@echo "Применение миграций..."
	go run main.go migrate -direction up

migrate-down:
	@echo "Откат миграций..."
	go run main.go migrate -direction down

lint:
	@echo "Проверка кода..."
	golangci-lint run ./...