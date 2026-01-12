# Makefile para GoSvelteKit Backend

# Variáveis
BINARY_NAME=server
BINARY_PATH=backend/cmd/server
BACKEND_DIR=backend
BUILD_DIR=backend/bin
COVERAGE_DIR=backend/coverage

# Cores para output
GREEN=\033[0;32m
YELLOW=\033[0;33m
RED=\033[0;31m
NC=\033[0m # No Color

.PHONY: help build run test test-coverage clean install format vet lint mod-tidy run-dev

# Comando padrão
.DEFAULT_GOAL := help

help: ## Mostra esta mensagem de ajuda
	@echo -e "$(GREEN)Comandos disponíveis:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-20s$(NC) %s\n", $$1, $$2}'

build: ## Compila o servidor
	@echo -e "$(GREEN)Compilando servidor...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@cd $(BACKEND_DIR) && go build -o bin/$(BINARY_NAME) ./cmd/server
	@echo -e "$(GREEN)Build concluído: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

run: build ## Compila e executa o servidor
	@echo -e "$(GREEN)Executando servidor...$(NC)"
	@cd $(BACKEND_DIR) && ./bin/$(BINARY_NAME)

run-dev: ## Executa o servidor em modo desenvolvimento (sem build)
	@echo -e "$(GREEN)Executando servidor em modo desenvolvimento...$(NC)"
	@cd $(BACKEND_DIR) && go run ./cmd/server

test: ## Executa todos os testes
	@echo -e "$(GREEN)Executando testes...$(NC)"
	@cd $(BACKEND_DIR) && go test -v ./...

test-short: ## Executa apenas testes rápidos (sem integração)
	@echo -e "$(GREEN)Executando testes rápidos...$(NC)"
	@cd $(BACKEND_DIR) && go test -short -v ./...

test-coverage: ## Executa testes com cobertura
	@echo -e "$(GREEN)Executando testes com cobertura...$(NC)"
	@mkdir -p $(COVERAGE_DIR)
	@cd $(BACKEND_DIR) && go test -coverprofile=coverage/coverage.out ./...
	@cd $(BACKEND_DIR) && go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo -e "$(GREEN)Cobertura gerada em $(COVERAGE_DIR)/coverage.html$(NC)"

test-integration: ## Executa apenas testes de integração
	@echo -e "$(GREEN)Executando testes de integração...$(NC)"
	@cd $(BACKEND_DIR) && go test -v ./internal/tests/integration/...

clean: ## Remove arquivos compilados e temporários
	@echo -e "$(YELLOW)Limpando arquivos...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -rf $(COVERAGE_DIR)
	@cd $(BACKEND_DIR) && go clean
	@echo -e "$(GREEN)Limpeza concluída$(NC)"

install: ## Instala dependências
	@echo -e "$(GREEN)Instalando dependências...$(NC)"
	@cd $(BACKEND_DIR) && go mod download
	@echo -e "$(GREEN)Dependências instaladas$(NC)"

mod-tidy: ## Limpa e atualiza go.mod
	@echo -e "$(GREEN)Limpando go.mod...$(NC)"
	@cd $(BACKEND_DIR) && go mod tidy
	@echo -e "$(GREEN)go.mod atualizado$(NC)"

format: ## Formata o código com gofmt
	@echo -e "$(GREEN)Formatando código...$(NC)"
	@cd $(BACKEND_DIR) && gofmt -s -w .
	@echo -e "$(GREEN)Código formatado$(NC)"

vet: ## Verifica o código com go vet
	@echo -e "$(GREEN)Verificando código com go vet...$(NC)"
	@cd $(BACKEND_DIR) && go vet ./...
	@echo -e "$(GREEN)Verificação concluída$(NC)"

lint: vet ## Executa verificações de código (vet)
	@echo -e "$(GREEN)Verificações de código concluídas$(NC)"

check: format vet test ## Formata, verifica e testa o código
	@echo -e "$(GREEN)Verificações concluídas$(NC)"

# Comando para verificar se o Go está instalado
check-go:
	@which go > /dev/null || (echo -e "$(RED)Go não está instalado$(NC)" && exit 1)

# Comando para verificar versão do Go
version:
	@echo -e "$(GREEN)Versão do Go:$(NC)"
	@go version
