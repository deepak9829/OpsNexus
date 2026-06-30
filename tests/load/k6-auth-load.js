/**
 * k6-auth-load.js — Load test for the OpsNexus auth service.
 *
 * Tests:
 *   - POST /api/v1/auth/login
 *   - POST /api/v1/auth/refresh  (uses token returned from login)
 *   - POST /api/v1/auth/logout
 *
 * Run:
 *   k6 run tests/load/k6-auth-load.js
 *
 * With overrides:
 *   BASE_URL=http://localhost:8081 \
 *   TEST_EMAIL=admin@opsnexus.com \
 *   TEST_PASSWORD=Admin123! \
 *     k6 run tests/load/k6-auth-load.js
 */

import http from 'k6/http'
import { check, sleep } from 'k6'
import { Rate, Trend, Counter } from 'k6/metrics'

// ---------------------------------------------------------------------------
// Custom metrics
// ---------------------------------------------------------------------------

export const errorRate        = new Rate('errors')
export const loginDuration    = new Trend('login_duration', true)
export const refreshDuration  = new Trend('refresh_duration', true)
export const logoutDuration   = new Trend('logout_duration', true)
export const totalLogins      = new Counter('logins_total')
export const totalRefreshes   = new Counter('token_refreshes_total')

// ---------------------------------------------------------------------------
// Test options
// ---------------------------------------------------------------------------

export const options = {
  stages: [
    { duration: '30s', target: 20  },  // ramp up to 20 virtual users
    { duration: '1m',  target: 100 },  // ramp up to 100 virtual users
    { duration: '2m',  target: 100 },  // steady state
    { duration: '30s', target: 0   },  // ramp down
  ],
  thresholds: {
    // 95th-percentile latency under 1 s for any request
    http_req_duration: ['p(95)<1000'],
    // Overall error rate under 5%
    errors: ['rate<0.05'],
    // Login-specific p95 under 800 ms
    login_duration: ['p(95)<800'],
    // Token refresh p95 under 400 ms
    refresh_duration: ['p(95)<400'],
  },
}

// ---------------------------------------------------------------------------
// Environment / configuration
// ---------------------------------------------------------------------------

const BASE_URL      = __ENV.BASE_URL       || 'http://localhost:8081'
const TEST_EMAIL    = __ENV.TEST_EMAIL     || 'admin@opsnexus.com'
const TEST_PASSWORD = __ENV.TEST_PASSWORD  || 'Admin123!'

const JSON_HEADERS = { 'Content-Type': 'application/json' }

// ---------------------------------------------------------------------------
// Setup — smoke test to verify the service is reachable
// ---------------------------------------------------------------------------

export function setup() {
  const healthRes = http.get(`${BASE_URL}/health`)
  if (healthRes.status !== 200) {
    console.warn(`WARNING: Auth service health check returned ${healthRes.status}`)
  }
  return {}
}

// ---------------------------------------------------------------------------
// Default function — executed by each VU on every iteration
// ---------------------------------------------------------------------------

export default function () {
  // ------------------------------------------------------------------
  // Step 1: Login
  // ------------------------------------------------------------------
  const loginPayload = JSON.stringify({
    email:    TEST_EMAIL,
    password: TEST_PASSWORD,
  })

  const loginRes = http.post(`${BASE_URL}/api/v1/auth/login`, loginPayload, {
    headers: JSON_HEADERS,
    tags:    { name: 'Login' },
  })

  loginDuration.add(loginRes.timings.duration)

  let accessToken    = null
  let refreshToken   = null

  const loginOk = check(loginRes, {
    'login: status 200': (r) => r.status === 200,
    'login: returns accessToken': (r) => {
      try {
        const body = JSON.parse(r.body)
        return body.data && typeof body.data.accessToken === 'string'
      } catch (_) {
        return false
      }
    },
    'login: returns refreshToken': (r) => {
      try {
        const body = JSON.parse(r.body)
        return body.data && typeof body.data.refreshToken === 'string'
      } catch (_) {
        return false
      }
    },
    'login: returns expiresIn': (r) => {
      try {
        const body = JSON.parse(r.body)
        return body.data && typeof body.data.expiresIn === 'number'
      } catch (_) {
        return false
      }
    },
  })

  if (!loginOk) {
    errorRate.add(1)
    sleep(1)
    return  // abort this iteration if login failed
  }

  totalLogins.add(1)

  try {
    const body  = JSON.parse(loginRes.body)
    accessToken  = body.data.accessToken
    refreshToken = body.data.refreshToken
  } catch (_) {
    errorRate.add(1)
    sleep(1)
    return
  }

  sleep(0.5)

  // ------------------------------------------------------------------
  // Step 2: Refresh the access token
  // ------------------------------------------------------------------
  const refreshPayload = JSON.stringify({ refreshToken })

  const refreshRes = http.post(`${BASE_URL}/api/v1/auth/refresh`, refreshPayload, {
    headers: JSON_HEADERS,
    tags:    { name: 'Refresh' },
  })

  refreshDuration.add(refreshRes.timings.duration)

  const refreshOk = check(refreshRes, {
    'refresh: status 200': (r) => r.status === 200,
    'refresh: returns new accessToken': (r) => {
      try {
        const body = JSON.parse(r.body)
        return body.data && typeof body.data.accessToken === 'string'
      } catch (_) {
        return false
      }
    },
  })

  if (!refreshOk) {
    errorRate.add(1)
  } else {
    totalRefreshes.add(1)
    // Use the new access token going forward
    try {
      accessToken = JSON.parse(refreshRes.body).data.accessToken
    } catch (_) { /* keep original */ }
  }

  sleep(0.5)

  // ------------------------------------------------------------------
  // Step 3: Logout (invalidate refresh token)
  // ------------------------------------------------------------------
  const logoutPayload = JSON.stringify({ refreshToken })

  const logoutRes = http.post(`${BASE_URL}/api/v1/auth/logout`, logoutPayload, {
    headers: {
      ...JSON_HEADERS,
      Authorization: `Bearer ${accessToken}`,
    },
    tags: { name: 'Logout' },
  })

  logoutDuration.add(logoutRes.timings.duration)

  const logoutOk = check(logoutRes, {
    'logout: status 200 or 204': (r) => r.status === 200 || r.status === 204,
  })

  if (!logoutOk) errorRate.add(1)

  sleep(1)
}

// ---------------------------------------------------------------------------
// Teardown
// ---------------------------------------------------------------------------

export function teardown(_data) {
  console.log('Auth load test complete.')
}
