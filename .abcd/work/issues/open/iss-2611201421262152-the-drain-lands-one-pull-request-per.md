---
schema_version: 1
id: "iss-2611201421262152"
slug: "the-drain-lands-one-pull-request-per"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "person's question on 20 Nov 2026 during the autonomous run of 15 Nov"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain.go"
remedy: "Let the drain bundle eligible issues of one kind and one area into one lane, not only overlapping files: issues whose fix is a docs or brief sentence plus the test that pins it, in one brief chapter folder or one package, up to a cap set by drain.bundle_max with a default of five (dcn-2611201429107457). One branch, one review pair, one preflight, one pull request with a Resolves line per issue; an issue that fails its own fix round is handed back without failing the rest. Measure it against these figures: pull requests, preflights and CI minutes per resolved issue over the next forty."
refines: [iss-2610161652341096]
---

The drain lands one pull request per issue, and on 19-20 Nov that was 40 pull requests in about 36 hours, each one to one and a half hours from opening to merge. Most are documentation fixes: 35 of the 40 change one brief or docs file, move the issue's record, and add a Go test that pins the corrected sentence (for example internal/core/lifeboat/platform_brief_test.go), so CI's fail-closed classifier reads them as code and runs the full check job (42 minutes on #1694) where a docs-only change runs it in 19 (#1698). Each issue also pays its own local make preflight (about 40 minutes), its own review pair and its own merge-queue entry. iss-2610161652341096 asks to bundle issues whose found_at paths overlap; most of these touch different files in the same place (the brief's chapters), so that rule alone would bundle few of them.
