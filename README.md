# EasyOffer

Микросервисное приложение для подготовки к техническим собеседованиям. Позволяет изучать вопросы по Go и смежным темам, проходить интерактивные сессии интервью и отслеживать прогресс.

## Архитектура

Приложение состоит из четырёх сервисов:

| Сервис | Порт | Описание |
|---|---|---|
| gateway | 8080 | API-шлюз, маршрутизация, JWT-аутентификация |
| auth-service | 8081 | Регистрация и вход, выдача JWT-токенов |
| question-service | 8082 | Управление вопросами, отзывами, Outbox |
| interview-service | 8083 | Сессии интервью, Event Store, CQRS-проектор |

Инфраструктура: PostgreSQL, Redis, Kafka, Prometheus, Grafana.

```
frontend
    |
  gateway (8080)
    |
    +-- auth-service (8081) -----> PostgreSQL (easyoffer_auth)
    |
    +-- question-service (8082) -> PostgreSQL (easyoffer_question)
    |       |
    |    Outbox --> Kafka (questions.events)
    |                   |
    +-- interview-service (8083) <-- Kafka consumer
            |
          Redis (Event Store, Session Store, Question Cache)
```



## Запуск

```bash
cd infrastructure
docker-compose up --build
```

После запуска:

- API: http://localhost:8080
- Grafana: http://localhost:3000 (admin / admin)
- Prometheus: http://localhost:9090



## Лабораторные работы

### Лаб 1. Контейнеризация

Все сервисы и инфраструктура описаны в [`infrastructure/docker-compose.yml`](infrastructure/docker-compose.yml).

- Миграции БД выполняются отдельными контейнерами (`migrate/migrate`) с `condition: service_completed_successfully`.
- PostgreSQL и Redis имеют healthcheck; сервисы запускаются только после их готовности.
- Базы данных `easyoffer_auth` и `easyoffer_question` создаются init-контейнером `postgres-db-init`.

### Лаб 2. Кеширование

`interview-service` не использует PostgreSQL — все данные хранятся в Redis:

- **Сессии интервью** — [`session_repository.go`](services/interview/internal/repository/session_repository.go), ключи `interview:session:<id>`.
- **Вопросы** — [`question_repository.go`](services/interview/internal/repository/question_repository.go), ключи `interview:questions:<category>`.
- **Event Store** — [`event_store.go`](services/interview/internal/repository/event_store.go), ключи `interview:events:session:<id>`.

### Лаб 3. Event-driven архитектура (Kafka + Outbox)

Взаимодействие `question-service` → `interview-service` через Kafka, топик `questions.events`.

**Outbox pattern в question-service:**

1. При создании вопроса событие атомарно записывается в таблицу `outbox_events` (миграция [`000003_create_outbox_events.up.sql`](services/question/migrations/000003_create_outbox_events.up.sql)).
2. Фоновый [`OutboxDispatcher`](services/question/internal/events/outbox_dispatcher.go) читает необработанные события и публикует их в Kafka.
3. При сбое публикации реализован exponential backoff (до 30 секунд).

**Consumer в interview-service:**

[`question_consumer.go`](services/interview/internal/consumer/question_consumer.go) слушает топик и обновляет локальный кеш вопросов в Redis.

### Лаб 4. Observability

Все четыре сервиса экспонируют `/metrics` в формате Prometheus.

Конфигурация скрейпинга: [`infrastructure/prometheus/prometheus.yml`](infrastructure/prometheus/prometheus.yml).

Кастомные метрики:

- `easyoffer_interview_projector_runs_total` — количество запусков проектора по статусу.
- `easyoffer_interview_projector_projected_sessions_total` — количество спроецированных сессий.
- `easyoffer_interview_projector_max_lag_events` — отставание проектора в событиях.
- `easyoffer_question_outbox_dispatched_total` — события Outbox по типу и статусу.
- `easyoffer_interview_kafka_consumed_total` — потреблённые сообщения Kafka.

Grafana на порту 3000 содержит преднастроенные дашборды из [`infrastructure/grafana/dashboards/`](infrastructure/grafana/dashboards/).

### Лаб 5. CQRS и Event Sourcing

Реализованы в `interview-service`.

**Event Sourcing:**

Каждое действие пользователя порождает доменное событие и сохраняется в Redis-лист (`RPUSH`):

| Событие | Когда |
|---|---|
| `session.started` | Старт сессии интервью |
| `answer.submitted` | Ответ на вопрос |
| `session.finished` | Завершение сессии |

Структура событий: [`domain/events.go`](services/interview/internal/domain/events.go).
Event Store с оптимистичной блокировкой через `WATCH`: [`repository/event_store.go`](services/interview/internal/repository/event_store.go).

**CQRS:**

- **Write side** — команды (`StartSession`, `SubmitAnswer`, `FinishSession`) пишут события в Event Store: [`service/interview_service.go`](services/interview/internal/service/interview_service.go).
- **Read side** — [`InterviewProjector`](services/interview/internal/service/projector.go) работает в фоновом горутине, периодически читает события из стора и перестраивает read-модель сессии в Redis.
- Проектор хранит контрольную точку (`checkpoint`) для каждой сессии и обрабатывает только новые события.
- `ReplaySession` позволяет восстановить состояние сессии полным проигрыванием лога событий.

## Структура проекта

```
.
├── frontend/                  # React + Vite
├── infrastructure/
│   ├── docker-compose.yml
│   ├── prometheus/
│   └── grafana/
└── services/
    ├── gateway/
    ├── auth/
    │   └── migrations/
    ├── question/
    │   ├── internal/events/   # Outbox, Kafka publisher
    │   └── migrations/
    └── interview/
        ├── internal/
        │   ├── domain/        # Events, Session domain
        │   ├── repository/    # Event Store, Session, Question (Redis)
        │   ├── service/       # Commands, Projector
        │   └── consumer/      # Kafka consumer
        └── tests/
```

## Переменные окружения

Все переменные задаются в `docker-compose.yml`. Ключевые:

| Переменная | Сервис | Описание |
|---|---|---|
| `JWT_SECRET` | gateway, auth | Секрет для подписи JWT |
| `KAFKA_ENABLED` | question, interview | Включить Kafka (по умолчанию `false`) |
| `KAFKA_BROKERS` | question, interview | Адрес брокера |
| `REDIS_ADDR` | interview | Адрес Redis |
| `INTERVIEW_PROJECTOR_ENABLED` | interview | Запустить фоновый проектор |
