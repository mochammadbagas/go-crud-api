# Database Engineering, SQLC, & Migrations

**Document Type:** Database Architecture & Tooling Reference  
**Target Audience:** Backend Developers & Database Administrators  
**Database Engine:** MySQL 8.0+ (InnoDB, `utf8mb4_0900_ai_ci`)

---

## 1. Identity Design: ULID Standard

All relational primary keys and foreign keys in this project standardize on **26-character Crockford Base32 Universally Unique Lexicographically Sortable Identifiers (ULID)** stored as `CHAR(26)`:

```sql
id CHAR(26) NOT NULL PRIMARY KEY
```

### Advantages over Alternatives:

- **Vs Auto-Increment (`BIGINT`)**: Eliminates ID enumeration scraping vulnerabilities, prevents sequence lock contention in distributed setups, and avoids 53-bit integer truncation issues in client JavaScript environments (`Number.MAX_SAFE_INTEGER`).
- **Vs UUIDv4**: ULIDs contain a 48-bit millisecond UNIX timestamp prefix. This ensures chronological sorting (`ORDER BY id DESC`) and avoids MySQL InnoDB B-tree clustering page fragmentation caused by completely random UUIDs.

---

## 2. Schema Migration Workflow (`golang-migrate`)

Migrations are located in `db/migrations/` and embedded directly into the Go binary at compile time via `db/embed.go`:

```go
// db/embed.go
package db

import "embed"

//go:embed migrations/*.sql
var MigrationsFS embed.FS
```

### 2.1 Adding a New Migration File

Migration filenames must follow the zero-padded 6-digit versioning pattern:

```text
db/migrations/000003_create_products.up.sql
db/migrations/000003_create_products.down.sql
```

1. **Write Forward DDL (`.up.sql`)**:
   ```sql
   CREATE TABLE IF NOT EXISTS products (
       id CHAR(26) NOT NULL,
       name VARCHAR(150) NOT NULL,
       price BIGINT NOT NULL DEFAULT 0,
       created_at DATETIME NOT NULL,
       updated_at DATETIME NOT NULL,
       PRIMARY KEY (id)
   ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
   ```
2. **Write Reversal DDL (`.down.sql`)**:
   ```sql
   DROP TABLE IF EXISTS products;
   ```

### 2.2 Migration Execution Lifecycle

- When `cmd/server/main.go` starts, `database.Migrate(db)` executes all pending `.up.sql` scripts sequentially inside an isolated schema transaction.
- When passing `-fresh`, `database.MigrateFresh(db)` drops all existing tables and re-executes all migrations from version `000001`.

---

## 3. SQLC Query Compilation

Instead of traditional reflection-heavy ORMs (like GORM), this project uses **SQLC** to compile raw SQL into type-safe, boilerplate-free Go code.

### 3.1 SQL Query File Conventions

Create or edit `.sql` files in `db/queries/`:

```sql
-- name: GetProductByID :one
SELECT id, name, price, created_at, updated_at
FROM products
WHERE id = ? LIMIT 1;

-- name: ListActiveProducts :many
SELECT id, name, price, created_at, updated_at
FROM products
ORDER BY id DESC
LIMIT ? OFFSET ?;

-- name: CreateProduct :exec
INSERT INTO products (id, name, price, created_at, updated_at)
VALUES (?, ?, ?, ?, ?);
```

### 3.2 SQLC Annotations Reference

| Annotation    | Return Signature      | Use Case                                  |
| :------------ | :-------------------- | :---------------------------------------- |
| `:one`        | `(T, error)`          | Single row queries (by ID, slug, email)   |
| `:many`       | `([]T, error)`        | Multi-row listing queries with pagination |
| `:exec`       | `error`               | INSERT, UPDATE, DELETE statements         |
| `:execresult` | `(sql.Result, error)` | Statements requiring `RowsAffected()`     |

### 3.3 Running the Compiler

Execute SQLC from the workspace root:

```bash
sqlc generate
```

Generated structs and methods are written directly to `internal/repository/sqlc/`.

---

## 4. Query Performance & Indexing Standards

To maintain sub-10ms query execution times under load:

1. **SARGable Timestamp Queries**: Avoid wrapping indexed columns in SQL functions. Use range comparisons:

   ```sql
   -- RECOMMENDED (SARGable, hits index)
   WHERE created_at >= ? AND created_at < ?

   -- AVOID (Non-SARGable, forces full table scan)
   WHERE DATE(created_at) = ?
   ```

2. **Composite B-Tree Indexes**: As demonstrated in `000002_add_performance_indexes.up.sql`, multi-column queries must have matching composite indexes matching the equality-then-range rule:
   - Index: `(status, published_at, deleted_at)` on `articles`.
   - Index: `(article_id, analytics_date)` on `article_daily_analytics`.
