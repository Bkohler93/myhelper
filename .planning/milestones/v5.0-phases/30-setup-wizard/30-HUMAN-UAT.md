---
status: partial
phase: 30-setup-wizard
source: [30-VERIFICATION.md]
started: 2026-05-10T00:00:00Z
updated: 2026-05-10T00:00:00Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. Wizard on machine without Ollama
expected: Platform-specific install instructions display (brew on macOS, curl on Linux/WSL) and wizard exits cleanly
result: [pending]

### 2. Wizard with Ollama running + pull confirmation
expected: NDJSON streaming progress displays during pull, model key written to ~/.config/myhelper/config.json after successful pull
result: [pending]

### 3. Tavily key write to real config
expected: Key written to ~/.config/myhelper/config.json with tavily_key field, pre-existing keys preserved, file has 0600 permissions
result: [pending]

### 4. SearXNG endpoint validation
expected: Bare hostname (e.g. "192.168.0.9:8083") triggers a warning, valid https:// URL is accepted and written to config
result: [pending]

## Summary

total: 4
passed: 0
issues: 0
pending: 4
skipped: 0
blocked: 0

## Gaps
