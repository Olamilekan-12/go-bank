# Go Bank: Product Requirements Document

| Field    | Value                         |
|----------|-------------------------------|
| Product  | Go Bank API                   |
| Version  | 0.2.0 (Draft)                 |
| Date     | 2026-09-23                    |
| Owner    | Bello Olamilekan              |
| Status   | Draft, open for revision      |

---

## 1. Purpose

Go Bank is a retail banking backend (REST API) that simulates how a real digital bank works: customer onboarding with KYC documents, authentication, sessions, authorization, accounts, a double-entry ledger, deposits, withdrawals, transfers, limits, fraud checks, notifications, statements, and back-office operations.

It has two equal goals:

1. **Product goal:** build a correct, secure, and scalable banking API that follows industry practice.
2. **Learning goal:** use every part of the system to master the Go language and the engineering habits of a senior backend engineer.

Every feature in this document exists for a reason on both sides. The Go Learning Map (Section 13) shows which Go concept each feature teaches.

---

## 2. Goals and Non-Goals

### 2.1 Goals

- Correct money handling. No lost, duplicated, or invented money under any load or failure.
- Real-world onboarding. KYC checks are tiered and use documents, and a compliance officer reviews each application.
- Production-grade security: encryption of personal data, strong password hashing, sessions that can be revoked, and role-based access.
- Concurrency that is safe and tested. The race detector stays clean and there are no deadlocks under parallel transfers.
- Code that is clean, testable, and well documented, with an OpenAPI spec and a Postman collection that stay in sync with the code.
- Professional Git workflow, CI pipeline, and release process.

### 2.2 Non-Goals

- Integration with real payment networks, credit bureaus, or government identity services. All external providers are **simulated behind Go interfaces** so they can be swapped for real ones later.
- A frontend. Postman and automated tests are the clients.
- Real regulatory certification. We mimic the practices; we do not claim compliance.

---

## 3. Users and Roles

| Role                 | Description                                                                 |
|----------------------|-----------------------------------------------------------------------------|
| Applicant            | A person who has started registration and is not yet an approved customer. |
| Customer             | An approved account holder.                                                 |
| Support Agent        | Staff who views customer profiles, handles disputes, and cannot move money. |
| Teller               | Staff who records cash deposits and cash withdrawals at a branch.           |
| Compliance Officer   | Staff who reviews KYC applications, flagged transactions, and freezes.      |
| Admin                | Staff who manages staff users, roles, fees, and limits.                     |
| Auditor              | Read-only staff with access to audit logs and reports.                      |
| System               | Background workers and scheduled jobs acting on behalf of the bank.        |

---

## 4. Glossary

| Term             | Meaning                                                                                  |
|------------------|------------------------------------------------------------------------------------------|
| KYC              | Know Your Customer. Identity verification required before and during a banking relationship. |
| BVN / NIN        | Bank Verification Number / National Identification Number. Used here as simulated identity numbers. |
| NUBAN            | A 10-digit account number standard with a check digit. We implement a simplified version. |
| Ledger           | The immutable record of all money movements.                                             |
| Double entry     | Every transaction has at least two postings (debit and credit) that sum to zero.        |
| Posting          | A single debit or credit line against one account inside a journal entry.               |
| Available balance| Ledger balance minus holds (liens).                                                      |
| Hold / Lien      | An amount reserved on an account that cannot be spent.                                  |
| Idempotency key  | A client-supplied unique key that makes retrying a request safe.                        |
| Maker-checker    | A control where one staff member initiates an action and a different one approves it.  |
| Minor units      | Money stored as integers in the smallest unit (kobo for NGN, cents for USD).             |
| Outbox           | A table that stores events in the same database transaction as the business change, for reliable async delivery. |

---

## 5. Functional Requirements

Requirement IDs are used in commits, pull requests, and tests (for example `feat(auth): add refresh rotation [FR-AUTH-05]`).

### 5.1 Onboarding and Registration (ONB)

Registration works like opening an account at a real digital bank. The applicant moves through a series of steps, and the application has a state that can be resumed.

**Application states:** `draft` -> `submitted` -> `under_review` -> `approved` | `rejected` | `more_info_required` (which returns to `submitted` after the applicant responds).

