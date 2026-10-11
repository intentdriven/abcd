---
schema_version: 1
id: "iss-2608270500209352"
slug: "abcd-capture-kebab-cases-the-home-path"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "github-ledger-dedup-2026-08-27"
found_at: "internal/core/capture"
resolution: "capture redacts input before deriving the issue slug, so a home-path username never lands in the filename (#485)"
impact: fix
---

abcd capture kebab-cases the home path into the slug/filename before redaction, so the username still lands in the issue filename even though the body is redacted. GitHub mirror: #485