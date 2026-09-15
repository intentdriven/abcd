---
schema_version: 1
id: "iss-2609012037119454"
slug: "ghsa-865x-5m7q-qm79-cwe-59-ensureledgerdirs-alloc-go-provisi"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/alloc.go"
resolution: "The ledger's ancestors are now judged the way its leaves always were. ledgerAncestors (alloc.go) walks every segment between repoRoot and the ledger's parent: resolveRoots runs it without create, so every verb — List and Status included — refuses a committed symlink at .abcd or .abcd/work with ErrPathUnsafe before touching the store; ensureLedgerDirs runs it with create, provisioning a missing segment one level at a time through safeMkdirLeaf instead of os.MkdirAll, so nothing is ever created through a link. repoRoot is threaded through mutationPreamble, withLedgerLock, reservePath and commitCapture to carry the boundary. An issuesRoot outside repoRoot keeps the leaf-only guards as an operator-typed operand. TestLedgerAncestorSymlinkRefused proves, for both .abcd/work and .abcd symlinked out of tree, that Capture refuses, the target stays empty, and List refuses too; TestPathUnsafeSymlinkedLedger keeps the leaf case green. That first walk stopped at the ledger PARENT, so the read half was only half made: os.ReadDir follows a symlinked directory and the guarded reader's O_NOFOLLOW covers only a record's final component, so a committed symlink at issues/ or at a status directory still let list --json and the bare status render serialize a sibling checkout's records under repo-relative paths. The follow-up commit on this branch closes that half: ledgerAncestors becomes ledgerDirs and judges the ancestors, issuesRoot itself and every issueschema.StatusDirs entry by the one rule, and TestLedgerAncestorSymlinkRefused now covers all four link sites with the target both outside the checkout and inside it, proving capture, list, status and promote all refuse and nothing behind the link is serialized."
impact: fix
---

GHSA-865x-5m7q-qm79 (CWE-59): ensureLedgerDirs (alloc.go) provisions the ledger with os.MkdirAll(filepath.Dir(issuesRoot)), which follows every ancestor, and only then Lstat-checks the leaves through safeMkdirLeaf; resolveRoots (roots.go) derives issuesRoot with filepath.Abs alone and checks no ancestor. A committed mode-120000 symlink at .abcd or .abcd/work therefore redirects the whole store — issues/, the .iss-alloc.lock flock, the three status directories and the record — to the symlink target outside the checkout, while the result still reports the repo-relative path; and because List and Status walk the same issuesRoot, every read serializes an out-of-tree ledger too. The fix must establish that every segment from repoRoot down to Dir(issuesRoot) is a real directory (a per-segment Lstat walk, created one level at a time with os.Mkdir, modelled on memory.memoryDir from the GHSA-72rp fix), placed where every verb passes first so the readers are covered, and keep the leaf guards; a custom IssuesRoot outside the repo is an operator-typed operand and keeps leaf-only behaviour. Test: .abcd/work symlinked out of tree, capture refuses with ErrPathUnsafe and nothing is written in the target.

## Grounds

- pursued: the guard is placed in resolveRoots because every verb passes through it first, which covers the readers in the same change; the per-segment walk mirrors memory.memoryDir so the tree has one shape for this class rather than a second idiom, and the os.Root closure of the remaining lstat-to-mkdir window is recorded separately as the package-wide follow-up it is
