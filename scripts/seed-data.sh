#!/usr/bin/env bash
# seed-data.sh — populate OpsNexus with realistic demo data
set -uo pipefail

BASE_AUTH="${BASE_AUTH:-http://localhost:8081/api/v1}"
BASE_WF="${BASE_WF:-http://localhost:8083/api/v1}"
BASE_NOTIF="${BASE_NOTIF:-http://localhost:8085/api/v1}"
TENANT="00000000-0000-0000-0000-000000000001"

log() { echo "  ✓ $1" >&2; }
err() { echo "  ✗ $1" >&2; }

# ── 1. Auth ───────────────────────────────────────────────────────────────────
echo ""
echo "━━ [1/5] Authenticating ━━"
LOGIN=$(curl -s -X POST "$BASE_AUTH/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@opsnexus.com","password":"Admin123!"}')
TOKEN=$(echo "$LOGIN" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])")
ME=$(curl -s "$BASE_AUTH/auth/me" -H "Authorization: Bearer $TOKEN")
ADMIN_ID=$(echo "$ME" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['id'])")
log "Logged in as admin (id: $ADMIN_ID)"

# ── 2. Users ──────────────────────────────────────────────────────────────────
echo ""
echo "━━ [2/5] Creating users ━━"

# Register a user; if already exists try to login and return id
register_user() {
  local email="$1" first="$2" last="$3" pass="${4:-Pass1234!}"
  # Try register (snake_case fields)
  local res
  res=$(curl -s -X POST "$BASE_AUTH/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$email\",\"password\":\"$pass\",\"first_name\":\"$first\",\"last_name\":\"$last\",\"tenant_id\":\"$TENANT\"}")
  local uid
  uid=$(echo "$res" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null) || uid=""
  if [ -n "$uid" ]; then
    log "Created $first $last ($email)"
    echo "$uid"
    return
  fi
  # Already exists — login to get id
  local tok
  tok=$(curl -s -X POST "$BASE_AUTH/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$email\",\"password\":\"$pass\"}" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])") || tok=""
  if [ -z "$tok" ]; then
    err "Could not create or login $email"; echo "$ADMIN_ID"; return
  fi
  uid=$(curl -s "$BASE_AUTH/auth/me" -H "Authorization: Bearer $tok" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['id'])") || uid="$ADMIN_ID"
  log "$first $last already exists (id: $uid)"
  echo "$uid"
}

U1=$(register_user "sarah.chen@opsnexus.com"    "Sarah"  "Chen")
U2=$(register_user "james.miller@opsnexus.com"  "James"  "Miller")
U3=$(register_user "priya.patel@opsnexus.com"   "Priya"  "Patel")
U4=$(register_user "lucas.kim@opsnexus.com"     "Lucas"  "Kim")
U5=$(register_user "anna.novak@opsnexus.com"    "Anna"   "Novak")

# ── 3. Cases ──────────────────────────────────────────────────────────────────
echo ""
echo "━━ [3/5] Creating cases ━━"

# Helpers — all use TOKEN + TENANT; reporter passed as X-User-ID
wf_post() {
  local path="$1" uid="$2" body="$3"
  curl -s -X POST "$BASE_WF$path" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -H "X-Tenant-ID: $TENANT" \
    -H "X-User-ID: $uid" \
    -d "$body"
}

case_id() {
  echo "$1" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('ID') or d.get('id',''))" 2>/dev/null || echo ""
}

create_case() {
  local title="$1" desc="$2" priority="$3" tags="$4" uid="${5:-$ADMIN_ID}"
  local res
  res=$(wf_post "/cases" "$uid" "{\"title\":\"$title\",\"description\":\"$desc\",\"priority\":\"$priority\",\"tags\":$tags}")
  local cid; cid=$(case_id "$res")
  if [ -n "$cid" ]; then log "Case [$priority]: $title"; else err "Failed to create: $title"; fi
  echo "$cid"
}

transition() {
  local cid="$1" status="$2" reason="$3"
  [ -z "$cid" ] && return
  wf_post "/cases/$cid/transitions" "$ADMIN_ID" "{\"toStatus\":\"$status\",\"reason\":\"$reason\"}" > /dev/null
}

comment() {
  local cid="$1" uid="$2" body="$3"
  [ -z "$cid" ] && return
  wf_post "/cases/$cid/comments" "$uid" "{\"body\":\"$body\"}" > /dev/null
}

task() {
  local cid="$1" title="$2" desc="$3" uid="$4"
  [ -z "$cid" ] && return
  wf_post "/cases/$cid/tasks" "$ADMIN_ID" "{\"title\":\"$title\",\"description\":\"$desc\",\"assigneeId\":\"$uid\"}" > /dev/null
}

