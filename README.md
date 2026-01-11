# GoSvelteKit

Um template fullstack pronto para uso com **autenticação baseada em sessões**, combinando backend Golang com SQLite e frontend SvelteKit.

## 📋 Visão Geral

GoSvelteKit é um projeto base projetado para acelerar o desenvolvimento de aplicações web fullstack. Este template vem pré-configurado com autenticação plugável (inspirada no Lucia Auth), banco de dados SQLite e páginas de login/registro, permitindo que você pule a configuração inicial repetitiva e foque nas funcionalidades específicas do seu projeto.

## 🚀 Recursos

### Backend (Golang)

-   **Autenticação plugável** com adapters (estilo Lucia Auth)
-   Sessões armazenadas no banco de dados
-   Banco de dados SQLite com GORM
-   Estrutura modular e escalável
-   Middleware de autenticação
-   API RESTful com Gin

### Frontend (SvelteKit)

-   Páginas de autenticação prontas (login, registro, recuperação de senha)
-   Gerenciamento de estado com Svelte 5 runes (`$state`, `$derived`)
-   Layout responsivo com TailwindCSS
-   Interceptação automática de requisições com session ID

## 🛠️ Pré-requisitos

-   Go 1.21+
-   Bun (ou Node.js 18+)
-   Docker e Docker Compose (opcional)

## 🔧 Instalação e Uso

### Clonando o template

```bash
git clone https://github.com/lucas-varjao/gosveltekit.git meu-novo-projeto
cd meu-novo-projeto
```

### Usando Docker Compose (recomendado)

```bash
docker-compose up
```

### Execução manual

#### Backend

```bash
cd backend
go mod download
go run cmd/server/server.go
```

#### Frontend

```bash
cd frontend
bun install
bun run dev
```

## 📁 Estrutura do Projeto

```bash
gosveltekit/
├── backend/
│   ├── cmd/server/           # Ponto de entrada
│   └── internal/
│       ├── auth/             # Sistema de autenticação
│       │   ├── interfaces.go # UserAdapter, SessionAdapter
│       │   ├── auth_manager.go
│       │   └── adapter/gorm/ # Implementação GORM
│       ├── config/
│       ├── handlers/
│       ├── middleware/
│       ├── models/
│       ├── repository/
│       ├── router/
│       └── service/
│
└── frontend/
    └── src/
        ├── lib/
        │   ├── api/          # Cliente HTTP e auth
        │   └── stores/       # Estado (auth store)
        └── routes/
            ├── login/
            ├── register/
            └── (protected)/  # Rotas autenticadas
```

## 🔐 Autenticação

O sistema usa **autenticação baseada em sessões** com adapters plugáveis:

```go
// Interfaces que você pode implementar para qualquer banco
type UserAdapter interface {
    FindUserByIdentifier(identifier string) (*UserData, error)
    ValidateCredentials(identifier, password string) (*UserData, error)
    // ...
}

type SessionAdapter interface {
    CreateSession(userID string, expiresAt time.Time, metadata SessionMetadata) (*Session, error)
    GetSession(sessionID string) (*Session, error)
    // ...
}
```

### Resposta de Login

```json
{
    "session_id": "abc123...",
    "expires_at": "2024-02-11T12:00:00Z",
    "user": {
        "id": "1",
        "identifier": "admin",
        "email": "admin@example.com",
        "display_name": "Administrator",
        "role": "admin"
    }
}
```

## ⚙️ Configuração

Copie o arquivo `.env.example` para `.env` e ajuste as variáveis conforme necessário:

```bash
cp .env.example .env
```

## 🔄 Começando um Novo Projeto

1. Clone este repositório com um novo nome
2. Personalize o `.env` e as configurações
3. Modifique os modelos no backend conforme necessário
4. Adapte as páginas do frontend para seu caso de uso
5. Para integrar com outro banco de usuários, implemente `UserAdapter`

## 📄 Licença

Este projeto está licenciado sob a MIT License - veja o arquivo [LICENSE](LICENSE) para detalhes.

## 🤝 Contribuição

Contribuições são bem-vindas! Por favor, sinta-se à vontade para enviar um pull request.

---

Desenvolvido com ❤️ para agilizar seu fluxo de trabalho de desenvolvimento.
