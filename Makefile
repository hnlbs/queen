.PHONY: help lint test test-short build clean install fmt vet staticcheck golangci-lint tidy

# Цвета для вывода
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
RESET  := $(shell tput -Txterm sgr0)

help: ## Показать справку
	@echo 'Доступные команды:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  ${GREEN}%-18s${RESET} %s\n", $$1, $$2}'

##@ Разработка

fmt: ## Форматирование кода
	@echo "${YELLOW}Форматирование кода...${RESET}"
	@gofmt -s -w .
	@if command -v goimports > /dev/null 2>&1; then \
		goimports -w .; \
	else \
		echo "${YELLOW}⚠ goimports не установлен (запустите: make deps)${RESET}"; \
	fi
	@echo "${GREEN}✓ Код отформатирован${RESET}"

vet: ## go vet проверка
	@echo "${YELLOW}Запуск go vet...${RESET}"
	@go vet ./...
	@echo "${GREEN}✓ go vet пройден${RESET}"

staticcheck: ## staticcheck проверка
	@echo "${YELLOW}Запуск staticcheck...${RESET}"
	@if command -v staticcheck > /dev/null 2>&1; then \
		staticcheck ./...; \
		echo "${GREEN}✓ staticcheck пройден${RESET}"; \
	else \
		echo "${YELLOW}⚠ staticcheck не установлен (запустите: make deps)${RESET}"; \
	fi

golangci-lint: ## Запуск golangci-lint
	@echo "${YELLOW}Запуск golangci-lint...${RESET}"
	@if command -v golangci-lint > /dev/null 2>&1; then \
		golangci-lint run; \
		echo "${GREEN}✓ golangci-lint пройден${RESET}"; \
	else \
		echo "${YELLOW}⚠ golangci-lint не установлен (запустите: make deps)${RESET}"; \
	fi

lint: fmt vet staticcheck golangci-lint ## Полная проверка кода (fmt + vet + staticcheck + golangci-lint)
	@echo "${GREEN}✓ Все проверки пройдены${RESET}"

##@ Тестирование

test: ## Запуск всех тестов
	@echo "${YELLOW}Запуск тестов...${RESET}"
	@go test -v -race -coverprofile=coverage.out ./...
	@echo "${GREEN}✓ Тесты пройдены${RESET}"

test-short: ## Запуск коротких тестов
	@echo "${YELLOW}Запуск коротких тестов...${RESET}"
	@go test -short -v ./...
	@echo "${GREEN}✓ Короткие тесты пройдены${RESET}"

test-coverage: test ## Показать покрытие тестами
	@go tool cover -html=coverage.out

bench: ## Запуск бенчмарков
	@echo "${YELLOW}Запуск бенчмарков...${RESET}"
	@go test -bench=. -benchmem ./...

##@ Сборка

build: ## Собрать проект
	@echo "${YELLOW}Сборка проекта...${RESET}"
	@go build -v ./...
	@echo "${GREEN}✓ Проект собран${RESET}"

build-cli: ## Собрать CLI бинарник
	@echo "${YELLOW}Сборка CLI...${RESET}"
	@go build -o bin/queen ./cmd/queen
	@echo "${GREEN}✓ CLI собран: bin/queen${RESET}"

install: ## Установить queen CLI
	@echo "${YELLOW}Установка queen CLI...${RESET}"
	@go install ./cmd/queen
	@echo "${GREEN}✓ queen установлен${RESET}"

##@ Зависимости

tidy: ## Обновить go.mod и go.sum
	@echo "${YELLOW}Обновление зависимостей...${RESET}"
	@go mod tidy
	@echo "${GREEN}✓ Зависимости обновлены${RESET}"

deps: ## Установить зависимости для разработки
	@echo "${YELLOW}Установка инструментов разработки...${RESET}"
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install honnef.co/go/tools/cmd/staticcheck@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "${GREEN}✓ Инструменты установлены${RESET}"

##@ Очистка

clean: ## Очистить временные файлы
	@echo "${YELLOW}Очистка...${RESET}"
	@rm -f coverage.out
	@rm -rf bin/
	@go clean -cache -testcache
	@echo "${GREEN}✓ Очистка завершена${RESET}"

##@ Pre-commit

pre-commit: lint test-short ## Проверки перед коммитом
	@echo "${GREEN}✓ Готово к коммиту${RESET}"

##@ CI/CD

ci: lint test ## CI pipeline (lint + полные тесты)
	@echo "${GREEN}✓ CI проверки пройдены${RESET}"

##@ Документация

docs-serve: ## Запустить локальный сервер документации
	@echo "${YELLOW}Запуск сервера документации...${RESET}"
	@cd docs && zencial serve

docs-build: ## Собрать документацию
	@echo "${YELLOW}Сборка документации...${RESET}"
	@cd docs && zencial build

##@ Примеры

example-tui: ## Запустить TUI demo
	@echo "${YELLOW}Запуск TUI demo...${RESET}"
	@cd examples/tui-demo && go run . tui --driver sqlite --dsn ./test.db

##@ Разное

version: ## Показать версию Go
	@go version

env: ## Показать Go окружение
	@go env

.DEFAULT_GOAL := help
