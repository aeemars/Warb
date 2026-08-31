# Warb: Proactive Client Opportunity Engine


An AI-powered engine that understands Warba Bank's corporate clients and proactively surfaces smart product/service suggestions to Relationship Managers, shifting the RM role from reactive admin to proactive advisory.

##  What It Does

- **Analyzes corporate client profiles** against Warba Bank's Shariah-compliant product catalog using AI (InclusionAI Ling 3.0 Flash Fin via OpenRouter)
- **Proactively identifies opportunities** with confidence scores, urgency ratings, and actionable next steps
- **Ensures Shariah compliance** — every suggestion includes Islamic finance governance notes
- **Google Sign-In & User Profiles** — Authenticate with Google, track RM sessions, and display customized portfolios
- **Provides a premium dashboard** for RMs to explore clients, review AI suggestions, and manage their opportunity pipeline

## 🏗 Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                  PROACTIVE OPPORTUNITY ENGINE                   │
├─────────────────────────────────────────────────────────────────┤
│  Web Dashboard   ◄──►   Go HTTP API   ◄──►   AI Engine          │
│  (HTML/CSS/JS)         (REST + CORS)       (OpenRouter)         │
│         │                     │                                 │
│  Google Sign-In         SQLite Database                         │
│  (Google Identity)    (Users, Sessions,                         │
│                       Clients, Products,                        │
│                       Opportunities)                            │
└─────────────────────────────────────────────────────────────────┘
```

| Component | Technology |
|-----------|-----------|
| Backend API | Go (stdlib `net/http`) |
| Database | SQLite (pure Go via `modernc.org/sqlite`) |
| AI Integration | OpenRouter API → InclusionAI Ling 3.0 Flash Fin |
| Authentication | Google Identity Services (GSI) + SQLite Sessions |
| Frontend | Vanilla HTML/CSS/JS + Chart.js |

##  Quick Start

### Prerequisites
- Go 1.21+
- OpenRouter API key ([openrouter.ai](https://openrouter.ai))
- *(Optional)* Google OAuth 2.0 Client ID ([Google Cloud Console](https://console.cloud.google.com/))

### Setup

```bash
# Clone
git clone <repo-url> && cd Warb

# Configure
cp .env.example .env
# Edit .env and add your OPENROUTER_API_KEY (and optional GOOGLE_CLIENT_ID)

# Run
make run
```

The server starts at **http://localhost:8080** with a seeded database of 20 realistic corporate clients.

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `OPENROUTER_API_KEY` | (required) | Your OpenRouter API key |
| `PORT` | `8080` | Server port |
| `AI_MODEL` | `inclusionai/ling-3.0-flash-fin:free` | AI model slug |
| `DB_PATH` | `./data/opportunity.db` | SQLite database path |
| `GOOGLE_CLIENT_ID` | (optional) | Google OAuth 2.0 Client ID for Google Sign-In |

## 🔐 Google Sign-In Setup

1. Go to the [Google Cloud Console](https://console.cloud.google.com/).
2. Create a project and configure the **OAuth consent screen**.
3. Create **Credentials** -> **OAuth client ID** (Web application).
4. Add to **Authorized JavaScript origins**: `http://localhost:8080` (and `http://127.0.0.1:8080`).
5. Copy your **Client ID** and add it to your `.env` file:
   ```env
   GOOGLE_CLIENT_ID=your_client_id_here.apps.googleusercontent.com
   ```
6. Run `make run`. The dashboard will automatically render the official Google Sign-In button and authenticate your account.

*Note: If no `GOOGLE_CLIENT_ID` is provided, the dashboard offers a quick one-click Demo Sign-In mode for immediate local testing.*

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/auth/config` | Get Google Client ID and auth status |
| `POST` | `/api/auth/google` | Authenticate with Google ID token credential |
| `GET` | `/api/auth/me` | Get currently logged-in user profile |
| `POST` | `/api/auth/logout` | Log out and invalidate session |
| `GET` | `/api/clients` | List all corporate clients |
| `GET` | `/api/clients/:id` | Client detail with products & interactions |
| `POST` | `/api/clients/:id/analyze` | Trigger AI analysis for a client |
| `GET` | `/api/opportunities` | List opportunities (filterable by status, urgency, client) |
| `PATCH` | `/api/opportunities/:id` | Update opportunity status |
| `POST` | `/api/portfolio/scan` | AI-scan the entire portfolio |
| `GET` | `/api/portfolio/summary` | Portfolio health metrics |
| `GET` | `/api/products` | Warba Bank product catalog |

## Shariah Compliance

All product recommendations are limited to Warba Bank's Shariah-compliant offerings:
- **Murabaha** (cost-plus & deferred sale)
- **Ijara** (lease-to-own & usufruct)
- **Wakala** (agency)
- **Kafalah** (guarantee)
- **Mudaraba** (profit-sharing)
- **Musharakah** (participatory)
- **Bai Al-Dayn** (sale of receivables)

The AI system prompt enforces these constraints and every suggestion includes Shariah governance notes.

## Synthetic Data

The MVP includes 20 realistic Kuwaiti corporate client profiles across 14 industries, with:
- Full interaction histories (meetings, calls, transactions)
- Current product holdings mapped to Warba's real product suite
- Client notes with expansion plans, contract wins, and financing needs

This data models real-world corporate banking scenarios and is designed to demonstrate the engine's analytical capabilities.


Built for the **Warba Bank × Ignyte Corporate Banking AI Challenge 2026**
