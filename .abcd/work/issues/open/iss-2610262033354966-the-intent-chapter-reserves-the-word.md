---
schema_version: 1
id: "iss-2610262033354966"
slug: "the-intent-chapter-reserves-the-word"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 422 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:534"
remedy: "none (filed automatically)"
finding_ref: cfk-38353fa1648aa045
---

The intent chapter reserves the word review for change-judgement, while universal patterns calls the audit and consistency sub-verbs review sub-verbs.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:534` — "the four-bucket vocabulary reserves *review* for family-1 change-judgement"
- End B: `.abcd/development/brief/05-internals/04-universal-patterns.md:130` — "registered review sub-verbs of /abcd:intent"

05-intent says the four-bucket vocabulary reserves review for family-1 change-judgement, and its table files audit and consistency under the audit bucket. 04-universal-patterns calls audit and consistency the registered review sub-verbs of /abcd:intent. It applies the reserved word to the promise-versus-reality verbs that the vocabulary keeps out of it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 422.
