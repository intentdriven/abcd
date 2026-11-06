---
schema_version: 1
id: "iss-2611060810014724"
slug: "17-guard-md-335-says-a-question-without"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/17-guard.md"
remedy: "Say such a question is refused on its header and writes no marker, or admit it if marking it open is the intent."
---

`17-guard.md:335` says a question without abcd's chip, while the mode names somebody, is held to the field limits and marked open. The header rule always refuses it: any header outside the chip grammar is a refusing finding (`internal/core/question/check.go:190`), so `""`, `Q1`, `Pick` and `Pick me` were all denied and no marker was written. Lines 302-305 of the same chapter already describe this refusal.
