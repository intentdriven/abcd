---
id: dcn-2610040000000007
slug: decided-without-a-question-by-abcd-50
date: 2026-10-04
---
2026-10-04 — Decided without a question by abcd-50 (one defensible answer), on the security review of the home-mode fix (iss-2610032205304585): an existing abcd home keeps the mode it has, because every home-scope writer's contract is that a level which already exists keeps its mode (fsutil.EnsureHomeScope, fsutil.EnsureRealDir) and re-moding a folder on the person's machine is a change the issue did not ask for; the two records an earlier version wrote 0o644 (the path-entry record and the registry's index) are narrowed to 0o600 at their next write instead (the bootstrap re-stamps path-entry at every provision, the registry rewrites its index at every registration), so on an existing 0o755 home the residual is that other accounts can list the folder's names, never read a record abcd keeps there. Not chosen: chmod an existing home to 0o700 when the account owns it.
