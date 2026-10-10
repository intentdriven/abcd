---
schema_version: 1
id: "iss-2610100649479892"
slug: "the-public-name-check-was-silently-disarmed-its-roots-still"
severity: "major"
category: "bug"
source: "managed-repo"
found_during: "abcd inbox report rpt-2610071636214174 from a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0)"
origin: researcher-authored
production_mode: hand-written
found_at: "abcd lint, abcd lint docs, docs-lint roots, ahoy conventions-file retire"
remedy: "Make bare abcd lint surface a target's refusal as an error finding that names the target and its refusal, so the aggregate exits 2 instead of reading clean; in particular the docs target runs whenever .abcd/docs-lint.json exists, not only when docs/ does. Have ahoy and ahoy doctor report a docs-lint root that does not resolve as a gap naming the file to edit, which covers CLAUDE.md retired by ahoy or by hand. Dropping CLAUDE.md from the roots when ahoy retires it is a separate decision, since ahoy has never edited an existing .abcd/docs-lint.json."
---

The public name check was silently disarmed: its roots still named a retired CLAUDE.md, and bare abcd lint reported no findings

## What happened

In a managed repository, docs.target moved from both to agents_md during an upgrade install, and the owner removed CLAUDE.md, which then held only abcd's block (the install itself offers this as conventions.retire). The repository's .abcd/docs-lint.json still listed CLAUDE.md in roots.

From then on:

- **abcd lint docs refused,** exit 2: roots entry "CLAUDE.md" does not exist; a configured root that does not resolve silently disarms every per-file rule for that tree.
- **Bare abcd lint** (every target but outbound) reported "findings": [] and exit 0. The docs target's refusal did not appear, so every lint run looked clean.
- **ahoy and ahoy doctor** reported no gaps.

The public banned-name list (third-party game titles) therefore checked nothing for hours, while several brief edits were made and reported "lint clean". Nothing slipped through in this case (after fixing roots: 91 checks over 32 documents, 0 findings), but only by luck.

## Expected

A target that refuses must make the aggregate run fail loudly (abcd's own loud-staging principle: a stage that no-ops must say so). And retiring CLAUDE.md, which abcd itself offers, should not leave a root that disarms the name check.

## How to see it again

In a managed repository whose docs-lint roots include CLAUDE.md, delete CLAUDE.md; run abcd lint --json (no findings, exit 0) and abcd lint docs --json (refusal, exit 2).

Remedy the reporter proposes: Make bare abcd lint surface a target's refusal as an error finding and a non-zero exit; and when CLAUDE.md is retired (by ahoy or by hand, with docs.target moving to agents_md), drop it from the docs-lint roots, or have ahoy report the dangling root as a gap.

Reported by a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0) through the abcd inbox as rpt-2610071636214174, a defect against abcd v0.13.1, surface abcd lint, abcd lint docs, docs-lint roots, ahoy conventions-file retire.

Evidence:

- rpt-2610071636214174 (the report, kept in the inbox)
- rpt-2610071224565746
