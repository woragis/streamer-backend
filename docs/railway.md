# Deploy no Railway

Dois serviços no repo `streamer-backend`:

| Serviço | Dockerfile | Domínio público | Config |
|---------|------------|-----------------|--------|
| **API** | `Dockerfile` | Sim (`streamer.api.woragis.me`) | `railway.toml` |
| **Worker** | `Dockerfile.worker` | **Não** | copie `railway.worker.toml` → `railway.toml` no serviço worker |

## API — checklist

1. **Root Directory:** raiz do repo backend (onde está `Dockerfile`)
2. **Builder:** Dockerfile (não Nixpacks)
3. **Start Command:** vazio (usa `ENTRYPOINT ["/server"]`)
4. **Health check:** `/health` (já em `railway.toml`)
5. **Não defina `PORT` manualmente** — Railway injeta; sobrescrever quebra o proxy

### Variáveis

```env
DATABASE_URL=${{Postgres.DATABASE_URL}}
REDIS_URL=${{Redis.REDIS_URL}}
STATE_API_TOKEN=seu-token-forte
CORS_ORIGINS=https://streamer.woragis.me
CONSUMER_ENABLED=false
INGEST_MODE=sync
INSTANCE_ID=state-api
```

Postgres e Redis devem estar **linkados** ao serviço (reference variables).

### Diagnóstico

```bash
curl https://streamer.api.woragis.me/health
```

| Resposta | Ação |
|----------|------|
| `{"status":"starting"}` | Aguarde init |
| `{"status":"error","error":"database: ..."}` | Postgres URL / link |
| `{"status":"ok",...}` | OK |
| `502 Application failed to respond` | Build falhou, Start Command errado, ou serviço não redeployou |

Logs da API devem mostrar:
```text
state-api boot (host=0.0.0.0 port=XXXX ...)
state-api listening on 0.0.0.0:XXXX
state-api ready
```

## Worker

- Mesmo repo, **Dockerfile path:** `Dockerfile.worker`
- Sem domínio público
- Env: `DATABASE_URL`, `REDIS_URL`, `INSTANCE_ID=platform-worker`
