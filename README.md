# Woragis Stream — Backend

API de estado para sync entre painel `/control` e overlays OBS.

## Quick start

```bash
# Na raiz streamer/: make up  (Postgres + Redis)
cp .env.example .env
make run
```

Servidor em `http://localhost:8080`.

## Endpoints (Fase A)

| Método | Path | Auth |
|--------|------|------|
| GET | `/health` | — |
| GET/PUT | `/api/v1/rooms/{roomId}/session` | PUT: Bearer |
| GET/PUT | `/api/v1/rooms/{roomId}/branding` | PUT: Bearer |
| GET/PUT | `/api/v1/rooms/{roomId}/timers/stream` | PUT: Bearer |
| GET/PUT | `/api/v1/rooms/{roomId}/leetcode/state` | PUT: Bearer |
| GET/POST | `/api/v1/rooms/{roomId}/leetcode/sessions` | POST: Bearer |
| GET/PATCH | `/api/v1/rooms/{roomId}/leetcode/sessions/{sessionId}` | PATCH: Bearer |
| GET/POST | `/api/v1/rooms/{roomId}/leetcode/plan` | POST: Bearer |
| PATCH/DELETE | `/api/v1/rooms/{roomId}/leetcode/plan/{itemId}` | Bearer |
| POST | `/api/v1/rooms/{roomId}/leetcode/plan/{itemId}/toggle` | Bearer |
| GET/POST | `/api/v1/rooms/{roomId}/leetcode/problems` | POST: Bearer |
| GET/PATCH/DELETE | `/api/v1/rooms/{roomId}/leetcode/problems/{problemId}` | write: Bearer |
| POST | `/api/v1/rooms/{roomId}/leetcode/problems/{problemId}/activate` | Bearer |
| POST | `/api/v1/rooms/{roomId}/leetcode/problems/{problemId}/solve` | Bearer |
| POST | `/api/v1/rooms/{roomId}/leetcode/problems/{problemId}/skip` | Bearer |
| GET | `/api/v1/rooms/{roomId}/leetcode/stats?month=2025-06` | — |
| GET | `/api/v1/rooms/{roomId}/leetcode/stats?liveSessionId={id}` | — |
| GET | `/api/v1/rooms/{roomId}/leetcode/stats/streak` | — |
| GET | `/api/v1/rooms/{roomId}/leetcode/attempts?liveSessionId=` | — |
| GET/PUT | `/api/v1/rooms/{roomId}/leetcode/timers/{timerId}` | PUT: Bearer |
| GET/PUT | `/api/v1/rooms/{roomId}/calisthenics/state` | PUT: Bearer |
| GET | `/api/v1/rooms/{roomId}/calisthenics/workouts` | — |
| POST | `/api/v1/rooms/{roomId}/calisthenics/workouts` | Bearer |
| GET/PATCH/DELETE | `/api/v1/rooms/{roomId}/calisthenics/workouts/{workoutId}` | write: Bearer |
| GET/POST | `/api/v1/rooms/{roomId}/calisthenics/workouts/{workoutId}/exercises` | POST: Bearer |
| PATCH/DELETE | `/api/v1/rooms/{roomId}/calisthenics/exercises/{exerciseId}` | Bearer |
| POST | `/api/v1/rooms/{roomId}/calisthenics/exercises/{exerciseId}/activate` | Bearer |
| GET | `/api/v1/rooms/{roomId}/calisthenics/exercises/{exerciseId}/sets` | — |
| POST | `/api/v1/rooms/{roomId}/calisthenics/sets/{setId}/complete` | Bearer |
| POST | `/api/v1/rooms/{roomId}/calisthenics/sets/{setId}/increment-rep` | Bearer |
| POST | `/api/v1/rooms/{roomId}/calisthenics/sets/{setId}/skip` | Bearer |
| GET/PUT | `/api/v1/rooms/{roomId}/calisthenics/timers/{timerId}` | PUT: Bearer |
| GET | `/api/v1/rooms/{roomId}/calisthenics/movements?level=learning` | — |
| GET/POST | `/api/v1/rooms/{roomId}/calisthenics/movements` | POST: Bearer |
| GET/PATCH/DELETE | `/api/v1/rooms/{roomId}/calisthenics/movements/{movementId}` | write: Bearer |
| GET/PUT | `/api/v1/rooms/{roomId}/calisthenics/movements/{movementId}/proficiency` | PUT: Bearer |
| GET | `/api/v1/rooms/{roomId}/calisthenics/movements/{movementId}/history` | — |
| GET/POST | `/api/v1/rooms/{roomId}/calisthenics/acquisitions` | POST: Bearer |
| POST | `/api/v1/rooms/{roomId}/calisthenics/acquisitions/{id}/ack` | Bearer |
| GET | `/api/v1/rooms/{roomId}/calisthenics/stats?month=2026-06` | — |
| WS | `/api/v1/rooms/{roomId}/subscribe?domain=all&token=` | token opcional se `STATE_API_TOKEN` set |
| POST | `/api/v1/rooms/{roomId}/chat/ingest` | Bearer |
| GET | `/api/v1/rooms/{roomId}/chat/messages?limit=50` | — |
| DELETE | `/api/v1/rooms/{roomId}/chat/messages/{messageId}` | Bearer |
| POST | `/api/v1/rooms/{roomId}/events/ingest` | Bearer |
| GET | `/api/v1/rooms/{roomId}/events?limit=50` | — |
| GET/POST | `/api/v1/rooms/{roomId}/rules` | POST: Bearer |
| PATCH/DELETE | `/api/v1/rooms/{roomId}/rules/{ruleId}` | Bearer |
| GET | `/api/v1/rooms/{roomId}/dashboard?month=2026-06` | — |

Room padrão: `default` (seed automático na primeira execução).

