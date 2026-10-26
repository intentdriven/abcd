---
schema_version: 1
id: "iss-2610240932493158"
slug: "related-intents-on-an-issue-carries-two"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "/abcd:build iss-2610140652230444 session, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/promote.go"
remedy: "Waits on a ruling of the technical facilitator (narrows B3, dcn-2609230000000016, and with it what invariant 19 holds back): keep the two-way pair, and let the intent's side say which issues it delivers and which it cites as evidence, with a key that is not one of the retired promoted_to or promoted_from; only a delivered issue counts as promoted for capture promote and the drain alike, an evidence-only issue is promotable, and today's pairs are classified by a person, never by the migration."
---

The two-way pair that marks an issue as promoted (the issue's `related_intents` naming an intent that names it back in `related_issues`) cannot tell an issue the intent delivers from an issue the intent only cites as evidence, so an evidence-only issue reads as promoted and `capture promote` refuses it.

`capture promote` counts an issue promoted when the pair exists (`internal/core/capture/promote.go:176-180`, `:344-385`), as ruling B3 chose (dcn-2609230000000016, item 2: the pair, because `related_intents` already carries loose relations; item 1 retired `promoted_to` and `promoted_from`, which the ledger reader now refuses and `.abcd/record-lint.json:188` bans). The drain reads any `related_intents` entry as promoted (`internal/core/capture/eligible.go:212-223`, except a consistency finding's intents under H12, `:212-214`), and brief invariant 19 (`.abcd/development/brief/02-constraints/03-invariants.md:172`) holds such an issue back "whatever the record says". A person's own `build <iss-N>` sets the promoted rule aside when no other rule holds the issue (`eligible.go:127-132`, `:486`; dcn-2610161428244461, refined by dcn-2610161519047899).

Case of 2026-10-24: iss-2610140652230444 (the review set as a setting) lists itd-2610161733316518 (the model matrix) because its evidence tables feed that intent, and the intent names it back in `related_issues`. The intent does not deliver the issue's remedy. `build` handed the issue back on its `process` category (a separate rule), and `capture promote` refused it as "already promoted to itd-2610161733316518", so it reached an intent only through a hand-filed draft (itd-2610240922461281) joined to it by `refines`.

Reversal flags (advisory, for the technical facilitator): the remedy narrows ruling B3's reading of the pair, and whatever it changes in what counts as promoted changes what invariant 19 holds back from an unattended drain, a trust rule; that part belongs in a decision record before any code.
