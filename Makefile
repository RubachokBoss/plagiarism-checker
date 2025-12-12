.PHONY: up down build rebuild logs ps clean

up:
	docker compose up --build -d

down:
	docker compose down

build:
	docker compose build

rebuild:
	docker compose build --no-cache

logs:
	docker compose logs -f --tail=200

ps:
	docker compose ps

clean:
	docker compose down -v --remove-orphans


