---
phase: 13
slug: commands-flags
status: verified
threats_open: 0
asvs_level: 1
created: 2026-04-10
---

# Phase 13 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| test file → source scan | Tests read .go source files from disk via relative paths under `go test` | local source code, read-only |
| CLI flag → cfg.TokenThreshold | User-supplied `--token-limit` integer overwrites config value | integer scalar, no external data |
| CLI flag → retrieval bypass | `--no-context` bool skips all project artifact reads | boolean flag, reduces data flow |
| CLI arg → retrieval query | User-supplied query string passed to `BuildInspectContext` | user text, local processing only |
| disk → loadArtifacts | `.myhelper/*.json` files read from local filesystem | local project metadata |

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-13-01 | Tampering | test file source scan | accept | Tests read `.go` files relative to cwd; only runs under `go test` in controlled dev environment; no user-controlled paths | closed |
| T-13-02 | Tampering | `--token-limit` → `cfg.TokenThreshold` | accept | Integer flag; cobra parses and validates type; extreme values (e.g. 1) degrade quality but cause no security issues; no external data involved | closed |
| T-13-03 | Information Disclosure | `--no-context` bypass | accept | Flag omits project context from LLM prompt; reduces information sent to model, never increases it; no new data exposed | closed |
| T-13-04 | Information Disclosure | `inspect` stdout output | accept | `inspect` prints local `.myhelper/` artifact metadata to stdout — intended behavior; no new data exposed beyond what `init`/`sync` already wrote to disk | closed |
| T-13-05 | Denial of Service | `loadArtifacts` file read | accept | Files are bounded by local disk; no network I/O; same risk profile as existing `BuildContext`; graceful fallback on missing artifacts already implemented | closed |
| T-13-06 | Spoofing | `--no-context` flag bypass | accept | Flag prevents retrieval from reading `.myhelper/` artifacts; reduces information sent to model; user-controlled opt-out with no security downside | closed |

*Status: open · closed*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-13-01 | T-13-01 | Test source scan uses bare relative paths (WR-01 in REVIEW.md); runs only in dev environment under `go test`, no user-controlled input, no production code path | gsd-secure-phase | 2026-04-10 |
| AR-13-02 | T-13-02 | `--token-limit` with extreme low values degrades retrieval quality but has no security impact; cobra type validation prevents non-integer input | gsd-secure-phase | 2026-04-10 |
| AR-13-03 | T-13-03 | `--no-context` reduces context sent to model; all changes are subtractive | gsd-secure-phase | 2026-04-10 |
| AR-13-04 | T-13-04 | `inspect` outputs same metadata surface as `BuildContext`; no new attack surface introduced | gsd-secure-phase | 2026-04-10 |
| AR-13-05 | T-13-05 | `loadArtifacts` has same file-read profile as existing `BuildContext`; no new disk paths introduced | gsd-secure-phase | 2026-04-10 |
| AR-13-06 | T-13-06 | `--no-context` is a user opt-out that reduces data flow; no spoofing vector | gsd-secure-phase | 2026-04-10 |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-04-10 | 6 | 6 | 0 | gsd-secure-phase (all dispositions: accept, no implementation required) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-04-10
