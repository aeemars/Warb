# Warba Bank — Proactive Client Opportunity Engine

> **Track 4: Proactive Client Opportunity Engine**  
> Ignyte Corporate Banking AI Challenge 2026

An AI-powered engine that understands Warba Bank's corporate clients and proactively surfaces smart product/service suggestions to Relationship Managers — shifting the RM role from reactive admin to proactive advisory.

## ✨ What It Does

- **Analyzes corporate client profiles** against Warba Bank's Shariah-compliant product catalog using AI (Claude Sonnet via OpenRouter)
- **Proactively identifies opportunities** with confidence scores, urgency ratings, and actionable next steps
- **Ensures Shariah compliance** — every suggestion includes Islamic finance governance notes
- **Provides a premium dashboard** for RMs to explore clients, review AI suggestions, and manage their opportunity pipeline

## 🏗 Architecture

```
┌───────────────────────────────────────────────────────┐
│              PROACTIVE OPPORTUNITY ENGINE              │
├───────────────────────────────────────────────────────┤
│  Web Dashboard  ◄──►  Go HTTP API  ◄──►  AI Engine   │
│  (HTML/CSS/JS)       (REST + CORS)     (OpenRouter)   │
│                          │                             │
│                    SQLite Database                      │
│                 (Clients, Products,                     │
│               Opportunities, History)                   │
└───────────────────────────────────────────────────────┘
```

| Component | Technology |
|-----------|-----------|
| Backend API | Go (stdlib `net/http`) |
| Database | SQLite (pure Go via `modernc.org/sqlite`) |
| AI Integration | OpenRouter API → Claude Sonnet 4 |
| Frontend | Vanilla HTML/CSS/JS + Chart.js |

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- OpenRouter API key ([openrouter.ai](https://openrouter.ai))

### Setup

```bash
# Clone
git clone <repo-url> && cd Banking-AI

# Configure
cp .env.example .env
# Edit .env and add your OPENROUTER_API_KEY

# Run
make run
```

The server starts at **http://localhost:8080** with a seeded database of 20 realistic corporate clients.

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `OPENROUTER_API_KEY` | (required) | Your OpenRouter API key |
| `PORT` | `8080` | Server port |
| `AI_MODEL` | `anthropic/claude-sonnet-4-20250514` | AI model slug |
| `DB_PATH` | `./data/opportunity.db` | SQLite database path |

## 📁 Project Structure

```
Banking-AI/
├── cmd/server/main.go           # Entry point
├── internal/
│   ├── api/
│   │   ├── server.go            # HTTP server + middleware
│   │   └── handlers.go          # REST API handlers
│   ├── engine/
│   │   ├── engine.go            # AI opportunity engine
│   │   └── prompts.go           # Prompt templates
│   ├── models/
│   │   └── models.go            # Domain types
│   └── store/
│       ├── store.go             # SQLite data access
│       └── seed.go              # Synthetic data seeder
├── web/
│   ├── index.html               # Dashboard SPA
│   ├── styles.css               # Design system
│   └── app.js                   # Frontend logic
├── go.mod / go.sum
├── Makefile
└── .env.example
```

## 🔌 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/clients` | List all corporate clients |
| `GET` | `/api/clients/:id` | Client detail with products & interactions |
| `POST` | `/api/clients/:id/analyze` | Trigger AI analysis for a client |
| `GET` | `/api/opportunities` | List opportunities (filterable by status, urgency, client) |
| `PATCH` | `/api/opportunities/:id` | Update opportunity status |
| `POST` | `/api/portfolio/scan` | AI-scan the entire portfolio |
| `GET` | `/api/portfolio/summary` | Portfolio health metrics |
| `GET` | `/api/products` | Warba Bank product catalog |

## 🕌 Shariah Compliance

All product recommendations are limited to Warba Bank's Shariah-compliant offerings:
- **Murabaha** (cost-plus & deferred sale)
- **Ijara** (lease-to-own & usufruct)
- **Wakala** (agency)
- **Kafalah** (guarantee)
- **Mudaraba** (profit-sharing)
- **Musharakah** (participatory)
- **Bai Al-Dayn** (sale of receivables)

The AI system prompt enforces these constraints and every suggestion includes Shariah governance notes.

## 📊 Synthetic Data

The MVP includes 20 realistic Kuwaiti corporate client profiles across 14 industries, with:
- Full interaction histories (meetings, calls, transactions)
- Current product holdings mapped to Warba's real product suite
- Client notes with expansion plans, contract wins, and financing needs

This data models real-world corporate banking scenarios and is designed to demonstrate the engine's analytical capabilities.

## 🏆 Challenge Criteria Addressed

| Criterion | How We Address It |
|-----------|-------------------|
| **Innovation** | AI proactively surfaces opportunities — RMs never need to ask |
| **Technical Excellence** | Clean Go architecture, SQLite for portability, OpenRouter for model flexibility |
| **User Experience** | Premium dark dashboard with confidence meters, urgency badges, and one-click actions |
| **Real-World Impact** | Targets documented RM pain point: 70-75% time on admin vs. advisory |
| **Compliance by Design** | Shariah governance enforced in AI system prompt and product catalog |

---

Built for the **Warba Bank × Ignyte Corporate Banking AI Challenge 2026**
