# Production Deployment & Operations Guide

**Document Type:** Production Readiness, Infrastructure, & Deployment Guide  
**Target Audience:** DevOps Engineers, System Administrators, and Backend Developers  
**Environments:** Docker, VPS (Ubuntu/Debian), Cloudflare R2, AWS S3, Resend

---

## 1. Production Readiness Checklist

Before exposing the API to public traffic, ensure the following items are verified:

- [ ] **Generate High-Entropy Secret Key**: Set `JWT_SECRET` to a 64+ character random string (minimum 32 characters enforced by application fail-fast validation in `APP_ENV=production`).
- [ ] **Configure Production Database**: Enable SSL/TLS encryption for database connections (`tls=true`).
- [ ] **Enforce CORS Allowed Origins**: Set `CORS_ALLOWED_ORIGINS` to your production frontend domain (e.g., `https://app.yourdomain.com`). Never leave it as `*`.
- [ ] **Switch Mailer Driver**: Set `MAILER_DRIVER=resend` and configure a verified sender domain.
- [ ] **Switch Storage Driver**: Set `STORAGE_DRIVER=r2` or `s3` with cloud object storage credentials.
- [ ] **Disable `-fresh` Flag**: Never pass `-fresh` in production commands as it will drop all existing tables.
- [ ] **Configure Reverse Proxy**: Route requests through Nginx, Caddy, or Cloudflare with TLS termination and rate limit headers.

---

## 2. Environment Variables Reference

| Variable                 | Description                               | Development Default              | Production Recommended                          |
| :----------------------- | :---------------------------------------- | :------------------------------- | :---------------------------------------------- |
| `PORT`                   | HTTP port the server listens on           | `8080`                           | `8080` (behind reverse proxy)                   |
| `APP_ENV`                | Application runtime environment           | `development`                    | `production`                                    |
| `FRONTEND_URL`           | Base URL used for email action links      | `http://localhost:3000`          | `https://app.yourdomain.com`                    |
| `DB_HOST`                | MySQL hostname or IP address              | `127.0.0.1`                      | Cloud MySQL host / Docker network               |
| `DB_PORT`                | MySQL connection port                     | `3306`                           | `3306`                                          |
| `DB_USER`                | MySQL database user                       | `root`                           | Dedicated application user                      |
| `DB_PASSWORD`            | MySQL user password                       | `""`                             | Strong entropy password                         |
| `DB_NAME`                | MySQL database schema name                | `go_crud_api`                    | `production_db_name`                            |
| `ADMIN_NAME`             | Initial administrator display name        | `Administrator`                  | `Your Admin Name`                               |
| `ADMIN_EMAIL`            | Initial administrator login email         | `admin@mmdbconnect.com`          | `admin@yourdomain.com`                          |
| `ADMIN_PASSWORD`         | Initial administrator password            | `adminpassword123`               | Strong 16+ char password                        |
| `JWT_SECRET`             | HMAC SHA-256 signing key                  | `super-secret-key...`            | `openssl rand -hex 64`                          |
| `JWT_ACCESS_TTL_MINUTES` | Access token lifespan                     | `15`                             | `15`                                            |
| `JWT_REFRESH_TTL_DAYS`   | Refresh token lifespan                    | `7`                              | `7` (or `30`)                                   |
| `CORS_ALLOWED_ORIGINS`   | Comma-separated allowed origins           | `http://localhost:3000`          | `https://yourdomain.com`                        |
| `MAILER_DRIVER`          | Active email driver (`log` / `resend`)    | `log`                            | `resend`                                        |
| `RESEND_API_KEY`         | Resend REST API authentication key        | `""`                             | `re_xxxxxxxxxxxx`                               |
| `MAIL_FROM`              | RFC 5322 sender display and address       | `Go Kit <onboarding@resend.dev>` | `AppName <noreply@yourdomain.com>`              |
| `STORAGE_DRIVER`         | Active storage driver (`local`/`r2`/`s3`) | `local`                          | `r2` or `s3`                                    |
| `UPLOAD_DIR`             | Local disk directory for uploads          | `./uploads`                      | `./uploads`                                     |
| `S3_BUCKET`              | S3 / R2 bucket name                       | `""`                             | `app-production-media`                          |
| `S3_REGION`              | S3 region code (`auto` for R2)            | `auto`                           | `auto` (or AWS region e.g. `us-east-1`)         |
| `S3_ENDPOINT`            | Custom endpoint URL for Cloudflare R2     | `""`                             | `https://<account_id>.r2.cloudflarestorage.com` |
| `S3_ACCESS_KEY_ID`       | Cloudflare R2 / AWS Access Key ID         | `""`                             | `your_access_key_id`                            |
| `S3_SECRET_ACCESS_KEY`   | Cloudflare R2 / AWS Secret Access Key     | `""`                             | `your_secret_access_key`                        |
| `S3_PUBLIC_URL`          | Public CDN base URL for uploaded files    | `""`                             | `https://media.yourdomain.com`                  |

