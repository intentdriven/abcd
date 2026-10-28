---
schema_version: 1
id: "iss-2610262036113224"
slug: "the-history-chapter-puts-the-transcript"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 630 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/11-history.md:15"
remedy: "none (filed automatically)"
related_intents: [itd-59]
finding_ref: cfk-26b7c7fe587a560b
---

The history chapter puts the transcript corpus under ~/.abcd.noindex/transcripts/ and keeps ~/.abcd.noindex/history/ for ahoy's registry, holding no transcripts; itd-59 places the corpus at ~/.abcd.noindex/history/< root_sha>/.

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/11-history.md:15` — "ahoy's registry stays under `~/.abcd.noindex/history/` and holds no transcripts."
- End B: `.abcd/development/intents/drafts/itd-59-autonomous-worker-transcript-capture.md:19` — "an operator's interactive session is redirected into the native transcript corpus at `~/.abcd.noindex/history/<root_sha>/`"

11-history says the store is ~/.abcd.noindex/transcripts/< root-sha>/records/ and that ahoy's registry stays under ~/.abcd.noindex/history/ and holds no transcripts. itd-59 says interactive sessions are redirected into the native transcript corpus at ~/.abcd.noindex/history/< root_sha>/, and its whole scope is to feed that directory. Both cannot say where the corpus lives. Building itd-59 as written would put transcripts into the registry directory that the brief says holds none.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 630.
