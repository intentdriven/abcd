---
schema_version: 1
id: "iss-2608261029074763"
slug: "brief-05-internals-06-lint-md-section-3"
severity: "major"
category: "documentation"
source: "user-observation"
found_during: "abcd lint assessment 2026-08-26"
found_at: ".abcd/development/brief/05-internals/06-lint.md"
resolution: "06-lint.md section 3 describes the real gates and marks the staged tiers as design targets, not a phantom CLI"
impact: internal
resolved_by:
  commit: "b760a67c"
---

brief/05-internals/06-lint.md section 3 documents a lint architecture that does not exist, in flat present tense. Verified 2026-08-26 on main. It describes four CI trigger points for a program invoked as 'internal/core/lint --stdin --display-path <canonical> --json --touched-line <N>'; internal/core/lint is a library package with no package main, so that invocation cannot run at all, and cmd/ holds only abcd, abcd-gen-cli-ref, abcd-gen-surface, record-lint and scaffold-sync. Four flags it specifies are absent from the whole codebase: --codes-filter, --promote-check, --touched-line and --display-path each return zero hits across internal/ and cmd/, and --corpus's only three hits are an unrelated ahoy config key and two guard test fixtures. The shipped verb takes exactly two flags, --root and --json. Trigger point 1 asserts a pre-commit hook configured by .pre-commit-config.yaml with pass_filenames: false; no such file exists. The named orchestrator internals -- _classify_and_filter, _parse_hunk_lines, _lint_terminology_files, IntentLinter, _detect_location -- are Python-shaped and appear in no source file outside the record itself. Note points 3 and 4 of the same section DO hedge correctly ('design target for CI', 'not among ci.yml's jobs today'), which is what makes points 1 and 2 read as shipped rather than planned; the fix is to mark them the same way or delete what was never built, not to build it. The exit-grammar inversion at line 174 of the same file is the same defect surfacing at one line, and is already captured separately and more precisely as iss-2608260944055780, which names repolint.go:147 exitCode() as the authority; that issue is not superseded here, it is the narrow instance of this general one. This is also the same shape as iss-2608231000561060, which records the surfaces README claiming abcd lint gates CI when it gates nothing, and the enforcement-claims-are-facts principle names the class.