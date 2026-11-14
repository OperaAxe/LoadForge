# Security policy

Only run LoadForge against systems you own or have explicit permission to test.

## Built-in controls

- CLI requires --authorization-confirmed; API requires authorization_confirmed: true.
- Concurrency, duration, request rate, request count, timeout, and response consumption are bounded.
- GET only; redirects are not followed; failed requests are not retried.
- Private, loopback, link-local, metadata, and reserved IP ranges are blocked by default.
- DNS is checked again when opening a connection to reduce DNS-rebinding risk.
- The API has no built-in authentication. Keep it on localhost or behind a trusted authenticated network boundary.

## Reporting a vulnerability

Please avoid public disclosure before maintainers have had a reasonable opportunity to investigate. Use GitHub Security Advisories when available, or contact the repository owner via the GitHub profile.