---
schema_version: 1
id: "iss-2610262032297115"
slug: "ahoy-s-setup-mints-the-drain-rule-as-an"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 276 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:766"
remedy: "none (filed automatically)"
finding_ref: cfk-168e0ed0b5ba441b
---

ahoy's setup mints the drain rule as an accepted record carrying drain_ fields through the decision store's seam; decide says a minted record lands empty and proposed, and names embark, a restore, as the only other writer

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:766` — "abcd's strict baseline as an accepted decision record carrying the four `drain_` fields, minted through the decision store's seam"
- End B: `.abcd/development/brief/04-surfaces/24-decide.md:160` — "the record lands with `status: proposed`, because a freshly minted file is a draft whose decision is not yet locked and only its author can say otherwise."

01-ahoy.md's drain-rule offer writes abcd's strict baseline as an accepted decision record carrying the four drain_ fields, minted through the decision store's seam. 24-decide.md says the store's mint writes an empty record that lands with status: proposed, because only its author can say otherwise. It names embark, a restore rather than a mint, as the store's only other writer, and a decision (dcn) record carries no status and no extra fields. No writer decide describes produces the accepted, field-carrying record ahoy says it mints, and decide does not list setup as a minting writer.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 276.
