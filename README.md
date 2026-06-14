# Woragis Stream — Backend

API de estado para sync entre painel `/control` e overlays OBS.

## Quick start

```bash
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
- [ ] Fase D — Skill tracking
- [ ] Fase E — Chat & analytics

## Stack

- Go 1.22+
- chi router
- modernc.org/sqlite (pure Go)

## Env

| Variável | Default |
|----------|---------|
| `PORT` | `8080` |
| `DATABASE_URL` | `./data/state.db` |
| `STATE_API_TOKEN` | `dev-token` |
| `CORS_ORIGINS` | `http://localhost:5173,...` |
