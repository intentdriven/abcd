---
schema_version: 1
id: "iss-2610161207130167"
slug: "a-deadline-bound-stored-git-read-gitutil"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "spc-2610150728396006 step 3 fix round, item C(a)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/lazyfetch.go"
remedy: "Thread the context through guardObjectRead and lazyFetchGuard so the version probe and the config read run under it, or answer an ended context before either runs."
---

A deadline-bound stored git read (gitutil.RunCappedBytesAsStoredContext, which the status reader's walk and batch reads use) first runs the object-read guard's git version probe and, on a git older than 2.44, a git config read, both outside the caller's context: with a git that does not answer, a read given 300 ms returned after 20 s. The prompt hook still answers inside its budget, since PromptNotes selects on the deadline, but the reading goroutine outlives it and the function's contract that git is killed when the context ends does not hold for those two commands.
