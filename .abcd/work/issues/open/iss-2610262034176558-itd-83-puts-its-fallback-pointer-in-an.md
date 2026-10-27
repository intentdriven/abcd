---
schema_version: 1
id: "iss-2610262034176558"
slug: "itd-83-puts-its-fallback-pointer-in-an"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 484 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-83-review-bar-fires-itself.md:67"
remedy: "none (filed automatically)"
related_intents: [itd-83]
finding_ref: cfk-a1c4c344f292ecc9
---

itd-83 puts its fallback pointer in an AGENTS.md section that `prepare-this-repo` owns; the board chapter says the `prepare-this-repo` page retired into the install.

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-83-review-bar-fires-itself.md:67` — "of `AGENTS.md` that `prepare-this-repo` already owns"
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:459` — "`prepare-this-repo`, whose page retired into"

itd-83's fallback for hosts that do not run hooks depends on a marked conventions section of AGENTS.md that `prepare-this-repo` already owns. 08-abcd.md lists `prepare-this-repo` among the names retired with no alias, saying its page retired into the install. The dependency names an owner the brief no longer has as a surface, so the fallback has no surface to live on as written.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 484.
