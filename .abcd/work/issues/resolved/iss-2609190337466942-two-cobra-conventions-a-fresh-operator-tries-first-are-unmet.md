---
schema_version: 1
id: "iss-2609190337466942"
slug: "two-cobra-conventions-a-fresh-operator-tries-first-are-unmet"
severity: "nitpick"
category: "ux"
source: "agent-observation"
found_during: "Gropius autonomous sweep, session gropiusllm-66, relayed to abcd-17 on 2026-09-19"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
resolution: "abcd --version already works at base (7b579ca5, itd-2609212130136102); the intent, capture and spec refusals for a status or show sub-verb now name the record dispatcher abcd <record-id>, filled with the id given."
impact: fix
resolved_by:
  commit: "996f97a56e401980e8df92c07e05b67e30d5d571"
---

Two cobra conventions a fresh operator tries first are unmet: abcd --version is an unknown flag, and abcd intent status <itd-N> is an unknown sub-verb. Reproduced at v0.9.0 (4ae6f221). The version verb exists (abcd version), and the record dispatcher answers the status question (abcd <itd-N> prints the bucket, the path and the next move), but neither refusal points at the form that works: --version refuses with "unknown flag" alone, and the intent refusal lists the five sub-verbs and says nothing of abcd <itd-N>. Relayed from the Gropius session gropiusllm-66 on 2026-09-19 (a forty-lane autonomous sweep), where both were the first thing tried. Wanted: accept --version as an alias of the version verb (cobra's Version field does it in one line), and have the unknown-sub-verb refusal name abcd <itd-N> when the rejected word is status or show.

## Grounds

- pursued: the first thing a fresh operator tries is answered with the form that works; a status or show refusal under intent, capture or spec that does not name abcd <record-id> would show it wrong