| ID          | Requirement |
|-------------|-------------|
| FR-ONB-01 | Applicant starts an application with email and phone number. Duplicate email or phone for an active customer is rejected. |
| FR-ONB-02 | Email and phone are verified with one-time codes (6 digits, 10-minute expiry, max 5 attempts, resend cooldown of 60 seconds). |
| FR-ONB-03 | Applicant submits personal details: legal first, middle, and last name, date of birth (must be 18 or older), gender (optional), nationality, occupation, source of funds, and residential address. |
| FR-ONB-04 | Applicant submits identity numbers (BVN and/or NIN). These are checked against a simulated Identity Provider that returns a match score for name and date of birth. |
| FR-ONB-05 | Applicant uploads KYC documents: one government ID (international passport, national ID card, driver's license, or voter's card) with number and expiry date, a proof of address (utility bill or bank statement dated within 3 months), and a selfie. |
| FR-ONB-06 | Uploads are validated: allowlisted content types (JPEG, PNG, PDF) detected by file signature (magic bytes), not by extension; max 5 MB each; a SHA-256 checksum is stored; expired IDs are rejected. |
| FR-ONB-07 | Applicant sets a password (policy: min 12 chars, checked against a list of common passwords) and a separate 4-digit transaction PIN (no trivial sequences such as 1234 or 0000). |
| FR-ONB-08 | Applicant accepts Terms and Privacy Policy. Consent is recorded with document version, timestamp, and IP. |
| FR-ONB-09 | On submission, automated checks run concurrently: identity match, sanctions/PEP screening (simulated), duplicate identity detection, and document sanity checks. Results are attached to the application. |
| FR-ONB-10 | A compliance officer reviews the application and can approve, reject (with reason code), or request more information (with a note). |
| FR-ONB-11 | On approval, a customer record is created, a KYC tier is assigned, a default NGN savings account is opened, and a welcome notification is sent. |
| FR-ONB-12 | Abandoned `draft` applications and their files are purged after 30 days. |

### 5.2 KYC Tiers (KYC)

The tiers are modeled on the CBN tiered KYC framework. All values are configurable and only illustrative.

| Tier | Requirements                                      | Daily debit limit (NGN) | Max balance (NGN) |
|------|---------------------------------------------------|-------------------------|-------------------|
| 1    | Verified phone, email, BVN or NIN, selfie         | 50,000                  | 300,000           |
| 2    | Tier 1 plus verified government ID                | 200,000                 | 500,000           |
| 3    | Tier 2 plus verified proof of address and review  | 5,000,000               | Unlimited         |

| ID          | Requirement |
|-------------|-------------|
| FR-KYC-01 | A customer can request an upgrade to a higher tier by submitting the missing documents, which goes through review. |
| FR-KYC-02 | Tier limits are enforced on every debit and every credit (max balance). |
| FR-KYC-03 | Expired ID documents trigger a notification 30 days before expiry and a downgrade to the lower tier on expiry. |

### 5.3 Authentication and Sessions (AUTH)

| ID          | Requirement |
|-------------|-------------|
| FR-AUTH-01 | Login with email and password. Passwords are hashed with Argon2id. |
| FR-AUTH-02 | After 5 failed attempts, the account is locked for 15 minutes. Error messages never reveal whether an email exists. |
| FR-AUTH-03 | On success, the API issues a short-lived access token (JWT, 15 minutes) and an opaque refresh token (7 days). Only a hash of the refresh token is stored. |
| FR-AUTH-04 | Each login creates a session that records device name, user agent, IP address, created time, and last-seen time. |
| FR-AUTH-05 | Refresh tokens rotate on every use. Reusing an already-used refresh token revokes the whole session family (theft detection). |
| FR-AUTH-06 | Customer can list active sessions, revoke one session, or log out of all sessions. |
| FR-AUTH-07 | Login from a new device sends a security notification. |
| FR-AUTH-08 | Password reset uses a single-use emailed token (30-minute expiry). A successful reset revokes all sessions. |
| FR-AUTH-09 | Password change requires the current password and revokes all other sessions. |
| FR-AUTH-10 | TOTP multi-factor authentication. It is optional for customers and mandatory for all staff roles. Recovery codes are provided. |
| FR-AUTH-11 | Transaction PIN is required for money-moving operations. 3 wrong PIN attempts lock money movement for 30 minutes. PIN change requires password. |

