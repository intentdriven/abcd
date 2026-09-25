---
schema_version: 1
id: "iss-2609251638574543"
slug: "the-generic-account-floor-misses-a-windows-home-root-escaped"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/identity.go"
---

The generic-account floor misses a Windows home root escaped more than once. accountRootPrefixes (internal/adapter/scanner/identity.go) lists the single and the doubled backslash spellings of \users\ and the home-literal clause of standsAsAccountName lists the home and its doubled spelling, so C:\\\\Users\\\\LOGIN\\\\Desktop, the shape a transcript line carries when a tool result is itself JSON text (go env -json, npm config ls --json, any --json output), raises no finding for a login on the generic list while the doubled spelling hard-fails. Before the generic floor the bare word was flagged, so the floor narrowed a hard_fail rule in the redactor input: capture and history redact nothing on such a line.
