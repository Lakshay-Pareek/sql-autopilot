# SQL Autopilot

A real-time SQL query optimization engine that analyzes PostgreSQL execution plans, detects bottlenecks, and automatically rewrites slow queries.

## What it does

Paste any SQL query → get instant analysis showing:
- **Bottleneck detection** — finds Seq Scans, high-cost nodes, stale statistics
- **Query plan tree** — visual breakdown of how PostgreSQL executes your query
- **Rewrite suggestions** — specific index creation statements + optimized query
- **Speedup estimate** — measured before vs after comparison
- **Query history** — every analysis saved, searchable, re-runnable

## Demo

![SQL Autopilot Demo](demo.png)

## Architecture
```
Next.js Frontend (port 3000)
        ↓
Go API Gateway (port 8080)
  ├── EXPLAIN ANALYZE runner
  ├── Bottleneck detector
  └── History tracker
        ↓
Python Rewriter Service (port 8000)
  ├── Rule engine (12 rewrite rules)
  └── Index suggester
        ↓
PostgreSQL (port 5432)
```

## Tech stack

- **Go** — API gateway, EXPLAIN plan parser, bottleneck detection algorithm
- **Python + FastAPI** — SQL rewrite engine with rule-based optimization
- **Next.js 14 + TypeScript** — frontend with real-time analysis UI
- **PostgreSQL** — database + query execution
- **Docker** — containerized database setup
- **Redis** — (coming soon) plan caching

## Getting started

### Prerequisites
- Go 1.21+
- Python 3.11+
- Node.js 18+
- Docker

### Run locally

**1. Start the database**
```bash
docker compose up -d
```

**2. Start the Go gateway**
```bash
cd services/gateway
go run main.go analyzer.go rewriter_client.go history.go
```

**3. Start the Python rewriter**
```bash
cd services/rewriter
venv\Scripts\activate  # Windows
source venv/bin/activate  # Mac/Linux
uvicorn main:app --reload --port 8000
```

**4. Start the frontend**
```bash
cd frontend
npm run dev
```

Open http://localhost:3000

## Benchmark results

| Query | Before | After | Speedup |
|-------|--------|-------|---------|
| WHERE amount > 500 (no index) | 2.18ms | 0.15ms | 14x |
| WHERE customer_name = 'X' (no index) | 2.08ms | 0.09ms | 23x |
| SELECT * full table scan | 4.87ms | 0.14ms | 34x |

## What makes it different from GitHub Copilot

Copilot suggests SQL without knowing your data. SQL Autopilot runs `EXPLAIN ANALYZE` against your actual database, finds the exact bottleneck in the execution plan, and suggests fixes with measured speedup proof.

## Roadmap

- [ ] VS Code extension
- [ ] ML-based rewrite suggestions (T5 model)
- [ ] Vector similarity search for past query lookup
- [ ] Multi-database support (MySQL, SQLite)
- [ ] Deployment to Railway + Vercel 
