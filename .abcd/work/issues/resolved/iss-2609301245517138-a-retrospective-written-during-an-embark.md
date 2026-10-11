---
schema_version: 1
id: "iss-2609301245517138"
slug: "a-retrospective-written-during-an-embark"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lifeboat/embark.go"
remedy: "reflect.Write creates the file under intent.WithMintLock, the intent store's lock embark holds through WithLedgerThenMintLock, so the two are serialised; and writeEmbark writes every planned create through an exclusive create (fsutil.CreateExclusiveIn), so a file that lands after the rejudge from a writer holding no lock fails the embark loudly instead of being replaced."
resolution: "reflect.Write creates the retrospective under the intent store's lock, the lock embark writes under, and writeEmbark writes every planned create exclusively, so a late arrival fails the embark loudly instead of being replaced."
impact: fix
resolved_by:
  commit: "552c542d1"
---

A retrospective written during an embark is replaced silently: reflect write takes no lock, and embark's writeEmbark writes each planned create through a rename-over (writeIntoLifeboat, fsutil.WriteFileAtomic), so a retrospective reflect write creates between embark's rejudge and its write is replaced by the lifeboat's copy with no conflict.

## Grounds

- pursued: a reflect write arriving while embark holds the intent lock waits for it (TestWriteWaitsForTheIntentStoresLock), and writeEmbark over a pre-existing differing README refuses with the file intact (TestWriteEmbarkRefusesToReplaceAFileThatLandedAtACreateTarget); a retrospective replaced by an embark's copy with a nil error would show it wrong.
