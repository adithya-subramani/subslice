# 🍰 [subslice]

> **A lightning-fast, storage-agnostic Partial Data Replicator and Test Data Management (TDM) Engine written in Go.**

---

## 🚀 Executive Summary & Core Value Proposition

Enterprise engineering teams face a critical dilemma when populating staging, preview, and local developer environments:

1. **Full Database Clones:** Copying multi-hundred-gigabyte production or staging databases to every environment causes massive cloud storage bills, long restore wait times (45+ minutes), and serious compliance risks (PII sitting on developer laptops).
2. **Naive Seed Scripts:** Hand-written seed SQL files fail to capture real production data shapes and break continuously as database schemas evolve.

**subslice** solves this by operating on existing read-only staging replicas, backup snapshots, or isolated pre-prod instances. Given a root target selection (e.g., `tenants` where `id = 'org_123'`), subslice automatically inspects the target database schema or document model, extracts a referentially complete 1–5% sub-slice across dependent entities, masks PII in-flight using deterministic rules, and streams the result directly into local or ephemeral target datastores in seconds.

---

## 🏗️ Storage-Agnostic Architecture & Operational Models

subslice operates deterministically without AI dependencies, utilizing graph traversal and streaming I/O. To ensure true database agnosticism across both SQL and NoSQL engines, the core pipeline processes data as abstract Record documents (`map[string]interface{}`) through pluggable Connector adapters rather than coupling directly to specific SQL drivers.

```
                  ┌───────────────────────────────────────────────┐
                  │ Read-Only Staging Replica / Backup Snapshot   │
                  │ (PostgreSQL, MySQL, Cockroach, Mongo, Dynamo) │
                  └───────────────────────┬───────────────────────┘
                                          │
                                          ▼
┌──────────────────────────────────────────────────────────────────────────────────┐
│                                 subslice Engine                                  │
│                                                                                  │
│   ┌────────────────────────┐   ┌────────────────────────┐   ┌─────────────────┐  │
│   │  1. Storage Connector  │──▶│ 2. Deterministic Graph │──▶│  3. Streaming   │  │
│   │   (Schema Inspector &  │   │        Engine          │   │   Anonymizer    │  │
│   │    Entity Discovery)   │   │  (DAG & Topological)   │   │  & Copy Pipe    │  │
│   └────────────────────────┘   └────────────────────────┘   └─────────────────┘  │
└─────────────────────────────────────────┬────────────────────────────────────────┘
                                          │
                    ┌─────────────────────┴─────────────────────┐
                    ▼                                           ▼
┌─────────────────────────────────────     ┌─────────────────────────────────────┐
│ MODE A: CI/CD Ephemeral Target      │     │ MODE B: Developer Local Target      │
│ (PR Preview Apps / Integration Tests)│     │ (Local DB / Docker Container / S3)  │
└─────────────────────────────────────┘     └─────────────────────────────────────┘

```

### 🏢 Enterprise Operational Modes

* **Mode A: CI/CD Ephemeral Pipeline (Automated):** Triggered via GitHub Actions / GitLab CI. subslice connects to an isolated staging snapshot, extracts a 1% anonymized subset for a test tenant, and populates a temporary service container for integration testing.
* **Mode B: Developer Local Pull (On-Demand):** Developers run subslice locally against an authorized staging replica to pull a sub-second, 500 MB dataset for a specific tenant to debug issues locally without downloading a multi-hundred-gigabyte dump.

---

## ⚙️ Core Technical Modules

1. **Schema Inspector & FK Graph Builder:** Queries information schemas (`information_schema.table_constraints`, `key_column_usage`) or NoSQL catalogs to construct a Directed Acyclic Graph (DAG). Supports user-configured implicit foreign keys for application-level joins and polymorphic keys.
2. **Deterministic Graph Traversal Engine:** Resolves upstream parents (mandatory prerequisites like parent organizations or plans) and downstream children (associated records like orders, invoices, and audit logs) with cycle resolution and bloom filters to prevent duplicate fetches.
3. **Zero-Allocation Streaming & PII Masking:** Performs in-memory transformations before writing to target streams:
* `hash`: Deterministic HMAC-SHA256 with a secret salt (preserves join continuity across tables).
* `fake`: Seeded synthetic text generation (names, addresses).
* `nullify`: Clears high-risk fields (`credit_card_cvv`, auth tokens).
* `redact`: Retains partial patterns (`**** **** **** 1234`).



