---
schema_version: 1
id: "iss-2610020704152920"
slug: "banlist-hooks-path-armed-reports-false-in-the-bare-ahoy"
severity: "minor"
category: "bug"
source: "managed-repo"
found_during: "peer report: ahoy install --adopt on a private consumer repo (abcd v0.9.0), reproduced at 7fb52a6b5"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/banlist_scaffold.go"
remedy: "Judge armed by what git will run, not by the local key alone: when no local core.hooksPath is set but a global or system one is, report a third state (a foreign hooks path, armed unknown or chained) and stop advising the local override; optionally probe whether the effective pre-commit reaches .githooks/pre-commit. Test: a fixture with a global chaining dispatcher reads not-unarmed and the status prints no 'arm this clone' advice."
resolution: "Banlist health judges the hooks path by what git runs: with no local core.hooksPath it reads the global/system value as git does, reporting hooks_path armed, unarmed or foreign; a foreign hooks path (a global dispatcher) is no longer read as unarmed and the status prints no 'arm this clone' advice for it (TestAhoyStatusNeverAdvisesOverridingAGlobalDispatcher, TestHooksPathJudgesWhatGitWillRun)."
impact: fix
resolved_by:
  commit: "e9f25a54ea8d21a33d842be4f3cf9a3f53dc32d0"
---

banlist.hooks_path_armed reports false, in the bare ahoy status and in --json, when the committed .githooks/ hooks are reached through a global core.hooksPath dispatcher that chains to them: a false negative. hooksPathArmed (internal/core/ahoy/banlist_scaffold.go) reads only git config --local core.hooksPath and compares it to the clone's .githooks, so it cannot see a global hooks path, and it never asks whether the hook actually runs. Reproduced at tip with a scratch global gitconfig whose core.hooksPath names a dispatcher that runs <toplevel>/.githooks/pre-commit and then .git/hooks/pre-commit: 'git hook run pre-commit' ran abcd's name guard and exited 0, yet hooks_path_armed was false and the bare status said '(arm this clone: git config core.hooksPath .githooks)'. Following that advice sets a local core.hooksPath that overrides the person's global dispatcher in this clone, dropping whatever else it chains, so the false negative turns into harmful advice. Related: open iss-2609120447480903 proposes raising a gap on hooks_path_armed=false, which would make this a false gap.

## Grounds

- pursued: a clone whose global core.hooksPath dispatcher chains to .githooks reads hooks_path=foreign and gets no local-override advice, while an unset clone still gets it; a fixture with such a global dispatcher printing 'arm this clone', or hooks_path reading unarmed, would show it wrong.
