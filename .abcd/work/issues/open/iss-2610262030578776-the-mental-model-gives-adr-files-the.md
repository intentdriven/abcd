---
schema_version: 1
id: "iss-2610262030578776"
slug: "the-mental-model-gives-adr-files-the"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 66 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:74"
remedy: "Edit 03-mental-model.md (the fourth-capture-verdict paragraph): name the ADR store's files `<stamp>-<slug>.md` (ordinals `NNNN-<slug>.md` for 0001-0058), not `adr-N-<slug>.md`."
finding_ref: cfk-bc38578767a69419
---

The mental model gives ADR files the shape `adr-N-<slug>.md`; the decide chapter says an ADR's filename is its stamp followed by its slug.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:74` — "routes capture to the existing **ADR store** (`.abcd/development/decisions/adrs/`, `adr-N-<slug>.md`)"
- End B: `.abcd/development/brief/04-surfaces/24-decide.md:134` — "The filename is the stamp followed by the slug"

03-mental-model names the ADR store's files `adr-N-<slug>.md`. 24-decide describes the store's minting verb and says the filename is the stamp followed by the slug. It also says the ordinals `0001`–`0058` keep their filenames, so no ADR file carries an `adr-` prefix. A reader looking up an ADR by the mental model's pattern would find no file.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 66.
