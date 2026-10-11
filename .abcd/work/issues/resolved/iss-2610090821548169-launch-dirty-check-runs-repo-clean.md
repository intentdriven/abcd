---
schema_version: 1
id: "iss-2610090821548169"
slug: "launch-dirty-check-runs-repo-clean"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "private security advisory GHSA-qv3j-fq4f-g5xr, filed 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/gates.go"
remedy: "Before the `diff` subcommand pass `-c filter.<name>.clean=`, `-c filter.<name>.smudge=` and `-c filter.<name>.process=` for every filter name in repo config (an exit-1 `git config --get-regexp` meaning an empty list); prove it with a launch test (watched fail first) that a copied checkout with the clean script leaves the mark empty while a real edit is still listed, a filterless byte-identical tree still reports clean, and no artefact declaration still returns before the diff; sweep siblings (DirtyPayloadFiles, dirtyCorpusPaths and every worktree-versus-commit diff)."
resolution: "The launch dirty check now blanks every repository content filter (clean, smudge, process) before its diff against HEAD, through the new gitutil.FilterOverrides, so a copied checkout's filter program no longer runs and no longer decides what reads as clean; a required filter with its commands blanked makes the check fail closed. TestDirtyTreeFilesRunsNoRepoFilter and its process and required-filter siblings prove it."
impact: fix
---

The launch dirty-tree check (`git diff ... HEAD` in `DirtyTreeFiles`) runs a repository `filter.<name>.clean` program as the operator on a copied checkout, and the gate can still report the tree clean.

Private security advisory GHSA-qv3j-fq4f-g5xr (draft, severity high). Full text, evidence and reproduction: the security-drain-2026-10-09 run directory in the main checkout's local tier. This record stays uncommitted until its fix lands; the fix commit adds it directly to resolved/.

Evidence (lines at main 7549ca2d5): `DirtyTreeFiles` (internal/core/launch/gates.go:920) runs `diff --no-renames --name-only -z HEAD` through `gitutil.Run` (internal/core/launch/gates.go:929), whose `isolatedArgs` pins only hooksPath, fsmonitor and quotePath (internal/gitutil/repo.go:52). A copied checkout loses the index stat, so git re-hashes the worktree through the clean filter. `dirtyTreeGate` reports `clean` on an empty list (internal/core/launch/gates.go:950); `abcd launch --dry-run` reaches it once `.abcd/config/artefact.json` loads (`{"kind":"binary"}` suffices), and the ship ingest reaches it when it stages a payload, `--allow-dirty` still running the diff. Siblings of the same command class: `DirtyPayloadFiles` (internal/core/launch/archive.go:490) and `dirtyCorpusPaths` (internal/core/intent/consistency.go:250). `--no-ext-diff` and `--no-textconv` do not stop a clean filter, and blanking only `.clean` leaves `.process`.

Reproduction: on git 2.39.5, in a repo with `.abcd/config/artefact.json` = `{"kind":"binary"}`, point `filter.evil.clean` at a mode-0755 script that appends a mark and cats stdin, set `.git/info/attributes` to `* filter=evil`, copy the directory, and run `abcd launch --dry-run` in the copy from another cwd. The mark is written and the dirty-tree row says clean. Do not switch to `git status`: on a copied tree `git status --porcelain` also runs the filter.
