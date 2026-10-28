---
schema_version: 1
id: "iss-2610262030288703"
slug: "platform-says-past-disembarks-are"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 5 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/01-platform.md:23"
remedy: "none (filed automatically)"
finding_ref: cfk-5e172db90c736112
---

Platform says past disembarks are recorded with a label; the voyage-log chapter says the entry carries no label

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/01-platform.md:23` — "Past disembarks are recorded as manifests (hash + file list + label) at the operator level, not as preserved snapshots"
- End B: `.abcd/development/brief/04-surfaces/03-embark.md:302` — "Not a verdict, a label, or a recipient list."

The platform page says past disembarks are recorded as manifests with a hash, a file list and a label. The embark chapter's voyage layout says the deliberately minimal entry carries no verdict, no label and no recipient list, and its example entry has counts, not a file list. A reader cannot rely on both field sets.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 5.