# ── critical ──
C1=$(create_case \
  "Production database connection pool exhausted" \
  "All connection slots are taken. New connections refused. Customers cannot access the platform." \
  "critical" '["database","production","outage"]' "$ADMIN_ID")

C2=$(create_case \
  "Payment gateway returning 503 errors" \
  "Payment processing is failing for all customers since 14:30 UTC. Error rate 100% on /api/payment/charge." \
  "critical" '["payments","integration","outage"]' "$U1")

# ── high ──
C3=$(create_case \
  "SSL certificate expiring in 7 days" \
  "Wildcard cert for *.opsnexus.com expires 2026-06-25. Auto-renewal failed last cycle." \
  "high" '["ssl","infrastructure","security"]' "$U2")

C4=$(create_case \
  "Memory leak in notification worker" \
  "notification-service memory climbs from 200 MB to 2 GB over 6 hours before OOM kill. Heap dump attached." \
  "high" '["memory","notification-service","performance"]' "$U3")

C5=$(create_case \
  "Bulk user import fails for CSV files larger than 5 MB" \
  "Tenant admin reports CSV import silently fails when file exceeds ~5 MB. Small files work fine." \
  "high" '["import","users","bug"]' "$U1")

C6=$(create_case \
  "Dashboard load time exceeds 8 seconds" \
  "Customer portal dashboard takes 8-12 s on first load. Profiling shows N+1 queries on the cases list." \
  "high" '["performance","dashboard","api"]' "$U2")

# ── medium ──
C7=$(create_case \
  "Email notifications not delivered to Gmail addresses" \
  "Users with Gmail report never receiving notification emails. SPF/DKIM records may be misconfigured." \
  "medium" '["email","notifications","deliverability"]' "$U4")

C8=$(create_case \
  "Workflow template editor loses unsaved changes on navigation" \
  "If a user navigates away without saving, a confirmation dialog should appear but does not." \
  "medium" '["workflow","ux","frontend"]' "$U5")

C9=$(create_case \
  "API rate limiting not enforced per tenant" \
  "Rate limits are applied globally. High-traffic tenant can starve other tenants." \
  "medium" '["api","rate-limiting","multi-tenant"]' "$U3")

C10=$(create_case \
  "Audit log timestamps inconsistent across timezones" \
  "Audit events show different timestamps depending on browser timezone. All times should be UTC." \
  "medium" '["audit","timezone","data"]' "$ADMIN_ID")

C11=$(create_case \
  "Case search does not return partial-match results" \
  "Full-text search only matches exact words. Searching for connect does not find connection." \
  "medium" '["search","cases","feature"]' "$U1")

C12=$(create_case \
  "Two-factor authentication not enforced for admin accounts" \
  "Platform policy requires 2FA for admin roles. The enforcement check is missing from login flow." \
  "medium" '["security","2fa","auth"]' "$U2")

# ── low ──
C13=$(create_case \
  "Add dark mode support to customer portal" \
  "Multiple customers requested dark mode. Should respect prefers-color-scheme media query." \
  "low" '["feature","ui","portal"]' "$U4")

C14=$(create_case \
  "Outdated dependencies in auth-service" \
  "go list -m -u shows 12 outdated modules. None are critical CVEs but should be kept current." \
  "low" '["maintenance","dependencies","auth-service"]' "$U5")

C15=$(create_case \
  "Improve error messages for failed case transitions" \
  "When a transition fails validation the API returns a generic 400. Should return the specific reason." \
  "low" '["ux","api","workflow"]' "$U3")

C16=$(create_case \
  "Add CSV export for case list" \
  "Operations team wants to export filtered case lists to CSV for weekly reporting." \
  "low" '["feature","export","reporting"]' "$ADMIN_ID")

C17=$(create_case \
  "Onboarding checklist for new tenants" \
  "New tenants have no guided setup. Add a checklist: invite users, create workflow, configure notifications." \
  "low" '["onboarding","feature","ux"]' "$U1")

# ── Transitions ───────────────────────────────────────────────────────────────
echo ""
echo "  Transitioning case statuses..."

transition "$C1" "open"        "Escalated — investigating DB connection pool"
transition "$C1" "in_progress" "DBA on call, draining idle connections and tuning PgBouncer"

transition "$C2" "open"        "Confirmed 503 from Stripe status page"
transition "$C2" "in_progress" "Failover to Braintree backup gateway initiated"
transition "$C2" "pending"     "Waiting on vendor hotfix ETA"

transition "$C3" "open"        "Certificate renewal job triggered manually"
transition "$C3" "in_progress" "New cert issued, deploying to load balancers"
transition "$C3" "resolved"    "Certificate renewed and deployed successfully"

transition "$C4" "open"        "Memory profiling in progress on staging"
transition "$C4" "in_progress" "Found SQS event listener leak in queue consumer"