### 5.4 Authorization (AUTHZ)

| ID           | Requirement |
|--------------|-------------|
| FR-AUTHZ-01 | Role-based access control. Roles map to permissions (for example `accounts:read:own`, `kyc:review`, `ledger:adjust`). |
| FR-AUTHZ-02 | Ownership checks: a customer can only access their own accounts, transactions, beneficiaries, and documents. |
| FR-AUTHZ-03 | Maker-checker for high-risk staff actions: manual ledger adjustments, unfreezing an account flagged for fraud, and changing limits. The maker cannot be the checker. |
| FR-AUTHZ-04 | Every authorization denial is written to the audit log. |

### 5.5 Accounts (ACC)

| ID         | Requirement |
|------------|-------------|
| FR-ACC-01 | Account types: savings, current. Fixed deposit is a stretch goal. |
| FR-ACC-02 | Currencies: NGN (primary) and USD. An account has exactly one currency. |
| FR-ACC-03 | Account numbers are 10 digits with a check digit (simplified NUBAN) and are unique. |
| FR-ACC-04 | A customer can open additional accounts, with a maximum of 5. |
| FR-ACC-05 | Account statuses: `active`, `dormant` (no customer-initiated activity for 12 months), `frozen`, `closed`. Frozen accounts reject debits. Closed accounts reject everything. |
| FR-ACC-06 | Balance endpoint returns ledger balance and available balance. |
| FR-ACC-07 | An account can be closed only when its balance is zero and it has no pending transactions or holds. |
| FR-ACC-08 | Name enquiry: given an account number, return the masked account name (for transfer confirmation). |

### 5.6 Ledger (LED)

| ID         | Requirement |
|------------|-------------|
| FR-LED-01 | Double-entry bookkeeping. Every journal entry has at least two postings, and its postings sum to zero per currency. This is enforced in code and by a database constraint or trigger. |
| FR-LED-02 | Journal entries and postings are immutable. Corrections are made by reversal entries, never by updates or deletes. |
| FR-LED-03 | Internal bank accounts (general ledger accounts) exist for cash vault, fee income, interest expense, settlement, and suspense. |
| FR-LED-04 | A cached balance is kept on each account and updated in the same database transaction as the postings. |
| FR-LED-05 | A reconciliation job recomputes balances from postings and reports any mismatch. |
| FR-LED-06 | Holds (liens) can be placed and released, and they reduce the available balance. |

### 5.7 Money Movement (TXN)

| ID         | Requirement |
|------------|-------------|
| FR-TXN-01 | **Cash deposit** is recorded by a teller for a customer account. |
| FR-TXN-02 | **Inbound transfer** arrives through a simulated payment gateway webhook, verified with an HMAC-SHA256 signature and a timestamp that prevents replay. |
| FR-TXN-03 | **Cash withdrawal** is recorded by a teller and needs customer PIN confirmation (simulated). |
| FR-TXN-04 | **Internal transfer** between Go Bank accounts is instant and atomic. |
| FR-TXN-05 | **External transfer** to another bank goes through a simulated interbank switch. It is asynchronous with states `pending` -> `successful` or `failed`. Failed transfers are auto-reversed. |
| FR-TXN-06 | Every money-moving request requires an `Idempotency-Key` header. Replaying the same key with the same body returns the original response. Reusing a key with a different body is rejected with 422. |
| FR-TXN-07 | Every transfer checks: account status, sufficient available balance, KYC tier limits, daily cumulative limit, fraud rules, and transaction PIN. |
| FR-TXN-08 | Fees are configurable per transaction type and amount band, and they are posted as separate ledger lines. |
| FR-TXN-09 | Every transaction has a unique reference, a narration (max 100 chars), and a status. |
| FR-TXN-10 | Concurrent transfers touching the same accounts never deadlock and never overdraw. Rows are locked in a consistent order. |
| FR-TXN-11 | A customer can raise a dispute on a transaction. Support tracks it through `open` -> `investigating` -> `resolved` \| `rejected`. |

### 5.8 Beneficiaries and Scheduled Payments (BEN, SCH)

