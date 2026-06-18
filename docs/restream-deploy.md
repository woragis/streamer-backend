# Restream RTMP — deploy e variáveis

Arquitetura: **OBS → MediaMTX (restream) → FFmpeg → Kick + YouTube**.  
Overlays e chat continuam no **frontend + API Railway**.

## Serviços

| Serviço | Onde hospedar | Porta |
|---------|---------------|-------|
| Frontend | Railway | HTTPS 443 |
| API (`state-api`) | Railway | HTTPS 443 |
| Worker | Railway (sem domínio) | — |
| **Restream (MediaMTX)** | **VPS** (Hetzner, DO, etc.) | **1935/TCP** |

> Não rode ingest RTMP no Railway — banda e TCP 1935 não são adequados.

---

## Variáveis — API (Railway)

Mesmas de antes, **+ 3 novas**:

```env
DATABASE_URL=${{Postgres.DATABASE_URL}}
REDIS_URL=${{Redis.REDIS_URL}}
STATE_API_TOKEN=seu-token-forte
CORS_ORIGINS=https://streamer.woragis.me
CONSUMER_ENABLED=false
INGEST_MODE=sync
INSTANCE_ID=state-api

# Novas — restream
RESTREAM_PUBLIC_URL=rtmp://rtmp.woragis.me:1935
RESTREAM_RELAY_SOURCE=rtmp://127.0.0.1:1935
RESTREAM_INTERNAL_TOKEN=token-interno-restream
```

| Variável | Uso |
|----------|-----|
| `RESTREAM_PUBLIC_URL` | URL copiada no `/control` para o OBS (domínio do **VPS**) |
| `RESTREAM_RELAY_SOURCE` | Só relevante se API e restream estiverem na **mesma máquina**; no VPS use `rtmp://127.0.0.1:1935` |
| `RESTREAM_INTERNAL_TOKEN` | Segredo compartilhado com o container restream (≠ token público do control) |

Se `RESTREAM_INTERNAL_TOKEN` omitido, usa `STATE_API_TOKEN`.

---

## Variáveis — Worker (Railway)

```env
DATABASE_URL=${{Postgres.DATABASE_URL}}
REDIS_URL=${{Redis.REDIS_URL}}
INSTANCE_ID=platform-worker
```

Sem mudanças para restream.

---

## Variáveis — Frontend (Railway build)

```env
VITE_API_URL=https://streamer.api.woragis.me
VITE_STATE_API_TOKEN=<mesmo STATE_API_TOKEN>
VITE_API_SYNC=true
VITE_ROOM_ID=codes
```

Sem variáveis RTMP no frontend.

---

## Variáveis — Restream (VPS)

Deploy com `backend/Dockerfile.restream` (MediaMTX + FFmpeg + hook Go):

```env
RESTREAM_API_URL=https://streamer.api.woragis.me
RESTREAM_AUTH_URL=https://streamer.api.woragis.me/api/v1/restream/auth
RESTREAM_INTERNAL_TOKEN=<mesmo da API>
```

Porta exposta: **1935/tcp**. DNS opcional: `rtmp.woragis.me` → IP do VPS.

---

## Configuração no `/control`

1. Room **codes** ou **calisthenics**
2. **Restream RTMP** → ativar, colar Kick/YouTube keys de saída, Salvar
3. **Regenerar** ingest key → copiar para OBS
4. **YouTube & Kick** → slug + API (chat) — separado do vídeo

### OBS (Custom service)

| Campo | Valor |
|-------|--------|
| Servidor | `rtmp://rtmp.woragis.me:1935/live/codes` |
| Stream Key | `wrgs_…` (ingest key do control) |

Path por room: `live/codes`, `live/calisthenics`.

---

## Local (docker compose)

```bash
cd streamer
cp backend/.env.example backend/.env
docker compose up -d
```

- API: http://localhost:8080  
- Restream: rtmp://127.0.0.1:1935  
- Control: frontend dev apontando para API local

---

## Endpoints novos

| Método | Path | Auth |
|--------|------|------|
| GET | `/api/v1/rooms/{room}/restream-settings` | GET livre |
| PUT | `/api/v1/rooms/{room}/restream-settings` | Bearer |
| POST | `/api/v1/rooms/{room}/restream-settings/regenerate-ingest-key` | Bearer |
| POST | `/api/v1/restream/auth` | MediaMTX (interno) |
| GET | `/internal/restream/relay/{room}` | `X-Restream-Internal-Token` |
