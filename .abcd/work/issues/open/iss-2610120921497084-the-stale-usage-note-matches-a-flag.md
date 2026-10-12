---
schema_version: 1
id: "iss-2610120921497084"
slug: "the-stale-usage-note-matches-a-flag"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "cloud-candidate survey, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/staleusage.go"
remedy: "Scope surfaceDocuments' flag match to the typed sub-verb's synopsis or usage block on the page, and fall through to the plain unknown-flag error when only prose elsewhere mentions it"
---

The stale-usage note matches a flag anywhere on the verb's command page, not under the sub-verb typed: build next --dry-run was called documented because commands/build.md mentions '/abcd:drain --dry-run' in prose, so the note blamed the binary's age for a flag build next never had.
