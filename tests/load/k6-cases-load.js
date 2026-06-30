/**
 * k6-cases-load.js — Load test for the OpsNexus workflow service Cases API.
 *
 * Run:
 *   k6 run tests/load/k6-cases-load.js
 *
 * With overrides:
 *   BASE_URL=http://localhost:8083 TENANT_ID=<uuid> ACCESS_TOKEN=<jwt> \
 *     k6 run tests/load/k6-cases-load.js
 */

import http from 'k6/http'
import { check, sleep } from 'k6'
import { Rate, Trend, Counter } from 'k6/metrics'

// ---------------------------------------------------------------------------
// Custom metrics
// ---------------------------------------------------------------------------

export const errorRate = new Rate('errors')
export const casesListDuration = new Trend('cases_list_duration', true)
export const casesCreateDuration = new Trend('cases_create_duration', true)
export const casesGetDuration = new Trend('cases_get_duration', true)
export const totalCasesCreated = new Counter('cases_created_total')

// ---------------------------------------------------------------------------
// Test options
// ---------------------------------------------------------------------------

export const options = {
  stages: [
    { duration: '30s', target: 10 },  // ramp up to 10 virtual users
    { duration: '1m',  target: 50 },  // ramp up to 50 virtual users
    { duration: '2m',  target: 50 },  // steady state at 50 virtual users
    { duration: '30s', target: 0 },   // ramp down
  ],
  thresholds: {
    // 95th-percentile response time must stay under 500 ms
    http_req_duration: ['p(95)<500'],
    // Overall error rate must stay under 1%
    errors: ['rate<0.01'],
    // Cases list p95 under 400 ms
    cases_list_duration: ['p(95)<400'],
    // Cases create p95 under 600 ms
    cases_create_duration: ['p(95)<600'],
  },
}

// ---------------------------------------------------------------------------
// Environment / configuration
// ---------------------------------------------------------------------------

const BASE_URL  = __ENV.BASE_URL   || 'http://localhost:8083'
const TENANT_ID = __ENV.TENANT_ID  || 'test-tenant-id'
const TOKEN     = __ENV.ACCESS_TOKEN || 'test-token'

// Priority options for random case creation
const PRIORITIES = ['low', 'medium', 'high', 'critical']

function randomPriority() {
  return PRIORITIES[Math.floor(Math.random() * PRIORITIES.length)]
}

// ---------------------------------------------------------------------------
// Setup — runs once before the load test; returns shared data for VUs
// ---------------------------------------------------------------------------

export function setup() {
  // In a real environment, perform a login here and return the token.
  // For now we propagate the token from the environment.
  const data = { token: TOKEN, tenantId: TENANT_ID }

  // Smoke-check: verify the service is up before ramping load
  const healthRes = http.get(`${BASE_URL}/health`)
  if (healthRes.status !== 200) {
    console.warn(`WARNING: Health check returned ${healthRes.status} — service may not be ready`)
  }

  return data
}

// ---------------------------------------------------------------------------
// Default function — executed by each virtual user on each iteration
// ---------------------------------------------------------------------------

export default function (data) {
  const headers = {
    Authorization: `Bearer ${data.token}`,
    'X-Tenant-ID': data.tenantId,
    'Content-Type': 'application/json',
  }

  // ------------------------------------------------------------------
  // Scenario 1: List cases (page 1, 20 per page)
  // ------------------------------------------------------------------
  const listRes = http.get(`${BASE_URL}/api/v1/cases?page=1&limit=20`, { headers, tags: { name: 'ListCases' } })

  casesListDuration.add(listRes.timings.duration)

  const listOk = check(listRes, {
    'list cases: status 200': (r) => r.status === 200,
    'list cases: has data field': (r) => {
      try {
        const body = JSON.parse(r.body)
        return body.data !== undefined
      } catch (_) {
        return false
      }
    },
    'list cases: has pagination meta': (r) => {
      try {
        const body = JSON.parse(r.body)
        return body.meta !== undefined && body.meta.total !== undefined
      } catch (_) {
        return false
      }
    },
  })
  if (!listOk) errorRate.add(1)

  sleep(1)

  // ------------------------------------------------------------------
  // Scenario 2: Create a new case
  // ------------------------------------------------------------------
  const createPayload = JSON.stringify({
    title:       `Load Test Case ${Date.now()}`,
    description: 'Created by k6 load test — safe to delete',
    priority:    randomPriority(),
  })

  const createRes = http.post(`${BASE_URL}/api/v1/cases`, createPayload, {
    headers,
    tags: { name: 'CreateCase' },
  })

  casesCreateDuration.add(createRes.timings.duration)

  const createOk = check(createRes, {
    'create case: status 201': (r) => r.status === 201,
    'create case: returns case id': (r) => {
      try {
        const body = JSON.parse(r.body)
        return body.data && body.data.id !== undefined
      } catch (_) {
        return false
      }
    },
    'create case: has case number': (r) => {
      try {
        const body = JSON.parse(r.body)
        return body.data && /^CASE-\d+$/.test(body.data.caseNumber)
      } catch (_) {
        return false
      }
    },
  })
  if (!createOk) {
    errorRate.add(1)
  } else {
    totalCasesCreated.add(1)

    // ------------------------------------------------------------------
    // Scenario 3: Fetch the newly created case by ID
    // ------------------------------------------------------------------
    let caseId
    try {
      caseId = JSON.parse(createRes.body).data.id
    } catch (_) {
      caseId = null
    }

    if (caseId) {
      const getRes = http.get(`${BASE_URL}/api/v1/cases/${caseId}`, {
        headers,
        tags: { name: 'GetCase' },
      })

      casesGetDuration.add(getRes.timings.duration)

      const getOk = check(getRes, {
        'get case: status 200': (r) => r.status === 200,
        'get case: id matches': (r) => {
          try {
            return JSON.parse(r.body).data.id === caseId
          } catch (_) {
            return false
          }
        },
      })
      if (!getOk) errorRate.add(1)
    }
  }

  sleep(0.5)
}

// ---------------------------------------------------------------------------
// Teardown — runs once after all VUs finish
// ---------------------------------------------------------------------------

export function teardown(data) {
  console.log(`Load test complete. Tenant: ${data.tenantId}`)
}
