<!-- ========================================================================= -->
<!-- ENTERPRISE HOTFIX PULL REQUEST TEMPLATE (Emergency Production Fix)       -->
<!-- Title Convention: hotfix(<scope>): <urgent issue description>             -->
<!-- Example: hotfix(auth): fix token reuse false-positive during parallel refresh -->
<!-- ========================================================================= -->

# 🚨 PRODUCTION HOTFIX PULL REQUEST

> [!CAUTION]
> **CRITICAL PRODUCTION EMERGENCY:**  
> This template is reserved strictly for urgent production bug fixes, security patches, or outage remediations.  
> **Rule of Minimal Scope:** Do NOT include feature additions, refactoring, or unrelated dependency updates.

---

## 1. Incident Overview & Severity

| Attribute                     | Details                                                                                                                                                                                                                      |
| :---------------------------- | :--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Severity Level**            | [ ] **P0 - Blocker** (System outage / critical service down)<br>[ ] **P1 - Critical** (Data corruption / security vulnerability / core user flow blocked)<br>[ ] **P2 - Major** (Severe degradation with limited workaround) |
| **Incident Ticket / Alert**   | <!-- PagerDuty / Sentry / Datadog / Jira Incident URL -->                                                                                                                                                                    |
| **Target Branch**             | `main` / `production`                                                                                                                                                                                                        |
| **Impacted Services / Users** | <!-- e.g., All users attempting login via mobile, article publishing worker -->                                                                                                                                              |

---

## 2. Root Cause Analysis (RCA)

### Problem Description

<!-- What exact failure occurred in production? Include relevant error logs or error codes (e.g. 500 INTERNAL_SERVER_ERROR). -->

### Root Cause

<!-- Why did this happen? What edge case, race condition, or configuration flaw triggered the issue? -->

### Why This Fix Resolves It

<!-- Technical explanation of how the patch fixes the root cause safely. -->

---

## 3. Minimal Scope Verification

- [ ] **Minimal Diff Guarantee**: This PR contains ONLY the code strictly required to fix the issue.
- [ ] **Zero Refactoring**: No cosmetic refactoring, formatting changes, or unrelated code modifications are included.
- [ ] **Backward Compatibility**: Existing database schemas, API response envelopes, and client contracts remain intact.

---

## 4. Emergency Verification Checklist

- [ ] **Reproduction Test**: A failing unit or integration test was written reproducing the incident before applying the fix.
- [ ] **Regression Test Included**: The new test is committed to permanently prevent regression.
- [ ] **Local Test Suite Passed**: `go test -v -count=1 ./...`
- [ ] **Race Detector Passed**: `go test -race ./...`
- [ ] **Staging / Sandbox Verified**: Patch was deployed and validated against a staging or sandbox environment.

---

## 5. Deployment & Rollback Strategy

### Deployment Steps:

1. Deploy container / binary to production.
2. Monitor application health probes: `GET /health` and `GET /health/ready`.
3. Inspect structured logs for error rate drop: `slog.Error` monitoring.

### Immediate Rollback Procedure:

<!-- If this hotfix causes unexpected side effects, what exact steps restore the previous state? -->

1. Immediate container rollback command: `docker compose up -d --no-deps api` (reverting to previous stable image tag).
2. Database rollback steps (if any DDL was touched):

---

## 6. Approvals & Sign-Offs

- [ ] **On-Call / Incident Commander Approval**: `@lead-engineer`
- [ ] **Backend Lead / Architect Sign-Off**: `@architect`
- [ ] **Security Lead Sign-Off** _(required for CVE / auth issues)_: `@security`
