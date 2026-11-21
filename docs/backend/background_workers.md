# Concurrency & Background Worker Architecture

**Document Type:** Systems Engineering & Concurrency Architecture  
**Target Audience:** Backend Developers, Infrastructure Architects, and Site Reliability Engineers  
**Pattern:** Zero-Redis Native Concurrency via Goroutines, Channels, and Tickers

---

## 1. Concurrency Philosophy: The Zero-Redis Advantage

For mid-sized production systems, introducing an external job queue (Redis, RabbitMQ, Kafka) introduces unnecessary operational complexity:

- Additional infrastructure instances to provision, monitor, and back up.
- Network latency overhead between the application and the broker.
- Potential split-brain states and serializing/deserializing overhead.

The **Go REST API Starter Kit** leverages Go's native runtime primitives (`goroutines`, `time.Ticker`, `select`, and `context.Context`) to run periodic background workers within the same binary process with **zero external queue dependencies**.

---

## 2. Worker Roster & Operational Lifecycles

All workers are registered during the composition root bootstrap in `cmd/server/main.go` and receive a shared parent context and `sync.WaitGroup`.

```text
[ cmd/server/main.go ]
       │
       ├──► 1. Token Pruning Worker (Interval: 24 Hours)
       │       └── Deletes expired refresh tokens & used auth tokens
       │
       ├──► 2. Article Publisher Worker (Interval: 1 Minute)
       │       └── Publishes scheduled articles (scheduled_at <= NOW())
       │
       └──► 3. Daily Analytics Aggregator & Retention Worker (Interval: 1 Hour)
               ├── Aggregates daily engagement metrics (views, readers, shares)
               └── Prunes raw events > 30 days in bounded batches of 5,000 rows
```

### 2.1 Token Pruning Worker

- **Location:** `cmd/server/main.go` (`startTokenCleanup`)
- **Execution Interval:** Every **24 hours** (plus an immediate run on application startup).
- **Responsibility:**
  1. Purges revoked or expired refresh tokens from `refresh_tokens`.
  2. Purges consumed or expired one-time tokens (email verification, password reset) from `auth_tokens`.
- **Query:** `DELETE FROM refresh_tokens WHERE expires_at < NOW() OR is_revoked = TRUE`.

### 2.2 Scheduled Article Publisher Worker

- **Location:** `internal/worker/publisher_worker.go` (`StartPublisherWorker`)
- **Execution Interval:** Every **1 minute**.
- **Responsibility:**
  - Queries articles with `status = 'scheduled'` and `scheduled_at <= NOW()`.
  - Atomically transitions their status to `published` and sets `published_at = scheduled_at`.

### 2.3 Analytics Aggregation & Retention Pruning Worker

- **Location:** `internal/worker/analytics_worker.go` (`StartAnalyticsWorker`)
- **Execution Interval:** Every **1 hour**.
- **Dual Responsibilities:**
  1. **Metric Aggregation**: Scans `article_analytics_events` for recent event rows using SARGable timestamp bounds, computes daily totals (`views`, `unique_visitors`, `reading_time`), and upserts into `article_daily_analytics`.
  2. **Bounded Retention Pruning**: Deletes raw events older than **30 days** in discrete batches of **5,000 rows**:
     ```sql
     DELETE FROM article_analytics_events
     WHERE created_at < ?
     LIMIT 5000;
     ```
     > [!TIP]
     > **Lock Contention Prevention**: Pruning in 5,000-row chunks prevents long-running row/table locks on InnoDB, ensuring high-frequency live tracking ingestion is never blocked.

---

## 3. Graceful Shutdown Protocol

To prevent corrupted database states or abruptly terminated background operations during deployments:

```go
// 1. Trap OS interrupt signals
shutdownChan := make(chan os.Signal, 1)
signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

// 2. Wait for signal
<-shutdownChan
slog.Info("Shutting down gracefully...")

// 3. Cancel shared worker context
cancelWorkers()

// 4. Wait for all active worker loops to exit cleanly
workersWg.Wait()

// 5. Gracefully shutdown HTTP server with timeout
ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
defer cancelShutdown()
server.Shutdown(ctxShutdown)
```

Each worker monitors `ctx.Done()`. When shutdown is initiated:

1. The ticker stops immediately.
2. Any in-flight batch completes its current transaction.
3. The goroutine decrements the `WaitGroup` counter via `defer wg.Done()`.
