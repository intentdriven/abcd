---
schema_version: 1
id: "iss-2610262036171679"
slug: "itd-44-has-capture-write-an-adr-as-adr"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 637 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/24-decide.md:127"
remedy: "none (filed automatically)"
related_intents: [itd-44]
finding_ref: cfk-3367b576bf40ac01
---

itd-44 has capture write an ADR as adr-< next-free>-< slug>.md; the decide chapter mints ADR ids from a timestamp that reads no maximum, and names the file < stamp>-< slug>.md.

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/24-decide.md:127` — "The mint reads **no maximum**: not the store's, not the"
- End B: `.abcd/development/intents/drafts/itd-44-fourth-intent-kind-decision.md:122` — "`adr-<next-free>-<slug>.md` into `.abcd/development/decisions/adrs/`"

24-decide says an ADR id is adr-< yymmddHHMMSS>< rrrr> and the mint reads no maximum, neither the store's nor the citations'. It also says a hand-numbered ordinal allocated by reading the directory is exactly the collision the ruling closed. itd-44 allocates adr-< next-free> and writes adr-< next-free>-< slug>.md into the same store. That is allocation by reading the directory, and a filename shape the store does not use. Building itd-44 as written brings back the duplicate-number collision.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 637.
