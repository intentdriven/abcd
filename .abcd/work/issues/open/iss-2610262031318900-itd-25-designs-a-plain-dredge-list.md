---
schema_version: 1
id: "iss-2610262031318900"
slug: "itd-25-designs-a-plain-dredge-list"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 140 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:37"
remedy: "none (filed automatically)"
related_intents: [itd-25]
finding_ref: cfk-3a62e80805758987
---

itd-25 designs a plain dredge list sub-verb, which the bare-command-as-render discipline forbids

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:37` — "`dredge list` — show recent synthesis reports from `.abcd/logbook/dredge/`"
- End B: `.abcd/development/brief/03-evidence/04-tradeoffs.md:61` — "`<verb> show`, `<verb> stats`, `<verb> list` (plain), `<verb> view` are forbidden — they name what bare already does."

itd-25 gives /abcd:dredge a dredge list sub-verb that shows recent synthesis reports and takes no positional and no filter. The recorded discipline says < verb> list (plain) is forbidden because it names what the bare invocation already renders. The draft's surface breaks a rule the brief records as chosen.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 140.
