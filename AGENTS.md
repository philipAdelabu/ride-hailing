# AGENTS.md — Ride-Hailing Backend Agents

> This file defines autonomous agents and their collaboration rules for the backend side of the **RideHailing** project.  
> The goal is to ensure clear responsibility boundaries, smooth coordination, and consistent code quality across all services.

---

## 🔧 Global Context

**Tech Stack**

-   Language: Go 1.22+
-   Frameworks: Gin / Fiber / Gorilla Mux
-   Database: PostgreSQL (with pgx)
-   Caching: Redis
-   Messaging: Google Pub/Sub (or NATS)
-   Auth: Firebase Auth / JWT
-   Deployment: Cloud Run + Cloud SQL
-   Observability: Prometheus + Grafana

**Primary Repositories**

-   `ridehailing-backend/` – main monorepo for backend services
-   `ridehailing-shared/` – shared libraries (proto defs, DTOs, utils)

---

## 🧠 Agents Overview

| Agent           | Role                        | Description                                                                   |
| --------------- | --------------------------- | ----------------------------------------------------------------------------- |
| `planner`       | System Architect            | Designs backend architecture, defines APIs, data models, event flows          |
| `auth`          | Authentication Engineer     | Manages user identity, sessions, permissions, driver/rider roles              |
| `rides`         | Core Ride Logic Engineer    | Handles trip creation, matching, pricing, route updates, cancellations        |
| `payments`      | Payment & Wallet Specialist | Integrates payments (Stripe/Adyen/local PSP), fare calculations, transactions |
| `geo`           | Geolocation & Maps Engineer | Manages GPS tracking, driver location updates, route optimization             |
| `notifications` | Messaging Engineer          | Push/SMS/email notifications for status updates and alerts                    |
| `infra`         | DevOps & Observability      | Handles CI/CD, metrics, logging, scaling, and security policies               |

---

## ⚙️ Agent Responsibilities

### 1. `planner`

-   Drafts and updates backend architecture diagrams
-   Defines folder structure (`/cmd`, `/internal`, `/pkg`, `/api`, `/db`)
-   Maintains API contract (`/api/openapi.yaml`)
-   Reviews pull requests for consistency and standards

### 2. `auth`

-   Implements JWT/Firebase-based authentication
-   Builds driver and rider registration flows
-   Enforces RBAC (role-based access control)
-   Secures endpoints and tokens

### 3. `rides`

-   Manages ride lifecycle: `requested → accepted → in_progress → completed`
-   Integrates with `geo` for ETA and distance
-   Handles dynamic pricing logic
-   Exposes REST/gRPC endpoints for ride state changes

### 4. `payments`

-   Manages driver/rider wallets
-   Integrates with payment gateways (Stripe, local PSP)
-   Handles refund, cancellation fees, commission splits
-   Syncs fare receipts and invoices to `rides`

### 5. `geo`

-   Handles real-time driver location updates via WebSockets or Pub/Sub
-   Calculates ETA, distance, and optimal driver matching
-   Uses Google Maps or OpenStreetMap APIs
-   Maintains Redis-based driver proximity index

### 6. `notifications`

-   Sends push notifications to riders and drivers (Firebase / Twilio)
-   Subscribes to Pub/Sub ride events
-   Handles background job queues for retries

### 7. `infra`

-   Manages CI/CD pipelines via GitHub Actions
-   Defines Dockerfiles and deployment manifests
-   Exposes Prometheus metrics for each service
-   Configures logging, secrets, and backups

---

## 🪩 Coordination Rules

-   Each agent owns its domain. Cross-domain features must go through `planner`.
-   Common dependencies (models, utils, errors) live under `/shared` or `/pkg/common`.
-   All services must expose `/healthz`, `/metrics`, `/version`.
-   Every PR must pass:
    -   Linter (`golangci-lint run`)
    -   Unit tests (`go test ./...`)
    -   Integration tests (PostgreSQL + Redis)
-   Communication between services is via REST (short-term) → gRPC or Pub/Sub (long-term).
-   Code comments must follow GoDoc style for AI readability.

---

## 📚 Example Workflow

1. `planner` defines the new **Surge Pricing** feature.
2. `rides` adds pricing logic using time + location factors.
3. `payments` updates fare calculation pipeline.
4. `geo` provides distance matrix API.
5. `infra` ensures the services are redeployed and monitored.
6. `notifications` triggers ride completion messages.

---

## 🧩 Future Agents (Optional)

| Agent       | Role                                             |
| ----------- | ------------------------------------------------ |
| `analytics` | Collects ride and revenue metrics for dashboards |
| `fraud`     | Detects suspicious trips and duplicate accounts  |
| `admin`     | Internal dashboard API for support & operations  |

---

## 📄 Output Expectations

All agents must produce:

-   Structured Go code with clear package separation
-   API documentation (`/docs` or OpenAPI spec)
-   Unit + integration tests
-   Example API requests for Postman / Thunder Client

---

**Version:** 1.0  
**Scope:** Backend only  
**Last Updated:** 2025-11-04  
**Maintainer:** `planner` agent
