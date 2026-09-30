---
schema_version: 1
id: "iss-45"
slug: "surface-layer-boundary"
severity: "minor"
category: "architectural-insight"
source: "agent-finding"
found_during: "2026-07-08 multi-agent review"
found_at: "internal/surface/cli/cli.go"
deferred_after: "v0.11.1"
deferral_reason: "two of the three leaks are fixed at the v0.11.1 anchor: slug derivation lives in core (d0a1ebfd7; the CLI passes an empty slug and core derives it from the redacted text), and consult and ingest run through abcd source verbs, not home-directory scripts. Owed: a ruling on the third, whether cmd/record-lint (tested now, main_test.go) folds into abcd as one binary front door, and on the boundary detector; a lane once ruled."
remedy: "Waits on ruling (fold cmd/record-lint into abcd as one front door): if folded, expose record-lint as an abcd lint verb, point the Makefile and CI at it, and delete cmd/record-lint once its main_test.go cases pass against the verb; if kept, record why two binary front doors stay; either way the slug and consult/ingest leaks are closed at the v0.11.1 anchor per the deferral, and the boundary detector goes to its own intent."
---

surface-layer boundary leaks: slug derivation — business logic every front door needs — lives in the CLI surface (internal/surface/cli/cli.go:623) so each future front door must reimplement it; cmd/record-lint is a second binary front door duplicating abcd docs lint --config --root, and as a CI gate it has no tests; skills/consult and skills/ingest bypass the abcd binary entirely, their engine being unversioned scripts in the user home (converges with iss-27 corpus-tooling absorption and the script-first-mvp bounds). Detector: a transport-agnostic-core boundary check — decisions live below the surface, front doors only format; one binary front door per capability. Acceptance corpus: the three leaks above.

## Remedy grounds (2026-09-29)

Why: the deferral records two of the three leaks as fixed, so the remedy is scoped to the one fork left. Rejected: bundling the boundary detector into this record, which would keep a closed-out finding open on a detector with no acceptance corpus beyond the one leak left.
