---
schema_version: 1
id: "iss-2610262030273891"
slug: "meta-says-embark-writes-the-new-repo-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 3 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/00-meta.md:42"
remedy: "Rewrite the /abcd:embark bullet in 00-meta.md (line 42) to say embark verifies the lifeboat and writes its five record families plus the marker block into the target, and never writes brief/**; drop the 'amended press release becomes the new repo's initial brief' clause and re-check 'one canonical skeleton, used three ways' (line 44)."
finding_ref: cfk-893a6122bdfde0cb
---

Meta says embark writes the new repo's brief by reversing the mapping; the embark chapter says only the record families land and everything else is never written

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/00-meta.md:42` — "writes a new repo's brief by walking the brief↔lifeboat mapping in reverse — the amended press release becomes the new repo's initial brief."
- End B: `.abcd/development/brief/04-surfaces/03-embark.md:201` — "the five record families land at their canonical locations under the target, the current abcd marker block is re-injected into the target `AGENTS.md` (or nowhere, when the target's setup chose no conventions file), and everything else in the lifeboat informs the report but is never written."

00-meta's brief/lifeboat contract says embark writes a new repo's brief by walking the mapping in reverse, with the amended press release as the initial brief. The embark chapter's acceptance says only the five record families and the marker block land, and everything else in the lifeboat only informs the report. So the 'one skeleton, used three ways' contract has no embark leg as the surface is written.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 3.