| ID         | Requirement |
|------------|-------------|
| FR-BEN-01 | Customer can save, list, rename, and delete beneficiaries (internal or external). |
| FR-BEN-02 | First transfer to a newly added beneficiary above a threshold is held for fraud review (cooling-off period). |
| FR-SCH-01 | Customer can schedule a one-time future transfer or a recurring transfer (daily, weekly, monthly), with an end date or a count. |
| FR-SCH-02 | A scheduler executes due transfers. Failures are retried with backoff, and the customer is notified. |

### 5.9 History, Statements, Interest (HIS, STM, INT)

| ID         | Requirement |
|------------|-------------|
| FR-HIS-01 | Transaction history with filters (date range, type, direction, amount range) and cursor-based pagination. |
| FR-STM-01 | Customer requests an account statement for a date range. It is generated asynchronously (CSV required, PDF stretch) and can be downloaded when ready. |
| FR-INT-01 | Interest on savings accrues daily (annual rate configurable, computed on the end-of-day balance) and is paid monthly through ledger postings. |

### 5.10 Fraud and Risk (FRD)

| ID         | Requirement |
|------------|-------------|
| FR-FRD-01 | A rules engine evaluates each debit and returns `allow`, `review`, or `block` with the reasons. |
| FR-FRD-02 | Initial rules: velocity (more than N transfers in M minutes), large amount relative to the customer's average, new beneficiary with a large amount, activity at unusual hours, and a new device combined with a large amount. |
| FR-FRD-03 | Rules run concurrently with a strict time budget. If the budget is exceeded, the outcome is `review` (fail safe). |
| FR-FRD-04 | Compliance officers work a queue of flagged transactions and can release or reject them. |

### 5.11 Notifications (NTF)

| ID         | Requirement |
|------------|-------------|
| FR-NTF-01 | Email is sent through Resend in staging and production. SMS is sent through a pluggable provider (Twilio Verify or Termii). In local development, both channels use a console driver that writes the message to the logs, so no credits are spent. Configuration selects the driver. |
| FR-NTF-05 | Phone verification by SMS OTP is required at onboarding (FR-ONB-02), when the phone number changes, and as a step-up check for sensitive actions (login from a new device, PIN reset, adding a beneficiary). |
| FR-NTF-02 | In-app notifications can be listed and marked as read. |
| FR-NTF-03 | Notifications are produced from domain events through the transactional outbox and delivered by background workers with retries. |
| FR-NTF-04 | Credit and debit alerts are sent for every completed transaction. |

### 5.12 Audit and Back Office (AUD, ADM)

| ID         | Requirement |
|------------|-------------|
| FR-AUD-01 | Every sensitive action is written to an append-only audit log: actor, role, action, target, before and after (with PII masked), IP, request ID, and timestamp. |
| FR-AUD-02 | Audit records are hash-chained (each record stores the hash of the previous one) so tampering can be detected. A verify job checks the chain. |
| FR-ADM-01 | Staff can search customers by name, email, phone, account number, or BVN (through a blind index). |
| FR-ADM-02 | Staff can freeze and unfreeze accounts with a reason. |
| FR-ADM-03 | Admin manages staff users and roles. The first admin is created by a CLI bootstrap command, never through the public API. |
| FR-ADM-04 | Basic reports: daily transaction volume, new customers, KYC queue size, flagged transactions. |

### 5.13 Loans (LOAN) - Stretch

| ID          | Requirement |
|-------------|-------------|
| FR-LOAN-01 | Customer applies for a personal loan. Eligibility is based on tier, account age, and inflow history. |
| FR-LOAN-02 | Approved loans are disbursed through the ledger, and a repayment schedule (amortized) is generated. |
| FR-LOAN-03 | Auto-debit on repayment dates, with late fees and notifications. |

---

## 6. Non-Functional Requirements

