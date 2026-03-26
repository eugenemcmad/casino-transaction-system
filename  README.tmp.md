# Altenar - Technical Interview Test
Casino Transaction Management System
Overview:
You are tasked with building a simple transaction management system for a casino. The system will track user transactions related to their bets and wins. The transactions will be processed asynchronously via a message system, and the data will be stored in a relational database. You will also need to expose an API to allow querying of transaction data.
Вам поручено создать простую систему управления транзакциями для казино. Система будет отслеживать транзакции пользователей, связанные с их ставками и выигрышами. Транзакции будут обрабатываться асинхронно через систему обмена сообщениями, а данные будут храниться в реляционной базе данных. Вам также необходимо будет предоставить API для запроса данных о транзакциях.

## Key Components:
1. Message System:
   ○ Choose either Kafka or RabbitMQ as the message system. The system will receive transaction data (bet/win events) as messages, which need to be processed and saved in the database.
   Выберите в качестве системы обмена сообщениями либо Kafka, либо RabbitMQ. Система будет получать данные о транзакциях (события ставок/выигрышей) в виде сообщений, которые необходимо обработать и сохранить в базе данных
2. Database:
   ○ Choose either PostgreSQL or MySQL as the database to store the transaction data. Each transaction will include the following fields:
   ■ user_id (The ID of the user making the transaction)
   ■ transaction_type (Either "bet" or "win")
   ■ amount (The amount of money for the transaction)
   ■ timestamp (The time the transaction occurred)
3. Transaction API:
   ○ The client needs to query transaction data, either for a single user or for all transactions. It should also support filtering by transaction type (e.g., bet, win, or all transactions).
   Клиенту необходимо осуществлять запрос данных о транзакциях — как по отдельному пользователю, так и по всем транзакциям. Также должна быть реализована возможность фильтрации по типу транзакции (например, ставка, выигрыш или все транзакции).

## Requirements:
1. Message Consumer:
   ○ Create a message consumer that listens for messages (bet/win transactions) from the chosen message system (Kafka or RabbitMQ).
   ○ The consumer must process the messages asynchronously and store the transaction details in the chosen database.
2. Database:
   ○ Set up the database schema to store the transaction data.
3. API:
   ○ Implement the API in Go. The API must allow users to query their transaction history, with support for filtering by transaction_type (e.g., bet, win, or all).
   ○ Ensure the API returns the transactions in JSON format.
4. Testing:
   ○ Write unit and integration tests for all components.
   ○ Test coverage should be at least 85%.
5. Documentation:
   ○ Provide a README file with any relevant instructions
   Submission:
   ● Please submit the source code (how you prefer) along with the README file.

# TODO
- [x] project name n struct
- [ ] log/slog
- [ ] kafka
   - [ ] consumer
   - [ ] processor
- [ ] psql
   - [ ] db+schema
   - [ ] save
   - [ ] get
- [ ] rest api
   - [ ] rest doc
   - [ ] swagger
- [ ] graceful shutdown
- [ ] connections check n restart
- [ ] tests
   - [ ] unit-test
   - [ ] integration-test
   - [ ] example
- [ ] ci/cd
   - [ ] docker
   - [ ] docker compose
   - [ ] makefile

Поток данных: HTTP DTO -> Domain Entity -> Service -> Repository -> DB

Для тестового на Лида я бы советовал использовать log/slog. Это покажет, что вы следите за актуальным состоянием языка 
и не тащите лишние зависимости (вроде zap или logrus) там, где хватает стандарта.

├── cmd
│   ├── api
│   │   └── main.go       # Точка входа REST API
│   └── processor
│       └── main.go       # Точка входа Kafka Consumer
├── internal
│   ├── app               # Composition Root (сборка DI)
│   │   ├── api.go        # func NewApiApp(...)
│   │   └── processor.go  # func NewProcessorApp(...)
│   ├── config
│   │   └── config.go     # Структура Config + cleanenv/viper
│   ├── domain            # ЭНТИТИ (Чистая логика, без импортов)
│   │   ├── transaction.go
│   │   └── transaction_type.go
│   ├── repository        # РЕАЛИЗАЦИЯ БД (Postgres/MySQL)
│   │   └── postgres.go
│   ├── service           # USE CASES (Бизнес-логика)
│   │   ├── interface.go  # Интерфейсы для моков
│   │   └── transaction.go
│   └── transport         # DELIVERY (Внешние интерфейсы)
│       ├── http          # Хендлеры API (Echo/Gin)
│       │   ├── handler.go
│       │   └── router.go
│       └── kafka         # Логика консьюмера
│           └── consumer.go
├── migrations            # SQL файлы (001_init.up.sql)
├── api                   # OpenAPI/Swagger контракты (YAML/JSON)
├── go.mod
└── README.md