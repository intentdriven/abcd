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
remedy: "Let the drain bundle eligible issues of one kind and one area into one lane, not only overlapping files: issues of one category whose fix is a docs or brief sentence plus the test that pins it, in one brief chapter folder or one package, up to a cap set by drain.bundle_max with a default of five (dcn-2611201429107457). One branch, one review pair, one preflight, one pull request with a Resolves line per issue; an issue that fails its own fix round is handed back without failing the rest. Measure it against these figures: pull requests, preflights and CI minutes per resolved issue over the next forty."
refines: [iss-2610161652341096]
---

The drain lands one pull request per issue, and between 19 Nov 01:06Z and 20 Nov 14:04Z that was 45 pull requests from build/run branches, 42 of them merged by 14:21Z, each 0.9 to 1.9 hours from opening to merge (19 of the 42 over 1.5 hours). Most are documentation fixes: 33 of the 45 change exactly one brief or docs file, the issue's record and one Go test that pins the corrected sentence, 3 more pair a brief file and a test with other Go changes, and 7 change docs only; the pinning test (for example internal/core/lifeboat/platform_brief_test.go) makes CI's fail-closed classifier read them as code and run the full check job (42 minutes on #1694) where a docs-only change runs it in 19 (#1698). Each issue also pays its own local make preflight (about 40 minutes), its own review pair and its own merge-queue entry. iss-2610161652341096 asks to bundle issues whose found_at paths overlap; most of these touch different files in the same place (the brief's chapters), so that rule alone would bundle few of them.
