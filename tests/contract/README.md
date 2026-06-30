# Contract Tests

Contract tests verify that each running service's HTTP responses conform to the OpenAPI 3.x specification declared in `contracts/`. They catch schema drift (a service returning fields the contract doesn't declare, or omitting required fields) before it reaches production.

## Tool: schemathesis

[schemathesis](https://schemathesis.readthedocs.io/) generates test cases from OpenAPI specs and fires them at a live service, checking that every response matches the declared schema.

### Install

```bash
# Via pip (recommended inside a venv)
python3 -m venv .venv
source .venv/bin/activate
pip install schemathesis

# Or via pipx
pipx install schemathesis
```

### Run against a single service

```bash
# Auth service (must be running on localhost:8081)
schemathesis run contracts/auth.yaml \
  --base-url http://localhost:8081 \
  --validate-schema=true \
  --checks all

# Workflow service
schemathesis run contracts/workflow.yaml \
  --base-url http://localhost:8083 \
  --validate-schema=true \
  --checks all
```

### Run all services at once

Use the bundled shell script:

```bash
# From the repo root (requires services to be running)
bash tests/contract/contract_test.sh
```

The script auto-discovers `contracts/*.yaml`, maps each to its service port,
health-checks the service, and runs schemathesis. It exits non-zero if any
service fails.

### CI integration

In CI, the contract job runs after `docker compose up` and before the
integration test job:

```yaml
- name: Run contract tests
  run: bash tests/contract/contract_test.sh
```

## Service → Port mapping

| Service      | Port  | Contract file           |
|--------------|-------|-------------------------|
| auth         | 8081  | `contracts/auth.yaml`   |
| tenant       | 8082  | `contracts/tenant.yaml` |
| workflow     | 8083  | `contracts/workflow.yaml` |
| document     | 8084  | `contracts/document.yaml` |
| notification | 8085  | `contracts/notification.yaml` |

## Useful schemathesis flags

| Flag                        | Purpose                                                      |
|-----------------------------|--------------------------------------------------------------|
| `--validate-schema=true`    | Fail if the OpenAPI spec itself is invalid                   |
| `--checks all`              | Run all built-in checks (status codes, schema, headers, …)  |
| `--hypothesis-max-examples` | Number of generated test cases per operation (default: 100) |
| `--request-timeout`         | Seconds before a request is considered failed               |
| `--auth`                    | Bearer token for authenticated endpoints                     |
| `--output-truncation-size`  | Limit truncation of long response bodies in output          |
| `--junit-xml report.xml`    | Emit JUnit XML for CI test result collection                |

Example with auth header and JUnit output:

```bash
schemathesis run contracts/auth.yaml \
  --base-url http://localhost:8081 \
  --auth "Bearer $ACCESS_TOKEN" \
  --validate-schema=true \
  --checks all \
  --junit-xml tests/contract/results/auth-report.xml
```

## Adding a new service

1. Add the OpenAPI spec as `contracts/<service>.yaml`.
2. Add the port mapping to the `SERVICE_PORTS` map in `tests/contract/contract_test.sh`.
3. Update the table above in this README.
