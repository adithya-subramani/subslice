# Project Name: `subslice`

---

### 1. Executive Summary & Core Value Proposition

Enterprise engineering teams face a critical trade-off when populating staging, preview, and local developer environments:

1. **Full Database Clones**: Copying multi-hundred-gigabyte production/staging databases to every environment causes massive cloud storage bills, long restore wait times (45+ minutes), and serious compliance risks (PII on developer laptops).
2. **Naive Seed Scripts**: Hand-written seed SQL files fail to capture real production data shapes and break continuously as database schemas evolve.

**`subslice`** is an open-source, CLI-driven **Partial Data Replicator and Test Data Management (TDM) Engine** written in Go. It operates on existing **read-only staging replicas, backup snapshots, or isolated pre-prod instances**. Given a root target selection (e.g., entity `"tenants"` where `id = 'org_123'`), `subslice` automatically inspects the target database schema or document model, extracts a referentially complete 1–5% sub-slice across dependent entities, masks PII in-flight using deterministic rules, and streams the result directly into local or ephemeral target datastores in seconds.

---

### 2. Storage-Agnostic Architecture & Operational Models

`subslice` operates deterministically without AI dependencies, utilizing graph traversal and streaming I/O. To ensure true database agnosticism across both SQL and NoSQL engines, the core pipeline processes data as abstract `Record` documents (`map[string]interface{}`) through pluggable `Connector` adapters rather than coupling directly to specific SQL drivers.

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
┌─────────────────────────────────────┐     ┌─────────────────────────────────────┐
│ MODE A: CI/CD Ephemeral Target      │     │ MODE B: Developer Local Target      │
│ (PR Preview Apps / Integration Tests)│     │ (Local DB / Docker Container / S3)  │
└─────────────────────────────────────┘     └─────────────────────────────────────┘

```

#### Operational Modes

* **Mode A: CI/CD Ephemeral Pipeline (Automated)**: Triggered via GitHub Actions / GitLab CI. `subslice` connects to an isolated staging snapshot, extracts a 1% anonymized subset for a test tenant, and populates a temporary service container for integration testing.
* **Mode B: Developer Local Pull (On-Demand)**: Developers run `subslice` locally against an authorized staging replica to pull a sub-second, 500 MB dataset for a specific tenant to debug issues locally without downloading a multi-hundred-gigabyte dump.

---

### 3. Core Technical Modules & Abstractions

#### Module 1: Abstract Storage Connector Interface

To isolate the graph traversal and PII masking engine from underlying storage technologies, all database drivers implement a unified `Connector` interface:

* **Unified Record Model**: Internal data streams utilize generic `Record` primitives (`EntityName string`, `PKValue string`, `Data map[string]interface{}`).
* **Schema & Metadata Discovery**:
* **Relational Stores (SQL)**: Queries system views (`information_schema.table_constraints`, `key_column_usage`) to discover primary/foreign keys.
* **Document Stores (NoSQL)**: Inspects collection schemas, sampling records to infer implicit relationships (e.g., MongoDB `ObjectId` references or DynamoDB keys).


* **Implicit Relationship Overrides**: Parses user configuration to declare logical relationships not explicitly enforced by DB constraints (application-level joins, polymorphic keys, or cross-collection references).
* **Cycle Resolution**: Detects circular references (e.g., `User` $\rightarrow$ `Org` $\rightarrow$ `User`) and generates a two-pass insertion strategy (inserts primary records with `NULL` or placeholder keys, followed by deferred updates).

#### Module 2: Deterministic Graph Traversal Engine

* **Upstream & Downstream Resolution**:
* **Upstream**: Collects mandatory parent records required for referential integrity (e.g., parent `organizations` or `billing_plans`).
* **Downstream**: Collects associated child records (e.g., user's `orders`, `invoices`, and `audit_logs`).


* **Visited Set Tracking**: Uses in-memory bloom filters and dynamic key tracking to ensure every primary/entity key is traversed and fetched exactly once.

#### Module 3: Zero-Allocation Streaming & PII Masking

* **In-Flight Transformations**: Intercepts generic `Record` buffers in memory to mask sensitive fields prior to streaming:
* `hash`: Deterministic HMAC-SHA256 with a secret salt (preserves join continuity across tables/collections).
* `fake`: Seeded synthetic text generation (e.g., names, addresses).
* `nullify`: Clears high-risk fields (e.g., `credit_card_cvv`, `auth_tokens`).
* `redact`: Retains partial patterns (e.g., `**** **** **** 1234`).


* **Memory-Bounded Streaming**: Utilizes Go `sync.Pool` byte buffers and high-performance native bulk operations (`COPY FROM` for PostgreSQL, bulk writes for MongoDB/DynamoDB) to stream data in chunks without accumulating entire datasets in RAM.

---

### 4. Non-Functional Requirements (NFRs)

| NFR Metric | Requirement Target | Architectural Implementation |
| --- | --- | --- |
| **Referential Integrity** | **100% Zero FK / Link Violations** | Strict topological sort ordering; deferred key updates for cyclic graphs. |
| **Memory Footprint** | **$< 64\text{ MB}$ RSS** | Streaming batch processing ($1,000\text{ record}$ chunks) via Go channels. |
| **Throughput** | **$\ge 10,000\text{ records/sec}$** | Concurrent worker pools (`errgroup`) handling independent entity branches. |
| **Determinism** | **Zero AI / Hallucination Risk** | Pure algorithmic graph traversal and hash-based masking rules. |
| **Database Compatibility** | PostgreSQL, MySQL, CockroachDB, SQLite, MongoDB, DynamoDB | Abstracted via unified `Connector` interface and generic `Record` buffers. |

---

### 5. Enterprise-Compliant Configuration (`subslice.yaml`)

```yaml
version: "1"

# Connection configurations sourced from environment variables (No hardcoded credentials)
source:
  driver: "postgres" # Options: postgres, mysql, cockroach, sqlite, mongodb, dynamodb
  url: "${STAGING_SNAPSHOT_URL}" # e.g., postgres://reader@staging-replica.internal:5432/app_staging

target:
  driver: "postgres"
  url: "${DEV_TARGET_URL}"       # e.g., postgres://postgres:postgres@localhost:5432/app_dev

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

### 6. Command Line Interface (CLI) Workflow

```bash
# 1. Validate configuration and inspect storage graph without extracting data
subslice plan --config ./configs/subslice.yaml

# Output Example:
# [INFO] Inspected storage (driver: postgres): 54 entities, 82 foreign key relationships detected.
# [INFO] Graph traversal plan generated:
#        ├── Upstream parents: 4 entities (organizations, plans, regions, roles)
#        └── Downstream children: 12 entities (users, orders, invoices, audit_logs...)
# [INFO] Estimated slice size: ~1,450 records (~1.2 MB).

# 2. Execute replication with streaming in-flight PII masking
subslice run --config ./configs/subslice.yaml --verbose

# Output Example:
# [INFO] Streaming batch (workers=8)...
# [OK]   tenants (1 record)
# [OK]   users (140 records, 140 PII transformations applied)
# [OK]   orders (1,309 records)
# [SUCCESS] Replication complete in 1.42s. 0 referential integrity violations, Peak RSS: 21.8 MB.

```
