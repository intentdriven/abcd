---
id: dcn-2611071820445224
slug: the-commit-hooks-go-back-to-building
date: 2026-11-07
---
2026-11-07 — The commit hooks go back to building without -trimpath, byte for byte as they were before the shared-cache change. A running abcd drain --drive pins its repository's hook files when it starts and takes no git step in a lane whose hooks differ, so the hook edit stopped a live drive at its second lane, and only the person can restart it. Restoring the hooks' bytes lets that drive go on; the hooks lose only the cache reuse on a worktree's first commit. Hooks stay unchanged on the default branch until the drive ends or the defect it showed is fixed.
