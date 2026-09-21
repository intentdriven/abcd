---
schema_version: 1
id: "iss-2609211414208891"
slug: "a-reading-run-id-drawn-twice-in-one-second-refuses-the-second-run-instead-of-redrawing"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "PR 651's cold-reading-evals check, 2026-09-21"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/assemble.go (the run id mint before resolveOutDir)"
---

A reading run id drawn twice in one second refuses the second run instead of redrawing. The run id is the family tag, a second-resolution stamp and a uniform four-digit draw (adr-45), and reading assemble derives the default output directory from it; when two assemblies in the same second draw the same suffix, the second finds the first's directory non-empty and exits 2 with the one-run-one-directory refusal, though nothing about the second run was wrong. The rehearsal eval assembles several runs within one second, so the one-in-ten-thousand coincidence per same-second pair is met in CI: TestRehearseTheOpeningRunLoop/empty-output-commits-a-clean-run failed on PR 651 at 12:09Z with rdg-2609211209406944 already holding two entries, on a docs-only branch whose local preflight had passed twice. adr-45 leaves the same-second coincidence to the armed uniqueness detectors to assert against, and here the detector is the empty-directory refusal, which asserts by refusing the innocent run. Wanted: when the run will land in the default directory, the mint redraws while the drawn directory exists (a bounded number of draws, then a refusal that names the collision), so the detector's residue is absorbed at the one site that can tell a collision from an operator's occupied directory; an operator-named --out that is non-empty keeps refusing as it does.
