---
schema_version: 1
id: "iss-2610262044419558"
slug: "the-review-bar-draft-says"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1223 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-83-review-bar-fires-itself.md:66"
remedy: "none (filed automatically)"
related_intents: [itd-83, itd-2610090831227812]
finding_ref: cfk-e576eed6bf3a5622
---

The review-bar draft says prepare-this-repo owns the AGENTS.md conventions section, but prepare-this-repo has folded into ahoy install

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-83-review-bar-fires-itself.md:66` — "a one-line pointer in the marked conventions section of `AGENTS.md` that `prepare-this-repo` already owns"
- End B: `.abcd/development/intents/shipped/itd-2610090831227812-a-person-who-types-abcd-sees-only-the.md:48` — "`prepare-this-repo` folds into `ahoy install`; `consult`, `ingest`, `memory` and `source` merge into `library`"

itd-83 places its host-agnostic fallback in the marked conventions section of AGENTS.md that it says `prepare-this-repo` owns. itd-2610090831227812 folds `prepare-this-repo` into `ahoy install` with no surviving name, so the owner the draft names no longer exists.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1223.
