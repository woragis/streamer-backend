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
| GET/PUT | `/api/v1/rooms/{roomId}/calisthenics/state` | PUT: Bearer |

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

# Stream timer — ação rápida
curl -X PUT http://localhost:8080/api/v1/rooms/default/timers/stream \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{"action":"start"}'
```

### Optimistic locking

- Respostas incluem `"revision": N` e header `ETag: N`
- PUT com `If-Match: N` ou `"revision": N` no body
- Conflito → `409 revision conflict`

## Documentação completa

[../docs/README.md](../docs/README.md)

## Status

- [x] Fase A — MVP Sync
- [ ] Fase B — Calisthenics model
- [ ] Fase C — LeetCode model
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
