# Contributing to marz

Thanks for helping! Issues and pull requests in **English or Persian** are welcome.

The most useful contribution is **data**: a service marz does not know yet, or a verdict that is wrong
or out of date. Use the [knowledge base correction](https://github.com/mrzroot/marz/issues/new?template=kb_correction.yml)
form, or edit [`internal/kb/rules.json`](internal/kb/rules.json) directly.

## Development setup

```sh
git clone https://github.com/mrzroot/marz && cd marz
go test ./...                       # unit tests for scan, kb, probe, report, ooni, fonts and the CLI
go run ./cmd/marz testdata/demo     # scan the small demo app
sh scripts/build.sh 0.1.0           # cross-compiled release binaries + SHA256SUMS in dist/
```

marz is a single static Go binary with the knowledge base built in. Please keep it free of runtime dependencies.

## Where things live

| Path | What |
|---|---|
| `cmd/marz/` | The CLI (`scan`, `probe`, `check`, `fix fonts`, `rules`) |
| `cmd/marzwasm/` | The WebAssembly build behind the website's live demo |
| `internal/kb/rules.json` | The knowledge base: services, per-country status, sources, alternatives |
| `internal/scan/` | Finds URLs, hostnames and dependencies in a project |
| `internal/probe/` | DNS, TCP, TLS and HTTP checks from the current network |
| `internal/ooni/` | Live OONI measurements |
| `internal/report/` | Text, JSON, Markdown and SARIF output |
| `testdata/demo/` | The demo app used in the README and tests |

## Rules for knowledge base entries

1. Every hard verdict (`blocked`, `sanctioned`, `unavailable`) needs a **public source** marz can link to.
2. If nobody has published reliable data, the status is `unknown`. Guessing is worse than saying so.
3. Date the entry, and add a test in `internal/kb/kb_test.go` if you add a new matching rule.

## Commits

Small commits, one change each, with a title that says **why** the change was made.

## Security

Please report vulnerabilities privately via [GitHub security advisories](https://github.com/mrzroot/marz/security/advisories/new).
