---
schema_version: 1
id: "iss-2609260948440803"
slug: "local-tier-writes-by-path-memory-lint-writes-its-run-log"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review3-history item 6"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/memory/lint.go"
---

Local-tier writes by path: memory lint writes its run-log report by path into the local tier and follows a symlinked ancestor out of the checkout. Lint (internal/core/memory/lint.go, lintReportDir and the write after it) joins .abcd/.work.local/logs/memory/lint-<ts> onto the repo root, os.MkdirAll-s it, and writes report.json and report.md with fsutil.WriteFileAtomic by path. Nothing vets .abcd/.work.local or logs/ first, and a committed symlink beats .gitignore (git add -f), so a checkout that ships .abcd/.work.local as a symlink gets the directory chain and both reports created at the link's target: probed at 35d5cf5f with .abcd/.work.local linked to a directory outside the repo, Lint returned nil and logs/memory/lint-<ts>/report.json and report.md were written in the outside directory. The contained pattern for this same tier already exists: mode.SetAt (internal/core/mode/store.go) opens an os.Root on the checkout, root.Lstat-refuses a .abcd/.work.local that is not a real directory, and writes with fsutil.WriteFileAtomicInRoot. Two sites the review named alongside were checked at 35d5cf5f and are NOT in this class: intent/audit.go's review request and dead-letter writes vet .abcd/.work.local/reviews level by level with fsutil.EnsureRealDirAll, and history/location.go's moveFile writes under a chain Resolve proved real with fsutil.EnsureRealDir, so both refuse a symlinked ancestor; each keeps only a vet-by-path-then-write-by-path swap window.
