---
schema_version: 1
id: "iss-2611060809247857"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/37-reserve.md"
remedy: "Rewrite the sentence to say an exposed folder this account owns is narrowed first, and that one another account owns refuses."
---

`.abcd/development/brief/04-surfaces/37-reserve.md:81` says the holder-id mint judges each folder, so that abcd's home folder writable by another account refuses before anything is minted or pushed. For a folder this account owns, every command first takes the group and other bits off it (`narrowHome`, `internal/surface/cli/homestop.go:60`; `NarrowToOwner`, `internal/fsutil/home.go:837`), so `mintHolderID` (`internal/core/claim/holderid.go:408`) never sees it exposed and no refusal follows. The refusal is left only for a folder another account owns.
