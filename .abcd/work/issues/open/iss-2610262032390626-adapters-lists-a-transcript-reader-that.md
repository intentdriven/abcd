---
schema_version: 1
id: "iss-2610262032390626"
slug: "adapters-lists-a-transcript-reader-that"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 300 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/02-adapters.md:350"
remedy: "none (filed automatically)"
finding_ref: cfk-5df5eaf7ecefb0c4
---

adapters lists a transcript reader that disembark packs from; embark says no source tier reads a transcript store

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/02-adapters.md:350` — "Reads the root-SHA-keyed local corpus; merge by timestamp/content hash when an imported specstory source is also present"
- End B: `.abcd/development/brief/04-surfaces/03-embark.md:177` — "No source tier this build packs reads a transcript store"

The adapters chapter says disembark reads into the lifeboat through source readers, including a transcript reader over the root-SHA-keyed local corpus. The embark chapter says no source tier this build packs reads a transcript store, which is why every pack declares the pass B exemption. If the transcript reader existed, that exemption would be false.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 300.
