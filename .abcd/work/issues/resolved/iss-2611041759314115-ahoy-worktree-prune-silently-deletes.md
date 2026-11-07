---
schema_version: 1
id: "iss-2611041759314115"
slug: "ahoy-worktree-prune-silently-deletes"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "independent review of the shipped fix"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/worktree/notes.go"
remedy: "Leave out only the file abcdrepo.CheckoutBuildRel names, plus the abcd.<pid> file scripts/abcd.sh builds it through, and archive everything else in the tier's bin/; a test prunes a worktree with a script beside the binary and finds the script in the archive."
resolution: "Fixed: the notes archive leaves out only the binary scripts/abcd.sh builds and its <binary>.<pid> build file, and only when the file at that name is not text or is past the per-file cap. Everything else in the tier's bin/ is judged as a note, so a script kept there survives prune, and a file the archive cannot vouch for keeps the worktree; TestPruneArchivesWhatElseTheTiersBinHolds, TestPruneArchivesTextAtTheBuildsNames."
impact: fix
---

ahoy worktree prune silently deletes every file a person keeps in the top level of a worktree's local-tier bin/. The notes archive skips that whole directory (notes.go, buildProductRel with fs.SkipDir), though only the built binary needs leaving out, so a merged, clean worktree whose .abcd/.work.local/bin/ holds a hand-written script is reported reclaimed, the script is not in the archive or its manifest, and removing the worktree deletes the only copy. Reproduced at the default branch tip with bin/run.sh beside NEXT.md: outcome reclaimed, worktree gone, NEXT.md archived, run.sh not archived. The behaviour follows the remedy of iss-2611022247131866, which named bin/ as build products.
