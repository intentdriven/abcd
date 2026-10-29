---
schema_version: 1
id: "iss-2610262035302318"
slug: "the-docs-chapter-s-lint-walks-its-roots"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 577 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/10-docs.md:202"
remedy: "In 06-lint.md:48 replace 'read git-tracked bytes' with: the gates read the working tree under each root, pruning git-ignored untracked paths (internal/core/lint), so an untracked unignored file is read and a tree outside git is read whole."
finding_ref: cfk-dd8a1ddcc3dbfb87
---

The docs chapter's lint walks its roots and prunes only git-ignored untracked paths; the lint contract says the tree gates read git-tracked bytes.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/10-docs.md:202` — "the lint asks git once per root which untracked paths it ignores and reads none of them"
- End B: `.abcd/development/brief/05-internals/06-lint.md:48` — "The two tree gates read git-tracked bytes through `internal/core/lint`"

10-docs says the walk asks git which untracked paths it ignores and skips only those. A committed file is never pruned, and outside a repository nothing is pruned, so untracked files that are not ignored are read, and so is a tree with no git at all. 06-lint says the two tree gates read git-tracked bytes. If only tracked bytes were read, there would be no ignored untracked path to prune and nothing to read outside a repository. The two describe different input sets.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 577.
