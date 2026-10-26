---
schema_version: 1
id: "iss-2610262037582823"
slug: "the-build-next-intent-says-max-and"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 762 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/35-drain.md:254"
remedy: "none (filed automatically)"
related_intents: [itd-2609211116005482]
finding_ref: cfk-181abe162a79de7a
---

The build-next intent says --max and --until-empty are flags that next and drain share; the drain chapter's run ends at its cap and its shipped surface has no --until-empty.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/35-drain.md:254` — "When the drain has opened as many lanes as its cap, it ends and says so."
- End B: `.abcd/development/intents/shipped/itd-2609211116005482-abcd-build-next-picks-the-readiest.md:68` — "`--max <n>` and `--until-empty` are the two flags both share"

itd-2609211116005482 decision 4 rules that --max < n> and --until-empty are the two flags that build next and drain share, with drain reading all by default. 35-drain.md bounds a drain by a cap on the lanes it opens and ends the drain when that cap is reached. Its generated surface appendix lists --dry-run, --fix-rounds, --judgement, --max, --pace, --sub-agents and --take-over, with no --until-empty. One document promises drain a flag that the other's surface does not carry.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 762.
