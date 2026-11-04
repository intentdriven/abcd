---
schema_version: 1
id: "iss-2611041419354106"
slug: "rules-hook-output-overruns-the-host-cap"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "usage review of 4 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/inject.go"
remedy: "Hold the prompt hook's whole output under the host's cap: send a domain in full, or as its first rule and a line naming the command that shows the rest, and record only that; keep room for notes; and stop sending in advance what the guard refuses (itd-2611041636196213, delivered in the same change)."
resolution: "The prompt hook's whole output now stays under the host's cap; a domain that does not fit goes short and is recorded only as what was sent."
impact: fix
resolved_by:
  intent: "itd-2611041636196213"
  spec: "spc-2611041649558539"
---

The prompt hook lets rule text run to 64 KiB, but the host cuts any hook output over 10,000 characters down to a saved file and a 2,000 character preview, so most matched rules never reach the agent, and the session ledger records every domain as delivered, so it is not sent again until the fifteen-prompt backstop, which is cut the same way. At tip ad4a370a1 the real hook on a fresh session gives 12,191 characters for an ordinary prompt about a failing test on a branch, 24,834 for a prompt naming rm and git push, and 38,582 for one naming a worktree, a peer session and git push. The bundled SHELL domain alone is 23,095 characters over 45 rules, so it can never fit, and in one observed injection 5 of 41 rules were in the preview, the rm rule starting near character 17,600. Local session logs show 475 such cuts in the seven days to 4 Nov 2026, median 17,178 bytes, largest 53,095. The session-start hook already honours this cap (hook_checklist.go); the prompt hook does not. The comment on injectionBudgetBytes (the bundled set renders to a few KiB) and the brief's figure of about 2k tokens for SHELL are stale: the whole set is now 72,002 characters.