### Exemplos

```bash
# Ler estado LeetCode (OBS)
curl http://localhost:8080/api/v1/rooms/default/leetcode/state

# Atualizar cena (control)
curl -X PUT http://localhost:8080/api/v1/rooms/default/session \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{"scene":"brb","startedAt":null,"streamEvents":{"latestSubscriber":"","latestFollower":"","latestDonation":""}}'

# Calisthenics — increment rep no set ativo
curl http://localhost:8080/api/v1/rooms/default/calisthenics/state
# use setDetails[].id do exercise ativo:
curl -X POST http://localhost:8080/api/v1/rooms/default/calisthenics/sets/{setId}/increment-rep \
  -H "Authorization: Bearer dev-token"

# Rest timer
curl -X PUT http://localhost:8080/api/v1/rooms/default/calisthenics/timers/rest \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{"action":"start"}'

# LeetCode — iniciar live e resolver problema
curl -X POST http://localhost:8080/api/v1/rooms/default/leetcode/sessions \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{"domain":"leetcode","platforms":["youtube"],"title":"Daily #47"}'

curl -X POST http://localhost:8080/api/v1/rooms/default/leetcode/problems/239/activate \
  -H "Authorization: Bearer dev-token"

curl -X POST http://localhost:8080/api/v1/rooms/default/leetcode/problems/239/solve \
  -H "Authorization: Bearer dev-token"

curl "http://localhost:8080/api/v1/rooms/default/leetcode/stats/streak"

# Calisthenics — marcar skill adquirida
curl -X POST http://localhost:8080/api/v1/rooms/default/calisthenics/acquisitions \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{"movementId":"muscle-up","proficiencyAfter":"consistent","notes":"Primeiro rep limpo"}'

curl http://localhost:8080/api/v1/rooms/default/calisthenics/state
# skillAlert aparece no state até POST .../acquisitions/{id}/ack

# Chat ingest (webhook-style) — !brb troca scene se regra ativa
curl -X POST http://localhost:8080/api/v1/rooms/default/chat/ingest \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{"platform":"youtube","username":"viewer1","content":"!brb"}'

# Stream event (follower/sub/donation)
curl -X POST http://localhost:8080/api/v1/rooms/default/events/ingest \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{"type":"follower","platform":"kick","username":"newfan"}'

# Dashboard agregado
curl "http://localhost:8080/api/v1/rooms/default/dashboard?month=2026-06"

# WebSocket (OBS/control) — eventos: state.updated, message.created, session.updated
# wscat -c "ws://localhost:8080/api/v1/rooms/default/subscribe?domain=all&token=dev-token"
```

### Optimistic locking

- Respostas incluem `"revision": N` e header `ETag: N`
- PUT com `If-Match: N` ou `"revision": N` no body
- Conflito → `409 revision conflict`

## Documentação completa

[../docs/README.md](../docs/README.md)

## Status

- [x] Fase A — MVP Sync
- [x] Fase B — Calisthenics model (workout → exercise → set)
- [x] Fase C — LeetCode model (problems, attempts, stats, live sessions)
- [x] Fase D — Skill tracking (movements, proficiency, acquisitions)
- [x] Fase E — Chat, WebSocket & dashboard
- [x] Fase F — Redis pub/sub (multi-instance WebSocket)
- [x] Fase F3 — Redis Streams ingest queue + dedup
- [x] Fase G — Postgres (goose migrations, JSONB)

## Stack

- Go 1.25+
- PostgreSQL 16 + pgx
- goose (migrations)
- chi router
- gorilla/websocket
- redis/go-redis (pub/sub + streams)

## Env

| Variável | Default |
|----------|---------|
| `PORT` | `8080` |
| `DATABASE_URL` | `postgres://streamer:streamer@localhost:5432/streamer?sslmode=disable` |
| `STATE_API_TOKEN` | `dev-token` |
| `CORS_ORIGINS` | `http://localhost:5173,...` |
| `REDIS_URL` | _(vazio = desligado)_ |
| `INSTANCE_ID` | hostname ou `state-api` |
| `INGEST_MODE` | `sync` (default) ou `queue` |
| `CONSUMER_ENABLED` | `true` — processa fila ingest no mesmo processo |

## Quick start

Infra (Postgres + Redis) na **raiz do monorepo** `streamer/`:

```bash
# de streamer/
make up
cp .env.example backend/.env
cd backend && make run
curl http://localhost:8080/health
# {"status":"ok","database":"ok","driver":"postgres","redis":"ok",...}
```

Migrations rodam automaticamente no startup (`goose` embedded).

Testes de integração precisam do Postgres:

```bash
# streamer/
make up
cd backend && go test ./...
```

## Redis (Fase F)

Pub/Sub distribui eventos WebSocket entre múltiplas instâncias da API.

```bash
make up                 # na raiz streamer/ — sobe Postgres + Redis
cp .env.example .env    # backend/.env
make run
curl http://localhost:8080/health
# {"status":"ok","redis":"ok","instanceId":"...","ingestMode":"sync","ingestQueuePending":0}
```

Sem `REDIS_URL`, a API funciona como antes (hub in-memory local). Com Redis down, status `degraded` e pub/sub desligado.

### Ingest assíncrono (F3)

Com `INGEST_MODE=queue`, `POST /chat/ingest` e `POST /events/ingest` enfileiram jobs em Redis Streams (`streamer:jobs:ingest`) e retornam `202` com `jobId`. O consumer roda no mesmo processo (`CONSUMER_ENABLED=true`).

Dedup via `externalId` no body — replays idempotentes por 48h:

```bash
curl -X POST http://localhost:8080/api/v1/rooms/default/chat/ingest \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{"platform":"youtube","username":"v1","content":"hi","externalId":"yt-msg-001"}'
```
