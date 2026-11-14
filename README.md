# LoadForge

**A developer-first HTTP performance and backend response-signal testing tool.**

LoadForge combines a Go load-generation engine, a CLI, a JSON API, and a React dashboard. It helps developers measure latency under bounded load, inspect throughput and failures, and examine backend-identifying response headers where a service exposes them.

This is an MVP, not a claim that a URL reveals the true capacity of a backend. Results describe only the test run from the machine and network that executed it. Header observations are clues; they do not prove a load balancer's internal configuration.

## Features

- Bounded HTTP GET scheduling with configurable concurrency, duration, request rate, and total request cap.
- Average, minimum, maximum, p50, p90, p95, and p99 latency.
- Completed requests/second, status-code counts, HTTP errors, network errors, and sampled response bytes.
- Observations for Server, Via, X-Served-By, X-Backend, X-Backend-Server, X-Instance-Id, X-Pod-Name, X-Upstream-Addr, and X-Upstream-Host.
- CLI JSON report export, HTTP API, and React dashboard.
- Private/reserved IP blocking by default and a DNS re-check at connection time.
- Explicit authorization confirmation, no automatic retries, and no redirect following.

## Requirements

- Go 1.23+
- Node.js 20+ and npm for the dashboard

## CLI

    go run ./cmd/loadforge run --url https://your-service.example/health --authorization-confirmed --concurrency 5 --duration 10s --rps 10 --max-requests 1000 --json report.json

You must confirm that you own the target or have explicit permission to test it.

For local development only, private targets can be enabled explicitly:

    go run ./cmd/loadforge run --url http://127.0.0.1:3000/health --authorization-confirmed --allow-private-targets --concurrency 2 --duration 3s --rps 2 --max-requests 20

Never use the private-target switch on a shared or internet-facing instance.

## API and dashboard

Start the Go service:

    go run ./cmd/loadforge serve --addr 127.0.0.1:8080

Start the dashboard in a second terminal:

    cd web
    npm install
    npm run dev

Open http://127.0.0.1:5173. Vite proxies /api and /healthz to the local Go API.

For a production-style local build, run npm install and npm run build in web, then start the Go service. The built dashboard is served from web/dist.

## API

POST /api/v1/runs accepts one JSON object:

    {
      "url": "https://your-service.example/health",
      "concurrency": 5,
      "duration_seconds": 10,
      "requests_per_second": 10,
      "max_requests": 1000,
      "authorization_confirmed": true
    }

GET /healthz returns the service health.

The API blocks private targets by default. For a trusted local-only server, set LOADFORGE_ALLOW_PRIVATE_TARGETS=true. This is a process-wide switch, not a per-request option. The API has no built-in authentication, tenant isolation, or user quotas in this MVP. Keep it bound to localhost or behind a trusted authenticated network boundary; do not expose it publicly as-is.

## Safety ceilings

| Control | Default | Hard limit |
|---|---:|---:|
| Concurrency | 5 | 25 |
| Duration | 10 seconds | 30 seconds |
| Scheduled rate | 10 / second | 50 / second |
| Total requests | 1,000 | 5,000 |
| Per-request timeout | 5 seconds | 10 seconds |
| Response bytes consumed | — | 64 KiB / response |

Only GET is supported. Redirects are not followed. Failed requests are not retried. When the bounded work queue is full, a scheduled tick is skipped instead of being added to an unbounded backlog.

## Interpreting results

- Requests/second is completed requests divided by measured elapsed time; it is not a guaranteed target rate.
- Responses below 400 are counted as successful. A 3xx response is counted because redirects are intentionally not followed.
- Failed includes transport errors and HTTP status codes of 400 or higher.
- Tail percentiles become unstable when the sample size is small.
- Backend signals count selected response headers. A CDN, reverse proxy, web server, or application middleware may set them. A missing header does not prove that no load balancer exists.

Test from a machine and network location representative of your users. CPU, network conditions, DNS, TLS, caching, background traffic, and test-machine capacity influence measurements.

## Development

    gofmt -w ./cmd ./internal
    go test ./...
    go vet ./...

The tests use local httptest servers and explicitly allow private addresses only inside test configurations.

## Roadmap

1. Ramp-up, spike, and endurance profiles within the same safety ceilings.
2. Live progress and interval metrics during a run.
3. Local run history and comparable side-by-side reports.
4. Performance regression thresholds for CI.
5. CSV export and repeatable test profiles.
6. Additional HTTP methods and request bodies with explicit safe configuration.

## License

MIT. See LICENSE.