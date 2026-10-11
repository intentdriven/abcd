---
schema_version: 1
id: "iss-2610032205304585"
slug: "abcd-creates-its-home-folder-abcd-with"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "dashboard design security review, 2026-10-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/store.go"
remedy: "Create ~/.abcd through one helper that always uses 0o700 (the credential store's mode), and make each store's own files 0600; a test asserts the home's mode whichever verb creates it first."
resolution: "Every writer creates abcd's home and each folder in it at abcdhome.DirMode (0700) whichever command runs first, the hook's bootstrap included (mkdir -m 0700); the path-entry record and the registry's index are written abcdhome.FileMode (0600), and one an earlier version wrote 0644 is narrowed at its next write; an existing home keeps its mode (DECISIONS.md 2026-10-04)."
impact: fix
resolved_by:
  commit: "d6897d03e"
---

abcd creates its home folder ~/.abcd with different permissions depending on which command creates it first: the credential store makes it private to the account (0o700), while ahoy's history store and owned copy make it readable by other accounts on the computer (0o755). On a Mac shared by several accounts, whichever runs first decides whether others can list abcd's home. Found by the dashboard design's security review.

## Grounds

- pursued: TestEveryHomeWriterMakesTheHomePrivate holds every home-creating call (14 judged) to abcdhome.DirMode with an armed twin, TestHomeWritersMakeTheHomePrivate and TestBootstrapMakesTheHomePrivate assert 0700/0600 on a fresh home, TestAnEarlierIndexIsNarrowedAtItsNextWrite the narrowing; a home created 0755 by any command first, or a record left 0644 after its rewrite, would show it wrong. Two security reviews (Fable) closed every finding.