transition "$C5" "open"        "Reproduced locally with a 6 MB test file"

transition "$C6" "open"        "Query profiler enabled in staging"

transition "$C7" "open"        "Checking SPF and DKIM DNS records"
transition "$C7" "resolved"    "Added missing DKIM record, delivery confirmed"

transition "$C9" "open"        "Reviewing rate-limit middleware implementation"

transition "$C12" "open"        "Security audit flagged this as P1"
transition "$C12" "in_progress" "2FA enforcement added to login middleware"
transition "$C12" "resolved"    "2FA enforcement deployed to production"

transition "$C14" "open"        "Running go get -u to check compatibility"
transition "$C14" "resolved"    "All 12 dependencies updated, tests passing"

log "Transitions done"

# ── Comments ──────────────────────────────────────────────────────────────────
echo ""
echo "  Adding comments..."

comment "$C1" "$ADMIN_ID" "Confirmed: pg_stat_activity shows 100/100 connections active. max_connections needs tuning or PgBouncer pool_size needs to be reduced."
comment "$C1" "$U3"       "PgBouncer is configured with pool_size=20 per user. With 5 app instances we hit 100 connections exactly at peak."
comment "$C1" "$ADMIN_ID" "Temporary fix: restarted 2 idle app pods to free connections. Permanent fix: switch to transaction pooling mode."

comment "$C2" "$U1"       "Stripe status page shows ongoing incident across all regions. ETA unknown."
comment "$C2" "$ADMIN_ID" "Switched to Braintree fallback. Payment success rate back to 99.8%. Monitoring closely."
comment "$C2" "$U1"       "Stripe incident resolved at 16:45 UTC. Will switch back to primary gateway after 1-hour soak test."

comment "$C4" "$U3"       "Heap snapshot at t=0h: 198 MB. At t=6h: 1.87 GB. Growth is perfectly linear — classic listener leak."
comment "$C4" "$ADMIN_ID" "Root cause found: SQS consumer registers a new message listener on every poll cycle without removing old ones."

comment "$C6" "$U2"       "EXPLAIN ANALYZE on GET /cases: each row triggers a separate query for assignee and reporter user details. Classic N+1."
comment "$C6" "$ADMIN_ID" "Fix: preload user data with a single JOIN. Estimated improvement: 3000 ms to 50 ms per page."

comment "$C8" "$U5"       "Reproduced in Chrome and Safari. React Router navigation does not block and the beforeunload dialog is suppressed."
comment "$C8" "$U2"       "Fix: use React Router v6.4 blocker API to intercept navigation when the form isDirty flag is true."

log "Comments done"

# ── Tasks ─────────────────────────────────────────────────────────────────────
echo ""
echo "  Creating tasks..."

task "$C1" "Tune PgBouncer pool_size" \
  "Set server_pool_size=10, pool_mode=transaction and monitor connection wait time" "$ADMIN_ID"
task "$C1" "Add connection pool metrics to Grafana" \
  "Export pg_stat_activity and pgbouncer stats to Prometheus scrape endpoint" "$U3"
task "$C1" "Load test with new pool config" \
  "Run k6 scenario at 2x peak traffic to verify no connection exhaustion" "$U2"

task "$C4" "Fix SQS listener registration" \
  "Move listener setup outside the poll loop, use singleton registration on service start" "$U3"
task "$C4" "Add memory usage alert" \
  "Alert when notification-service RSS exceeds 500 MB for more than 5 minutes" "$ADMIN_ID"
task "$C4" "Write regression test" \
  "Unit test verifying listener count stays constant across 1000 consecutive poll cycles" "$U5"

task "$C6" "Rewrite cases list query with JOIN" \
  "Replace lazy-loaded user lookups with a single JOIN query in CaseRepository.List" "$U2"
task "$C6" "Add Redis cache for user profiles" \
  "Cache user display data (name, email) for 5 minutes to reduce DB load" "$ADMIN_ID"

task "$C3" "Set up auto-renewal with cert-manager" \
  "Configure ACME issuer in cert-manager to auto-renew 30 days before expiry" "$U2"
task "$C3" "Add certificate expiry monitoring" \
  "Alert at 30 days, 14 days, and 7 days before cert expiry via PagerDuty" "$ADMIN_ID"

task "$C9" "Add per-tenant rate-limit middleware" \
  "Implement token bucket keyed on X-Tenant-ID with configurable limits per tenant tier" "$U3"
task "$C9" "Test rate-limit tenant isolation" \
  "Verify tenant A traffic at 10x quota does not affect tenant B response times" "$U1"

log "Tasks done"

# ── 4. Notifications ──────────────────────────────────────────────────────────
echo ""
echo "━━ [4/5] Creating notifications ━━"

