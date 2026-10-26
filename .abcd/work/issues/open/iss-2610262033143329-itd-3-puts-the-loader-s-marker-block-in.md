---
schema_version: 1
id: "iss-2610262033143329"
slug: "itd-3-puts-the-loader-s-marker-block-in"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 378 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/03-embark.md:132"
remedy: "none (filed automatically)"
related_intents: [itd-3]
finding_ref: cfk-b15ad49da556f2d5
---

itd-3 puts the loader's marker block in CLAUDE.md; embark re-injects that same itd-3 block into AGENTS.md and treats claude_md as a retired target

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/03-embark.md:132` — "saved gives `AGENTS.md`; `skip` gives no file; a retired `claude_md` or"
- End B: `.abcd/development/intents/shipped/itd-3-modular-rules-loader.md:16` — "CLAUDE.md becomes a small marker block, not a 1000-line wall."

Embark names the block it re-injects as the modular-rules-loader block of itd-3. It writes that block into `AGENTS.md`, or nowhere, and treats a saved `claude_md` choice as retired. itd-3 says the loader turns CLAUDE.md into a small marker block owned by ahoy. A reader following itd-3 for where the block lives would write the file embark refuses to write.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 378.