| ID       | Category        | Requirement |
|----------|-----------------|-------------|
| NFR-01 | Correctness | Money is stored as `int64` minor units, never as floats. All money operations run inside database transactions. |
| NFR-02 | Consistency | Choose correctness over availability for money paths. Use row-level locks (`SELECT ... FOR UPDATE`) in a deterministic order. |
| NFR-03 | Performance | Reads have p95 under 100 ms locally and internal transfers have p95 under 200 ms at 200 concurrent requests. |
| NFR-04 | Resilience | Every request has a context deadline. Outbound calls have timeouts, retries with exponential backoff and jitter, and a circuit breaker. |
| NFR-05 | Shutdown | Graceful shutdown lets in-flight requests and jobs finish within 30 seconds on SIGINT and SIGTERM. |
| NFR-06 | Observability | Structured JSON logs (`log/slog`) carry the request ID and user ID, and PII is masked. Prometheus metrics are exposed at `/metrics`. `pprof` is available on an internal port. |
| NFR-07 | Configuration | Configuration comes from environment variables following 12-factor. The app validates it at startup and fails fast. `.env.example` is committed and `.env` is never committed. |
| NFR-08 | Quality | Service and domain packages have at least 80% test coverage. `go test -race` passes. `golangci-lint` passes. |
| NFR-09 | API | Versioned under `/api/v1`. Errors follow a single JSON format. Lists use cursor pagination. |
| NFR-10 | Scalability | The API is stateless, so any instance can serve any request. Workers scale horizontally. Jobs use `SELECT ... FOR UPDATE SKIP LOCKED` so two instances never process the same job. |
| NFR-11 | Rate limiting | Limits apply per IP and per user, using a token bucket. Login, OTP, and PIN endpoints have stricter limits. |
| NFR-12 | Documentation | OpenAPI 3.1 spec and Postman collection are updated in the same pull request as the code change. |

---

## 7. Security and Data Protection

| ID       | Requirement |
|----------|-------------|
| SEC-01 | Passwords and PINs are hashed with Argon2id, using separate parameters and a unique salt per hash. |
| SEC-02 | PII fields (BVN, NIN, ID numbers, date of birth, address) are encrypted at the application layer with AES-256-GCM. |
| SEC-03 | Every ciphertext stores a key version. Keys can be rotated, and a re-encryption job migrates old records. |
| SEC-04 | Searchable encrypted fields use a blind index (HMAC-SHA256 with a separate key) for exact-match lookup. |
| SEC-05 | Secrets (DB password, JWT signing key, encryption keys) come from the environment and never appear in logs or Git. |
| SEC-06 | All SQL is parameterized (sqlc-generated). No string-built queries. |
| SEC-07 | Security headers, strict CORS allowlist, and a request body size limit are applied. |
| SEC-08 | Uploaded files are stored outside the web root with random names, behind a storage interface (local disk now, S3-compatible later). |
| SEC-09 | Responses never leak internal errors or stack traces. Every error response includes a request ID for support tracing. |
| SEC-10 | Log masking: emails partially masked, phone numbers show the last 4 digits only, and identity numbers never appear in logs. |
| SEC-11 | Webhook signatures are verified with constant-time comparison. |
| SEC-12 | Dependencies are scanned with `govulncheck` in CI. |

---

## 8. Architecture and Technology

### 8.1 Style

This is a **modular monolith**. There is one deployable API binary and one worker binary. Code is organized by business domain, and each domain has clear layers:

```
HTTP handler  ->  Service (business rules)  ->  Repository (data access)  ->  PostgreSQL
                         |
                         +-> other domain services through interfaces
                         +-> outbox events -> workers -> notifications, external calls
```

Rules:
- Handlers do only HTTP work: decode, validate the shape, call the service, and encode the response.
- Services hold business rules and do not know about HTTP.
- Repositories hold SQL and do not know about business rules.
- Dependencies point inward and are injected through constructors. There are no global variables except in `main`.
- External providers (identity, sanctions, interbank switch, email, SMS, storage) are interfaces. Identity, sanctions, and the interbank switch have simulated implementations. Email (Resend) and SMS (Twilio or Termii) have real implementations, plus a console implementation for local development.

### 8.2 Technology Choices

Standard library first. Each third-party dependency must be justified in an ADR (Architecture Decision Record).

