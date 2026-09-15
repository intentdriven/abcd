---
schema_version: 1
id: "iss-2609152131205316"
slug: "the-outbound-text-check-shipped-with-no-record-so-the-releas"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "docs-currency review of the v0.9.0 content commit"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/lint_outbound.go, commands/lint.md"
resolution: "abcd lint outbound judges a commit message, pull-request body or release note against the outbound policy and refuses one carrying a session URL or a tool footer, without rewriting it; CI runs it over every message in a pull request's range and over the body"
impact: additive
resolved_by:
  commit: "f8541182"
---

The outbound text check shipped with no record, so the release notes could not name it

`abcd lint outbound` judges one piece of outbound text — a commit message, a pull-request body, a release note — against the outbound policy (never a live agent-session URL, never a tool's attribution footer), reports and refuses without rewriting, and is what CI runs over every commit message in a pull request's range and over the pull-request body. It landed in the same change that fixed the session-URL leak into commits, under records that name the leak and not the verb, so the derived changelog, which composes only from terminal records, had no line for a user-facing verb the cycle shipped. The release record must name what shipped; this record exists so it can.

## Grounds

- pursued: a shipped user-facing verb needs a terminal record for the derived changelog to name it, and the verb is what makes the outbound policy enforceable before a leak reaches a forge; what would show it wrong is a reader learning of the verb only from the command reference
