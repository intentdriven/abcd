---
schema_version: 1
id: "iss-2610262036446593"
slug: "the-conventions-file-intent-removes-the"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 670 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-2610030814013772-evaluate-whether-claude-md-can-be.md:84"
remedy: "none (filed automatically)"
related_intents: [itd-2610030814013772]
finding_ref: cfk-db646f5068efb090
---

The conventions-file intent removes the CLAUDE.md and GEMINI.md links; the docs chapter still justifies its root-symlink rule by the bridge files it lets pass.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-2610030814013772-evaluate-whether-claude-md-can-be.md:84` — "this repository's CLAUDE.md and GEMINI.md links are removed whichever way the rest falls"
- End B: `.abcd/development/brief/04-surfaces/10-docs.md:160` — "what lets the bridge files pass while appearing in no allowlist"

The intent removes this repository's CLAUDE.md and GEMINI.md links, and has abcd write no separate conventions copy for any tool. The docs chapter judges a root markdown symlink by its target's stem, and gives as the reason that this is what lets the bridge files pass while appearing in no allowlist. That reason assumes bridge files still stand at the root, and the intent says they were removed.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 670.