---

## 📊 Non-Functional Performance Profile

| NFR Metric | Requirement Target | Architectural Implementation |
| --- | --- | --- |
| **Referential Integrity** | **100% Zero FK Violations** | Strict topological sort ordering; deferred FK updates for cyclic graphs. |
| **Memory Footprint** | **$< 64\text{ MB}$ RSS** | Streaming batch processing (`1,000` row chunks) via Go channels and `sync.Pool`. |
| **Throughput** | **$\ge 10,000\text{ rows/sec}$** | Concurrent worker pools (`errgroup`) handling independent table branches. |
| **Determinism** | **Zero AI / Hallucination Risk** | Pure algorithmic graph traversal and hash-based masking rules. |
| **Database Support** | **Relational & NoSQL** | Abstracted via Go connector interfaces (`postgres`, `mysql`, `sqlite`, `mongodb`, etc.). |

---

## 📄 Enterprise-Compliant Configuration (`subslice.yaml`)

```yaml
version: "1"

# Connection configurations sourced from environment variables (No hardcoded credentials)
source:
  driver: "postgres" # Options: postgres, mysql, cockroach, sqlite, mongodb, dynamodb
  url: "${STAGING_SNAPSHOT_URL}" # e.g., postgres://reader@staging-replica.internal:5432/app_staging

target:
  driver: "mongodb"
  url: "${DEV_TARGET_URL}"       # e.g., mongodb://localhost:27017/app_dev

options:
  max_depth: 5
  batch_size: 1000
  workers: 8

# Root extraction boundary
root:
  entity: "tenants"
  where: "id = '${TARGET_TENANT_ID}'"

# Custom application relationships (Essential for NoSQL and unconstrained SQL keys)
implicit_foreign_keys:
  - entity: "audit_logs"
    field: "target_id"
    foreign_entity: "users"
    foreign_field: "id"

# In-flight PII Masking Rules
transformations:
  users:
    - field: "email"
      rule: "hash"
      salt: "${MASKING_SALT}"
    - field: "full_name"
      rule: "fake_name"
    - field: "ssn"
      rule: "nullify"
  billing_details:
    - field: "card_number"
      rule: "redact_pan"

```

---

## 🛠️ Command Line Interface (CLI) Workflow

```bash
# 1. Validate configuration and inspect storage graph without extracting data
subslice plan --config ./configs/subslice.yaml

# 2. Preview extracted records and applied PII masking safely without writing to target
subslice dry-run --config ./configs/subslice.yaml

# 3. Execute replication with concurrent batched workers and streaming in-flight PII masking
subslice run --config ./configs/subslice.yaml --verbose

```

### Sample Output (`subslice run --verbose`):

```text
[INFO] Run mode initialized using config: subslice.yaml
[INFO] Configured Workers: 8, Batch Size: 1000
[INFO] Extracting Root entity: tenants
[OK]   tenants (1 records replicated)
[OK]   users (2 records replicated concurrently with batching)
[OK]   orders (3 records replicated concurrently with batching)
[SUCCESS] Replicated 6 rows across 4 tables in 9.66s.

```

---

## 📦 Getting Started & Quick Installation

Clone the repository and compile the Go binary:

```bash
git clone https://github.com/your-org/subslice.git
cd subslice
go build -o subslice.exe main.go

```

Set your environment variables and test the plan:

```powershell
$env:STAGING_SNAPSHOT_URL="postgres://reader@staging:5432/db"
$env:DEV_TARGET_URL="mongodb://localhost:27017/dev"
$env:TARGET_TENANT_ID="org_123"
$env:MASKING_SALT="secret_salt"

.\subslice.exe plan --config subslice.yaml
.\subslice.exe dry-run --config subslice.yaml

```

---

## 📄 License

Distributed under the MIT License. See `LICENSE` for more information.