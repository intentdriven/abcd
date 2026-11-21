---
id: dcn-2611202018390575
slug: inside-any-working-tree-means-the-trees
date: 2026-11-20
---
2026-11-20 — 'Inside any working tree' means the trees the caller names to the judge, with no upward search (itd-2611041815429986). Step 2 passes the repository the act runs from, its main checkout and every linked worktree, read as files from .git/worktrees/*/gitdir without running git, and the abcd worktree store root ~/.abcd.noindex/worktrees/; steps 3 and 4 judge the registered path again against each listing repository's own trees, so every repository that ever lists the tool holds it to its own. Grounds: adr-2611041815445203 rule 3 extracts the runner's admit rule, which refuses only the trees it is handed, and the spec adds no search; the needs-step1 Opus review found an upward .git search would refuse all of Homebrew on Apple Silicon, where /opt/homebrew is a git checkout, and asking git which files are tracked would run git inside a foreign repository whose config can execute code, breaking rule 1's read, never run. The rule holds only with each link step held at its real location, which this fix round adds. Refs itd-2611041815429986.
