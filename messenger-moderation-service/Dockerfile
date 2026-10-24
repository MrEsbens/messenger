# ─── Build stage ─────────────────────────────────────────
FROM python:3.11-slim AS builder

WORKDIR /build

RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc python3-dev build-essential \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir --prefix=/install -r requirements.txt

# ─── Run stage ───────────────────────────────────────────
FROM python:3.11-slim

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends curl \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -m -u 1000 appuser

COPY --from=builder /install /usr/local
COPY --chown=appuser:appuser . .
COPY .env.docker.example .env

RUN mkdir -p /app/.cache/huggingface && chown -R appuser:appuser /app

USER appuser

EXPOSE 50058 8000

HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8000/health || exit 1

# 🔥 Исправлено: запускаем скрипт напрямую, как в Makefile
CMD ["python", "main.py"]