notif() {
  local uid="$1" type="$2" title="$3" body="$4"
  curl -s -X POST "$BASE_NOTIF/notifications" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -H "X-Tenant-ID: $TENANT" \
    -d "{\"tenantId\":\"$TENANT\",\"userId\":\"$uid\",\"type\":\"$type\",\"title\":\"$title\",\"body\":\"$body\",\"channel\":\"in_app\"}" > /dev/null
}

notif "$ADMIN_ID" "error"   "CRITICAL: DB Connection Pool Exhausted" \
  "Production database has hit max_connections limit. Case opened — DBA on call."
notif "$ADMIN_ID" "error"   "Payment Gateway Outage Detected" \
  "Stripe returning 503 errors since 14:30 UTC. Failover to Braintree in progress."
notif "$ADMIN_ID" "warning" "SSL Certificate Expiring in 7 Days" \
  "Wildcard cert *.opsnexus.com expires 2026-06-25. Auto-renewal failed."
notif "$ADMIN_ID" "success" "SSL Certificate Renewed Successfully" \
  "Certificate deployed to all load balancers. Next expiry: 2027-06-18."
notif "$ADMIN_ID" "success" "2FA Enforcement Live in Production" \
  "All admin accounts now require two-factor authentication on every login."
notif "$ADMIN_ID" "warning" "Memory Leak in Notification Worker" \
  "notification-service consuming 1.8 GB RAM. Root cause identified — fix in review."
notif "$ADMIN_ID" "info"    "Weekly Summary: 17 Cases This Sprint" \
  "4 resolved · 3 in progress · 5 open · 5 new. Strong progress this week!"
notif "$ADMIN_ID" "info"    "Q3 Roadmap Published" \
  "Dark mode, bulk export, and tenant onboarding checklist added to the Q3 backlog."

notif "$U1" "error"   "Payment Gateway Down — You Reported This" \
  "Your case is being handled. Failover to Braintree initiated. ETA for Stripe recovery: unknown."
notif "$U1" "success" "Bulk Import Issue Escalated to Engineering" \
  "The 5 MB CSV import failure has been reproduced and assigned to the backend team."
notif "$U1" "info"    "Case Search Enhancement Added to Backlog" \
  "Partial-match search support has been queued for the next sprint planning session."

notif "$U2" "warning" "SSL Renewal Task Assigned to You" \
  "You have been assigned the cert-manager setup task. Target completion: today."
notif "$U2" "success" "Dashboard N+1 Fix Ready for Review" \
  "Query rewrite merged to main branch. Deploy scheduled for tonight at 22:00 UTC."
notif "$U2" "info"    "Code Review Requested on PR #47" \
  "Sarah Chen requested your review on the workflow editor navigation guard fix."

notif "$U3" "error"   "Memory Leak Root Cause Confirmed" \
  "SQS listener leak identified in your service. PR with fix is in review — please check."
notif "$U3" "warning" "Rate Limiting Analysis Needed" \
  "Your input is needed on the per-tenant rate limiting design before implementation starts."
notif "$U3" "success" "Audit Timezone Fix Deployed" \
  "All audit event timestamps now stored and displayed in UTC across all regions."

notif "$U4" "info"    "Dark Mode Feature Planned for Q3" \
  "Dark mode support added to the Q3 roadmap. Design mockups requested."
notif "$U4" "success" "Gmail Deliverability Issue Resolved" \
  "DKIM record added to DNS. Please confirm you are now receiving notification emails."

notif "$U5" "info"    "Workflow Editor Bug Assigned to You" \
  "Navigation guard issue confirmed in staging. PR expected this sprint."
notif "$U5" "success" "Dependency Updates Complete in auth-service" \
  "All 12 outdated Go modules updated. CI green across all test suites."

log "Notifications sent (20 total)"

# ── 5. Summary ────────────────────────────────────────────────────────────────
echo ""
echo "━━ [5/5] Done ━━"
echo ""
echo "  Users:"
echo "    admin@opsnexus.com          / Admin123!"
echo "    sarah.chen@opsnexus.com     / Pass1234!"
echo "    james.miller@opsnexus.com   / Pass1234!"
echo "    priya.patel@opsnexus.com    / Pass1234!"
echo "    lucas.kim@opsnexus.com      / Pass1234!"
echo "    anna.novak@opsnexus.com     / Pass1234!"
echo ""
echo "  Cases:         17 (2 critical · 4 high · 6 medium · 5 low)"
echo "  Transitions:   17 status changes"
echo "  Comments:      12 threaded comments"
echo "  Tasks:         12 tasks across 5 cases"
echo "  Notifications: 20 across 6 users"
echo ""
echo "  http://localhost:3000  — Customer Portal"
echo "  http://localhost:3001  — Admin Console"