| Concern            | Choice                                   | Reason |
|--------------------|------------------------------------------|--------|
| Language           | Go 1.26                                  | Current stable. |
| HTTP routing       | `net/http` ServeMux (method and path patterns) | The standard library has been enough since Go 1.22, and it teaches the fundamentals. |
| Database           | PostgreSQL (Postgres.app locally)        | ACID, row locking, and it is standard for fintech. |
| Driver             | `jackc/pgx/v5` with `pgxpool`            | Fastest and most complete Postgres driver for Go. |
| Queries            | `sqlc`                                   | Type-safe Go code generated from real SQL. |
| Migrations         | `golang-migrate`                         | Versioned, reversible, and widely used. |
| Logging            | `log/slog`                               | Structured logging in the standard library. |
| Tokens             | `golang-jwt/jwt/v5`                      | Industry standard for JWTs. |
| Crypto             | `crypto/*`, `golang.org/x/crypto/argon2` | Standard library and official extended packages. |
| TOTP               | `pquerna/otp`                            | RFC 6238 implementation. |
| Testing            | `testing`, `httptest`, `go-cmp`          | Standard library first. |
| Lint               | `golangci-lint`                          | Standard aggregate linter. |
| Email              | Resend (REST API through `net/http`)     | Already available. Calling the API with the standard library teaches how to build HTTP clients. |
| SMS                | Twilio Verify or Termii (trial credits), console driver locally | No reliable, permanently free SMS provider exists. An interface keeps the provider swappable. |
| Build tasks        | `Makefile`                               | One command for every workflow. |
| CI                 | GitHub Actions                           | Runs lint, test with race, vulncheck, and Newman (Postman) tests. |
| API docs           | OpenAPI 3.1 (hand-written) and Postman   | Spec-first discipline. |

---

## 9. Data Model (High Level)

Main tables. The full schema lives in `db/migrations`.

- `users` (identity, credentials, role, status, lockout fields)
- `applications`, `application_checks`, `kyc_documents`, `consents`
- `customers` (encrypted PII, blind indexes, KYC tier)
- `staff_profiles`, `roles`, `permissions`, `role_permissions`
- `sessions`, `refresh_tokens`, `otp_codes`, `password_reset_tokens`, `mfa_secrets`
- `accounts` (number, type, currency, status, cached balance, version)
- `journal_entries`, `postings`, `holds`
- `transactions` (customer-facing view of money movement, status, reference, fees)
- `idempotency_keys` (key, user, request hash, response snapshot, expiry)
- `beneficiaries`, `scheduled_transfers`
- `fraud_evaluations`, `disputes`
- `outbox_events`, `jobs`
- `notifications`
- `statements`
- `audit_logs`
- `approvals` (maker-checker requests)

Conventions: UUIDv7 primary keys (time ordered), `created_at` and `updated_at` as `timestamptz` in UTC, soft delete only where the law requires retention, and money columns as `bigint` with a paired `currency` column.

---

## 10. API Conventions

- Base path: `/api/v1`
- JSON only, `snake_case` fields.
- Authentication: `Authorization: Bearer <access_token>`.
- Idempotency: `Idempotency-Key: <uuid>` is required on money-moving POST requests.
- Request tracing: `X-Request-ID` is accepted or generated, and echoed back.
- Pagination: `?limit=20&cursor=<opaque>`. The response has `data` and `next_cursor`.
- Success envelope: `{ "data": ... }`
- Error envelope:

```json
{
  "error": {
    "code": "INSUFFICIENT_FUNDS",
    "message": "Available balance is not enough for this transfer.",
    "request_id": "01J8Z...",
    "details": [{ "field": "amount", "issue": "exceeds available balance" }]
  }
}
```

- Status codes: 200, 201, 202 (accepted for async), 400, 401, 403, 404, 409 (conflict or duplicate), 422 (business rule violated), 423 (locked), 429, 500, 503.

---

## 11. Repository Structure