---

## 3. Transactional Email Setup (Resend)

The application includes an enterprise transactional email engine powered by **Resend**:

1. **Sign Up**: Register at [resend.com](https://resend.com).
2. **API Key Generation**:
   - Navigate to **API Keys** in the Resend dashboard.
   - Click **Create API Key**, name it `production-backend`, choose `Full access`, and copy the token starting with `re_`.
3. **Domain Verification**:
   - Under **Domains**, add your apex domain (e.g., `yourdomain.com`).
   - Add the required DNS records (DKIM, SPF, and MX) into your DNS provider (Cloudflare, Route53, Namecheap).
   - Wait for verification status to turn green.
4. **Update Production Environment**:
   ```ini
   MAILER_DRIVER=resend
   RESEND_API_KEY=re_1234567890abcdef...
   MAIL_FROM=App Name <noreply@yourdomain.com>
   FRONTEND_URL=https://app.yourdomain.com
   ```
5. Restart the server. Verification emails, password reset requests, and email change handshakes will now be delivered with responsive HTML templates.

---

## 4. Object Storage Setup (Cloudflare R2 / AWS S3)

To offload file storage to a scalable, zero-egress cloud object store:

1. **Create Bucket in Cloudflare R2**:
   - Go to Cloudflare Dashboard -> **R2 Object Storage**.
   - Click **Create bucket** and specify a name (e.g., `app-media-prod`).
2. **Generate API Credentials**:
   - In the R2 overview page, click **Manage R2 API Tokens** -> **Create API Token**.
   - Permissions: Select `Object Read & Write`.
   - Specify your bucket and click **Create API Token**.
   - Securely record: `Access Key ID`, `Secret Access Key`, and the `Endpoint URL`.
3. **Configure Public Access**:
   - In the bucket settings, open **Public Access**.
   - Connect a custom domain (e.g., `https://media.yourdomain.com`) or enable the managed `pub-<id>.r2.dev` subdomain.
4. **Update Production Environment**:
   ```ini
   STORAGE_DRIVER=r2
   S3_BUCKET=app-media-prod
   S3_REGION=auto
   S3_ENDPOINT=https://<account_id>.r2.cloudflarestorage.com
   S3_ACCESS_KEY_ID=your_r2_access_key_id
   S3_SECRET_ACCESS_KEY=your_r2_secret_access_key
   S3_PUBLIC_URL=https://media.yourdomain.com
   ```
5. When users upload avatars or content covers, files are automatically renamed with pure ULIDs (`avatar_<ULID>.<ext>`) and stored on Cloudflare R2. Responses return the public CDN URL.

---

## 5. Docker & Containerized Deployment

The repository includes a production-ready, multi-stage `Dockerfile` and `compose.yaml`:

### Multi-Stage Build Highlights:

- **Build Stage**: Uses `golang:1.25-alpine` to compile a statically linked binary (`CGO_ENABLED=0`).
- **Runtime Stage**: Uses minimal `alpine:3.20` with `ca-certificates` and `tzdata`, running as an unprivileged non-root user.

### Deploying with Docker Compose:

```bash
# 1. Prepare production environment file
cp .env.docker.example .env.docker
# Edit .env.docker with production passwords and keys

# 2. Start containers in detached mode
docker compose up -d --build

# 3. View live logs
docker compose logs -f api

# 4. Check container health status
docker compose ps
```

---

## 6. Reverse Proxy Configuration (Nginx & Caddy)

### Nginx Configuration

```nginx
server {
    server_name api.yourdomain.com;

    client_max_body_size 10M;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    listen 443 ssl;
    # SSL certificate directives...
}
```

### Caddy Configuration

```caddyfile
api.yourdomain.com {
    reverse_proxy 127.0.0.1:8080
}
```

---

## 7. Health Checks & Observability

- **Liveness Probe**: `GET /health` returns HTTP 200 `OK`.
- **Structured Logs**: The application emits structured JSON logs using Go's standard `log/slog` directly to `stdout`. Logs can be captured and indexed by vector, Fluentbit, Datadog, or Grafana Loki.
