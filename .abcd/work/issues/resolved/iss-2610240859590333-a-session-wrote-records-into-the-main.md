---
schema_version: 1
id: "iss-2610240859590333"
slug: "a-session-wrote-records-into-the-main"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "/abcd:capture of the autonomy-grant design"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli"
remedy: "Make every record-writing verb (capture, intent, decide, spec, and the index generator) and the guard's write path refuse in the default branch's working tree of a checkout another live session also holds, naming the machine-scoped worktree store and the git worktree add command to use, with an explicit override for a person working alone."
resolution: "every record writer (capture, intent, decide, spec close and the index generator) now refuses, exit 3, in a main checkout two or more live sessions started in, naming the worktree store; the session hooks keep one occupancy marker per session. A file the host's own edit tools write there is not yet refused: that remainder is iss-2610280051498465"
impact: fix
---

A session wrote records into the main checkout's working tree on main while another live session (abcd-0d) was working in the same checkout, because nothing enforces the rule that a concurrent session works in its own worktree. On 2026-10-24 this session ran capture, intent, decide --adr and go generate from the main checkout: an issue record, an intent draft, an ADR and two regenerated index regions landed in the shared tree, the regenerated indexes folding in a peer's untracked draft. Nothing refused or warned. capture, intent and decide write into whichever checkout they run in; the guard judges shell commands and questions, not where records are written; and the CONCURRENCY rules domain, which carries the rule, recalls only on prompts about worktrees, peers or sessions, so a prompt saying 'file the decision record' never injected it. The agent also skipped the required scan (session listing and abcd peers) before mutating. The rule held only as prose.

Related: iss-2608230847432285 records the same failure arriving by a different route, a session whose shell cwd reverts from its worktree to the shared checkout. Here the session never entered a worktree at all, so the two want one enforcement: a write is judged by the checkout it lands in, whatever route put it there.