```
go-bank/
  cmd/
    api/                 # HTTP API entry point (main.go)
    worker/              # Background worker entry point
    gobankctl/           # Admin CLI (bootstrap admin, rotate keys, reconcile)
  internal/
    app/                 # Wiring: build dependencies, start and stop servers
    config/              # Load and validate environment configuration
    platform/            # Cross-cutting technical packages (no business rules)
      database/          # pgx pool, transaction helper
      httpx/             # JSON helpers, error mapping, middleware
      logger/            # slog setup, PII masking
      crypto/            # AES-GCM, Argon2id, HMAC, key ring
      validate/          # Generic validation helpers
      clock/             # Time abstraction for testable code
      ratelimit/         # Token bucket limiter
      storage/           # File storage interface and local implementation
      worker/            # Worker pool, job runner, scheduler
    domain packages (each: handler.go, service.go, repository.go, model.go, errors.go, *_test.go)
    onboarding/
    kyc/
    auth/
    authz/
    customer/
    account/
    ledger/
    transfer/
    beneficiary/
    fraud/
    notification/
    statement/
    audit/
    admin/
    providers/           # Simulated external providers (identity, sanctions, switch, email, sms)
  db/
    migrations/          # Versioned SQL migrations (up and down)
    queries/             # SQL used by sqlc
    sqlc.yaml
  api/
    openapi.yaml         # Source of truth for the HTTP contract
  postman/
    go-bank.postman_collection.json
    local.postman_environment.json
  docs/
    PRD.md
    adr/                 # Architecture Decision Records
    runbooks/            # How to operate: key rotation, reconciliation
  scripts/
  testdata/
  .github/workflows/ci.yml
  .env.example
  .gitignore
  .golangci.yml
  Makefile
  README.md
  CHANGELOG.md
  go.mod
```

Why `internal/`: the Go toolchain forbids importing these packages from outside the module, which enforces encapsulation. Why package by domain: related code changes together, and each domain can later be extracted into a service if needed.

---

## 12. Engineering Workflow

### 12.1 Git

- `main` always builds and passes tests. Nobody commits directly to `main`.
- One branch per unit of work: `feat/auth-refresh-rotation`, `fix/transfer-deadlock`, `chore/ci-cache`, `docs/openapi-transfers`.
- Commit messages follow Conventional Commits: `type(scope): summary`, for example `feat(ledger): enforce zero-sum postings [FR-LED-01]`.
- Commits are small and atomic, and each one leaves the build green.
- Every change goes through a pull request with a description, a checklist, and linked requirement IDs. Merges are squash merges.
- Releases use semantic version tags (`v0.1.0`), and `CHANGELOG.md` is updated for each release.

### 12.2 Testing Strategy

| Level         | Scope                                          | Tooling |
|---------------|------------------------------------------------|---------|
| Unit          | Pure logic: money, NUBAN, fraud rules, validation | `testing`, table-driven, subtests |
| Fuzz          | Parsers and check-digit algorithms             | `go test -fuzz` |
| Integration   | Repositories and services against a real test database | Separate `gobank_test` database, transaction rollback per test |
| HTTP          | Handlers and middleware                        | `httptest` |
| Concurrency   | Parallel transfers, rate limiter, workers      | `-race`, goroutine fan-out tests |
| Benchmark     | Hot paths (hashing, ledger posting)            | `testing.B` |
| API / E2E     | Full flows against a running server            | Postman collection with Newman in CI |

### 12.3 Documentation

- `api/openapi.yaml` is the contract, updated in the same pull request as the code.
- The Postman collection has folders per domain, pre-request scripts for auth and idempotency keys, and test assertions on every request.
- ADRs record significant decisions (for example, why JWT plus opaque refresh tokens, or why sqlc).
- The README covers setup in under 10 minutes on a fresh machine.

### 12.4 Definition of Done (every feature)

1. The code follows the layering rules.
2. Unit and integration tests are written and pass with `-race`.
3. Lint passes.
4. OpenAPI and Postman are updated, and the Postman tests pass.
5. Audit logging is added for sensitive actions.
6. The pull request is reviewed and merged with a Conventional Commit title.

---

## 13. Go Learning Map

| Go concept                                   | Where it is used in Go Bank |
|----------------------------------------------|------------------------------|
| Modules, packages, visibility, `internal/`   | Project setup, package layout |
| Structs, methods, constructors               | Every service and repository |
| Interfaces (small, consumer-defined)         | Repositories, providers, storage, clock |
| Errors: sentinel, wrapping, `errors.Is/As`, custom types | Domain errors mapped to HTTP responses |
| `defer`, `panic`, `recover`                  | Transaction rollback, recovery middleware |
| Closures and higher-order functions          | Middleware chains, transaction helper `WithTx(ctx, fn)` |
| `context.Context`                            | Deadlines, cancellation, request-scoped values |
| Goroutines and channels                      | Worker pool, notification dispatch, fraud rule fan-out |
| `select`, `time.Ticker`, `time.After`        | Scheduler, timeouts, graceful shutdown |
| `sync.Mutex`, `sync.RWMutex`                 | Rate limiter buckets, in-memory caches, key ring |
| `sync.WaitGroup`, `errgroup`                 | Concurrent onboarding checks, statement generation |
| `sync.Once`, `sync/atomic`                   | Lazy initialization, metrics counters |
| Generics (type parameters, constraints)      | `Page[T]`, `Cache[K comparable, V any]`, validation helpers, `Result[T]` |
| Iterators (`iter.Seq`, range over func)      | Streaming postings for reconciliation and statements |
| `embed`                                      | Embedding migrations and email templates |
| `io.Reader` and `io.Writer` composition      | File upload hashing, CSV statement streaming |
| Struct tags and light reflection             | JSON encoding, config loading |
| Testing: table-driven, subtests, fuzzing, benchmarks, `-race` | Throughout |
| Profiling with `pprof`                       | Performance hardening phase |
| Build tags, `-ldflags` version injection     | Release builds |

