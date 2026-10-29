---
schema_version: 1
id: "iss-2610262034191519"
slug: "itd-44-writes-the-diverted-adr-as-adr"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 487 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-44-fourth-intent-kind-decision.md:122"
remedy: "In itd-44 replace capture writing `adr-<next-free>-<slug>.md` with a call to `abcd decide --adr`, which mints a timestamp id and names the file by the stamp then the slug (24-decide.md, adr-45); in 05-intent.md line 93 stop naming new ADR files `NNNN-<slug>.md` zero-padded (the ordinals 0001-0058 keep that shape, every newer ADR is stamp-named)."
related_intents: [itd-44]
finding_ref: cfk-968326c60e3149f0
---

itd-44 writes the diverted ADR as `adr-<next-free>-<slug>.md`; the intent chapter names the same ADR-store file `NNNN-<slug>.md`, zero-padded.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-44-fourth-intent-kind-decision.md:122` — "`adr-<next-free>-<slug>.md` into `.abcd/development/decisions/adrs/`"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:93` — "(`.abcd/development/decisions/adrs/`, `NNNN-<slug>.md`, zero-padded)"

Both documents describe the same diversion: a confirmed `decision` verdict is written into `.abcd/development/decisions/adrs/` instead of an intent draft. They give that file different shapes. itd-44 uses an `adr-` prefix and the next free number, while 05-intent.md uses a bare zero-padded `NNNN-` number. A builder cannot write both.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 487.
