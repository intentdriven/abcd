---
schema_version: 1
id: "iss-2608242043243131"
slug: "the-preflight-gate-list-is-prose-no-test"
severity: "major"
category: "process"
source: "agent-finding"
found_during: "v0.6.5 docs-currency release gate 2026-08-24"
found_at: "Makefile"
resolution: "the preflight gate list is derived from the Makefile recipe and checked against every surface that restates it — the Makefile's own comment block, the install guide, CONTRIBUTING.md, AGENTS.md and its CLAUDE.md mirror. The detector immediately caught two live instances in AGENTS.md, which said three gates in one place and four in another. An adversarial review then found two holes in the detector itself: a skip-when-absent sentinel made it defeatable by exactly the drift it targets, since rewriting a sentence to name two of five gates removes the sentinel with them; and the Makefile subtest read the whole file including the recipe line it derives the expected set from, so it could not fail. Both are closed and both defeats are proven to fail now."
impact: fix
---

the preflight gate list is prose no test derives from the recipe, so it drifts every time the recipe changes. Makefile's preflight target names its prerequisites once; docs/how-to/install.md, CONTRIBUTING.md and the Makefile's own comment each restate that list by hand. The restatements have now drifted twice in two releases: v0.6.4 corrected three surfaces that said three gates after lint-issues made it four (2852c095 updated AGENTS.md alone), and v0.6.5 re-introduced the same defect by adding site-render as a fifth prerequisite and leaving those surfaces saying four. Both times only the host-run docs-currency reviewer noticed, and both times it refused a release cut over it. The class is a derived-value restatement with no deriving test: the same shape as the marketplace slug, which TestInstallGuideDocumentsTheInstallAndUpdatePath solved by reading go.mod and asserting the documented string contains it. The detector is the same move here — parse preflight's prerequisites out of the Makefile and assert each enumerating surface names exactly that set. Acceptance corpus: the two instances above, both of which the test must flag when replayed against their pre-fix trees.