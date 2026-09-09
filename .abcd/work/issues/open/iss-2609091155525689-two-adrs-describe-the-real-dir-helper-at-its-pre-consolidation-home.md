---
schema_version: 1
id: "iss-2609091155525689"
slug: "two-adrs-describe-the-real-dir-helper-at-its-pre-consolidation-home"
severity: "nitpick"
category: "tech-debt"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/decisions/adrs/2609090717039680-the-transcript-corpus-is-a-sibling-store-that-creates-itself.md"
---

Two accepted decision records describe the directory primitive as it stood before the consolidation, and this repository supersedes rather than amends, so each owes a correction rather than an edit. adr-2609090717039680 (the transcript corpus is a sibling store that creates itself) names ensureRealDir as a helper of internal/core/history and presents its behaviour — walks the chain top-down, os.Mkdir per level rather than MkdirAll, re-verifies every level on every resolve — as a property of that package. Every one of those behaviours still holds, and the symbol does not: it is fsutil.EnsureRealDir, called in a loop, and the package that used to own the sequence now owns only the layout. A reader following the ADR to the code finds nothing at the name it gives. adr-2609091014087993 (a tool never creates directories in user-owned project space) commits a future machine-scoped worktree store to creating itself one level at a time, never through a symlink, exactly as the transcript store does. That sentence is now satisfied by calling the canonical primitive, and it was the reading of it that would have produced a fourth copy: it describes a mechanism where it could name the thing that implements it. What is owed is one correction record covering both — the mechanism is unchanged, the home moved, and a store that wants this behaviour calls internal/fsutil rather than reproducing it. Detector: an ADR that names a package-local symbol should not survive that symbol's removal, which no gate currently checks; the record-lint rule family has no cross-reference from decision prose to Go identifiers.
