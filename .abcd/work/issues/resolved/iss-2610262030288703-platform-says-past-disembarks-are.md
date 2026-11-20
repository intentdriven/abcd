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
remedy: "Edit 01-platform.md (lifeboat path paragraph): say past disembarks are recorded as minimal voyage entries (manifest hash, source name and root SHA, destination, file and byte counts), with no label and no file list."
finding_ref: cfk-5e172db90c736112
resolution: "The platform page's lifeboat path paragraph now says past disembarks are recorded as minimal voyage entries (manifest hash, source name and root SHA, destination, file and byte counts), with no label and no file list, matching the embark chapter's section 7 and voyageEntry; a test in internal/core/lifeboat holds the page to it."
impact: internal
resolved_by:
  commit: "e32f9af3721b4ffd197e00e3f40fd4b7b4ef3b9c"
---

Platform says past disembarks are recorded with a label; the voyage-log chapter says the entry carries no label

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/01-platform.md:23` — "Past disembarks are recorded as manifests (hash + file list + label) at the operator level, not as preserved snapshots"
- End B: `.abcd/development/brief/04-surfaces/03-embark.md:302` — "Not a verdict, a label, or a recipient list."

The platform page says past disembarks are recorded as manifests with a hash, a file list and a label. The embark chapter's voyage layout says the deliberately minimal entry carries no verdict, no label and no recipient list, and its example entry has counts, not a file list. A reader cannot rely on both field sets.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 5.

## Grounds

- pursued: the platform page and the embark chapter now name the same voyage-entry fields as voyageEntry in voyage.go; TestBriefPlatformPageNamesTheVoyageEntryFields would show it wrong by failing if the page brings back a label or file list, or if voyageEntry gains a label or file-list field.
