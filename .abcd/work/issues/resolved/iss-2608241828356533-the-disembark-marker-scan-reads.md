---
schema_version: 1
id: "iss-2608241828356533"
slug: "the-disembark-marker-scan-reads"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "v0.6.4 docs-currency release gate 2026-08-24"
found_at: "internal/core/lifeboat/sources_conventions.go"
resolution: "the disembark walk now honours .gitignore by default, and --include-ignored on probe/plan/pack is the explicit opt-in for the salvage case; the wide scan declares itself in the report and in the marker scan's searched line"
impact: fix
---

the disembark marker scan reads untracked and gitignored files, so a packed lifeboat can cite work the user kept out of git. convOpenQuestionsSource.probeLimited grounds evidence/open-questions from ctx.WalkFiles("."), which walks the working tree through an os.Root and skips only walkSkipDirs (.git, .tox, .venv, Pods, __pycache__, build, dist, generated, node_modules, target, vendor, venv). Neither probe.go nor sources_conventions.go reads the index, parses .gitignore, or invokes git at all. Demonstrated on a throwaway repo holding one committed, one never-added and one gitignored file: the probe cited all three by path:line. This repo's own .abcd/.work.local/ — the gitignored local tier holding scratch/ and logs/ — is not in walkSkipDirs, so a lifeboat packed here would carry path:line citations out of it. The blank-case wording is already accurate ('every regular file in the tree except <walkSkipDirs>'); the defect is the behaviour, not that string. Decide whether the scan should consult git, honour .gitignore, or extend walkSkipDirs, and note disembark is offered as read-only over a source repo including dead and archived ones, where untracked residue is likeliest.