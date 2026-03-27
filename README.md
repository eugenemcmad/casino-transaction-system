# Altenar - Technical Interview Test
Casino Transaction Management System
Overview:
You are tasked with building a simple transaction management system for a casino. The system will track user transactions related to their bets and wins. The transactions will be processed asynchronously via a message system, and the data will be stored in a relational database. You will also need to expose an API to allow querying of transaction data.

## Key Components:
1. Message System:
   ○ Choose either Kafka or RabbitMQ as the message system. The system will receive transaction data (bet/win events) as messages, which need to be processed and saved in the database.
2. Database:
   ○ Choose either PostgreSQL or MySQL as the database to store the transaction data. Each transaction will include the following fields:
   ■ user_id (The ID of the user making the transaction)
   ■ transaction_type (Either "bet" or "win")
   ■ amount (The amount of money for the transaction)
   ■ timestamp (The time the transaction occurred)
3. Transaction API:
   ○ The client needs to query transaction data, either for a single user or for all transactions. It should also support filtering by transaction type (e.g., bet, win, or all transactions).
   
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

## Runtime Configuration

The service uses `config.yaml` (or `CONFIG_PATH`) and environment overrides.

- HTTP:
  - `HTTP_PORT`
  - `HTTP_READ_HEADER_TIMEOUT_SECONDS`
  - `HTTP_READ_TIMEOUT_SECONDS`
  - `HTTP_WRITE_TIMEOUT_SECONDS`
  - `HTTP_IDLE_TIMEOUT_SECONDS`
- Postgres:
  - `PG_URL`
  - `PG_POOL_MAX_OPEN`
  - `PG_POOL_MAX_IDLE`
  - `PG_CONN_MAX_LIFETIME_MINUTES`