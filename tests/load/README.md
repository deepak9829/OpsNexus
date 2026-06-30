# Load Tests

Load tests are written with [k6](https://k6.io/) and live in this directory. They measure throughput, latency, and error rates for the OpsNexus backend services under simulated traffic.

## Install k6

```bash
# macOS
brew install k6

# Docker (no local install needed)
docker pull grafana/k6

# Linux (Debian/Ubuntu)
sudo gpg -k
sudo gpg --no-default-keyring \
  --keyring /usr/share/keyrings/k6-archive-keyring.gpg \
  --keyserver hkp://keyserver.ubuntu.com:80 \
  --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" \
  | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update && sudo apt-get install k6
```

## Running Load Tests

### Quick run (local defaults)

```bash
# Cases API load test
k6 run tests/load/k6-cases-load.js

# Auth service load test
k6 run tests/load/k6-auth-load.js
```

### With Docker

```bash
docker run --rm -i \
  -e BASE_URL=http://host.docker.internal:8083 \
  grafana/k6 run - < tests/load/k6-cases-load.js
```

### With environment variable overrides

```bash
BASE_URL=http://10.0.1.50:8083 \
TENANT_ID=550e8400-e29b-41d4-a716-446655440000 \
ACCESS_TOKEN=$(cat .local/token) \
  k6 run tests/load/k6-cases-load.js
```

### Save output to a JSON file

```bash
k6 run --out json=results/cases-load-$(date +%Y%m%d-%H%M%S).json \
  tests/load/k6-cases-load.js
```

## Environment Variables

| Variable       | Default                   | Description                                       |
|----------------|---------------------------|---------------------------------------------------|
| `BASE_URL`     | `http://localhost:8083`   | Base URL of the target service                    |
| `TENANT_ID`    | `test-tenant-id`          | Tenant UUID to include in `X-Tenant-ID` header    |
| `ACCESS_TOKEN` | `test-token`              | JWT bearer token for authenticated endpoints      |
| `TEST_EMAIL`   | `admin@opsnexus.com`      | Email for auth load test login                    |
| `TEST_PASSWORD`| `Admin123!`               | Password for auth load test login                 |

## Interpreting Results

k6 prints a summary table at the end of each run. Key fields to watch:

### `http_req_duration`
Response time distribution across all requests.

```
http_req_duration ................: avg=123ms  min=45ms  med=110ms  max=950ms  p(90)=210ms  p(95)=280ms
```

- **p(95)** is the primary SLO gate: 95% of requests must complete within the threshold.
- If p(95) approaches the threshold during ramp-up, the service is likely saturating.

### `errors` (custom metric)
Rate of requests where the response did not match the expected status or schema.

```
errors .........................: 0.12%  ✓ 0  ✗ 3
```

- Must stay below the defined `rate<X` threshold.
- An error rate spike that starts after a ramp-up plateau usually indicates DB connection pool exhaustion or GC pressure.

### `http_req_failed`
k6's built-in failed-request rate (non-2xx or network error).

## Thresholds and Baselines

Thresholds are enforced by k6: a run exits non-zero if any threshold is breached.

| Test file             | Metric                  | Threshold      | Meaning                             |
|-----------------------|-------------------------|----------------|-------------------------------------|
| `k6-cases-load.js`    | `http_req_duration`     | `p(95)<500ms`  | 95% of all requests under 500 ms    |
| `k6-cases-load.js`    | `errors`                | `rate<0.01`    | Error rate under 1%                 |
| `k6-cases-load.js`    | `cases_list_duration`   | `p(95)<400ms`  | List endpoint p95 under 400 ms      |
| `k6-cases-load.js`    | `cases_create_duration` | `p(95)<600ms`  | Create endpoint p95 under 600 ms    |
| `k6-auth-load.js`     | `http_req_duration`     | `p(95)<1000ms` | Auth requests (including login) under 1 s |
| `k6-auth-load.js`     | `errors`                | `rate<0.05`    | Error rate under 5% (login is expensive) |
| `k6-auth-load.js`     | `login_duration`        | `p(95)<800ms`  | Login endpoint p95 under 800 ms     |
| `k6-auth-load.js`     | `refresh_duration`      | `p(95)<400ms`  | Token refresh p95 under 400 ms      |

## Expected Baselines (single-node, local Docker Compose)

These are reference points for a developer laptop. Production targets will differ.

| Service    | Endpoint             | VUs | Expected p95 |
|------------|----------------------|-----|--------------|
| workflow   | GET /cases           | 50  | < 250 ms     |
| workflow   | POST /cases          | 50  | < 400 ms     |
| auth       | POST /auth/login     | 100 | < 600 ms     |
| auth       | POST /auth/refresh   | 100 | < 200 ms     |

## CI Integration

Load tests run as a separate GitHub Actions job on a schedule (nightly) and on
releases. They are NOT run on every pull request due to infrastructure cost.

```yaml
# .github/workflows/load.yml (excerpt)
- name: Run cases load test
  run: k6 run --exit-on-running tests/load/k6-cases-load.js
  env:
    BASE_URL: ${{ secrets.STAGING_WORKFLOW_URL }}
    ACCESS_TOKEN: ${{ secrets.STAGING_TOKEN }}
    TENANT_ID: ${{ secrets.STAGING_TENANT_ID }}
```