---

## 14. Milestones

Each phase ends with a merged pull request, passing tests, and updated docs. We do not move to the next phase until the current one is understood.

| Phase | Name                          | Key deliverables | Main Go concepts |
|-------|-------------------------------|------------------|------------------|
| 0  | Foundations                  | Repo, Git, module, Makefile, config, logger, HTTP server, health check, graceful shutdown | Packages, structs, errors, `context`, signals, goroutines |
| 1  | Database layer               | Postgres setup, migrations, sqlc, pgx pool, transaction helper | Interfaces, closures, `defer`, error wrapping |
| 2  | HTTP platform                | JSON helpers, error envelope, middleware (request ID, logging, recovery), validation | Higher-order functions, generics, `recover` |
| 3  | Security core                | Argon2id, AES-GCM key ring, blind index, PII masking | `crypto`, byte slices, `sync.RWMutex` |
| 4  | Onboarding                   | Application flow, OTP, personal details, identity check provider | State machines, interfaces, table tests |
| 5  | KYC documents and review     | Uploads, storage interface, concurrent checks, review workflow, tiers | `io` composition, `errgroup`, `WaitGroup` |
| 6  | Authentication and sessions  | Login, lockout, JWT, refresh rotation, sessions, reset, MFA, PIN | Time, crypto randomness, middleware |
| 7  | Authorization                | RBAC, ownership, maker-checker | Context values, closures |
| 8  | Accounts and ledger          | NUBAN, account lifecycle, double-entry postings, holds | Fuzzing, invariants, DB constraints |
| 9  | Money movement               | Deposits, withdrawals, internal transfers, idempotency, lock ordering | Concurrency testing, `-race`, transactions |
| 10 | Limits, fees, fraud          | Tier limits, fee engine, rules engine with time budget | Generics, goroutine fan-out, `select` |
| 11 | Async processing             | Outbox, worker pool, notifications, external transfers, scheduled transfers, retries | Channels, worker pools, backoff, `SKIP LOCKED` |
| 12 | History, statements, interest| Cursor pagination, CSV statements, interest accrual batch | Iterators, `io.Writer`, batch processing |
| 13 | Audit and back office        | Hash-chained audit log, admin endpoints, CLI bootstrap | Hashing, CLI with `flag` |
| 14 | Observability and hardening  | Metrics, pprof, rate limiting, load test, profiling fixes | `atomic`, `pprof`, benchmarks |
| 15 | CI/CD and release            | GitHub Actions, Docker, Newman, versioned release | Build flags, `ldflags` |
| 16 | Loans (stretch)              | Loan products, amortization, auto-debit | Everything combined |

---

## 15. Open Questions

1. Resend needs a verified domain to send to any recipient. Without one, it only delivers to the account owner's email address. Is there a domain we can verify?
2. Which SMS provider should we use for real delivery: Twilio Verify or Termii?
3. Should USD accounts be in scope from Phase 8, or added later?
4. Is PDF statement generation required, or is CSV enough?

---

## 16. Revision History

| Version | Date       | Change        |
|---------|------------|---------------|
| 0.1.0   | 2026-09-23 | Initial draft |
| 0.2.0   | 2026-09-23 | Module path set to `github.com/Olamilekan-12/go-bank`. Added Resend for email and an SMS provider for phone verification (FR-NTF-01, FR-NTF-05). |
