---
schema_version: 1
id: "iss-2610262033217382"
slug: "itd-89-files-sessionend-transcripts-in"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 394 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-89-start-the-transcript-clock.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-89]
finding_ref: cfk-25b4d25db5aef0b2
---

itd-89 files SessionEnd transcripts in the local per-repo store; 01-ahoy keeps the per-repo transcripts folder for opted-in checkouts only

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-89-start-the-transcript-clock.md:18` — "files it in the local per-repo store"
- End B: `.abcd/development/brief/04-surfaces/01-ahoy.md:515` — "local-transcript-roots checkouts whose transcripts are pulled in to < repo>/.abcd/.work.local/transcripts/ instead"

itd-89 says the SessionEnd hook files each transcript in the local per-repo store. 01-ahoy lists the default corpus as `transcripts/<root-sha>/` in the user-scope `~/.abcd.noindex/` home, and says that only checkouts named in `local-transcript-roots` have their transcripts pulled into `<repo>/.abcd/.work.local/transcripts/` instead. A default capture cannot land in both the per-repo store and the user-scope corpus.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 394.
