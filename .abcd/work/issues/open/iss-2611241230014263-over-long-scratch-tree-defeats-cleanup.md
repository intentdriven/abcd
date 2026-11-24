---
schema_version: 1
id: "iss-2611241230014263"
slug: "over-long-scratch-tree-defeats-cleanup"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/defaults/rules.json"
remedy: "Add a SCRATCH rule: a session or test that builds a tree past PATH_MAX or with hostile names removes it before it ends with a descriptor-relative delete (Go os.RemoveAll, which uses unlinkat; t.TempDir in tests) and proves it gone by a stat, never by rm -rf's exit status"
---

A session that built a directory tree deeper than macOS's 1,024-byte PATH_MAX in its scratchpad (a lifeboat-worktree session in the dev account, 250 or more levels of 60-character names under scratchpad/deep/src, beside hostile1) left it behind. Nothing removed it by the usual route: rm -rf, as root, exits 0 and prints nothing (fts_read: File name too long, hidden by -f); find stops the same way; the boot-time dirs_cleaner moved the old /private/tmp into /private/var/dirs_cleaner and could not delete it either. Time Machine then reported it as 'could not be backed up' on every backup to one destination, excluding the parent did not clear the warning, and the person lost a morning to it. Deleting it took a descriptor-relative delete (Python shutil.rmtree under raised open-file, recursion and stack limits). abcd's SCRATCH domain says where working files go, but not that a pathological tree (over-long path, hostile names) is torn down before the session ends, nor that a plain rm -rf can report success on a tree it did not remove.
