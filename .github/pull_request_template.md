<!-- ========================================================================= -->
<!-- ENTERPRISE PULL REQUEST TEMPLATE (Go REST API Starter Kit)               -->
<!-- Title Convention: <type>(<scope>): <short description>                    -->
<!-- Examples: feat(articles): add scheduled publishing support                -->
<!--           fix(auth): prevent token family desync on multi-refresh         -->
<!-- ========================================================================= -->

## 1. Summary of Changes

### What does this PR do?

<!-- Provide a clear, concise summary of the feature, fix, or enhancement. -->

### Why is this change needed?

<!-- Business context, user impact, or technical motivation. -->

### Related Issue / Ticket

<!-- Link Jira ticket or GitHub issue: Closes #123, Relates to ENG-456 -->

- Resolves:

---

## 2. Type of Change

- [ ] `feat` - New feature or capability (non-breaking change adding functionality)
- [ ] `fix` - Bug fix (non-breaking change fixing an issue)
- [ ] `refactor` - Code restructuring that neither fixes a bug nor adds a feature
- [ ] `perf` - Performance optimization (query tuning, caching, memory allocation)
- [ ] `migration` - Database schema change (DDL) or SQLC query update
- [ ] `security` - Security hardening, authorization fix, or vulnerability patch
- [ ] `docs` - Documentation updates or corrections
- [ ] `chore` - Tooling, CI/CD, dependency updates, or internal configuration
- [ ] **BREAKING CHANGE** - Modifies API contracts, database schema, or behavior in a backward-incompatible way

---

## 3. Clean Architecture Layer Impact

Check all layers modified in this PR:

- [ ] **Domain Models & DTOs** (`internal/models/`): Entity structs, input validation (`Validate()`).
- [ ] **Persistence & SQLC** (`db/migrations/`, `db/queries/`, `internal/repository/`): Schema, SQL queries, repository interfaces.
- [ ] **Business Logic & Services** (`internal/service/`): Domain invariants, orchestration, Argon2id, token handling.
- [ ] **Presentation & Handlers** (`internal/handlers/`): HTTP decoding, status mapping, envelope formatting.
- [ ] **Routing & Middleware** (`internal/routes/`, `internal/middleware/`): Route registration, RBAC guards, rate limits.
- [ ] **Background Workers** (`internal/worker/`, `cmd/server/main.go`): Goroutine workers, tickers, graceful shutdown.
- [ ] **Infrastructure Adapters** (`internal/mailer/`, `internal/storage/`, `internal/config/`): S3/R2, Resend, env loading.

---

## 4. Database & SQLC Verification (If Applicable)

- [ ] Sequential migration files created in `db/migrations/` (`<ver>_<name>.up.sql` and `<ver>_<name>.down.sql`).
- [ ] Down migration (`.down.sql`) tested and cleanly reverses the forward migration.
- [ ] SQLC queries compiled via `sqlc generate` with zero compilation errors.
- [ ] Queries adhere to indexing standards (SARGable range comparisons, composite index matching).
- [ ] Database seeder updated (`internal/database/seeder.go`) if new roles, permissions, or master data are introduced.
- [ ] Tested with `-fresh` flag locally (`go run cmd/server/main.go -fresh`).
- [ ] **Not Applicable** (no database or migration changes).

---

## 5. Security & Authorization Review

- [ ] All request payloads validated using multi-input field validation without early return.
- [ ] Route parameters strictly validated using `utils.IsValidULID` for 26-char Crockford Base32 identifiers.
- [ ] Appropriate RBAC middleware applied (`RequireRole` or `RequirePermission`).
- [ ] Sensitive operations (passwords, tokens, credentials) masked from logs and responses.
- [ ] Request body size strictly limited (`http.MaxBytesReader` applied).
- [ ] Super Admin Safeguard respected (cannot delete/demote the last active super admin).

---

## 6. Testing & Quality Assurance

- [ ] Unit tests added or updated in `internal/**/*_test.go` using in-memory mock repositories.
- [ ] Full test suite passes locally: `go test -v -count=1 ./...`
- [ ] Race condition check passes: `go test -race ./...`
- [ ] Manual end-to-end verification executed via `api.http` or Postman collection (`docs/tools/postman_collection.json`).
- [ ] Documentation updated in `docs/` (API contracts, data dictionary, or onboarding guides).

---

## 7. Deployment & Rollback Plan

### Environment Variables:

<!-- List any new or modified environment variables in .env.example -->

- [ ] No new environment variables required.
- [ ] New variables introduced (added to `.env.example` and `docs/devops/production_deployment.md`):

### Rollback Strategy:

<!-- What are the steps to safely roll back if issues arise in production? -->

1. Revert deploy commit / redeploy previous Docker container tag.
2. Execute reverse migration if DDL was applied: `migrate -path db/migrations -database ... down 1`.
