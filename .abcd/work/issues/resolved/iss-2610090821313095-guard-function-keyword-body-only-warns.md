---
schema_version: 1
id: "iss-2610090821313095"
slug: "guard-function-keyword-body-only-warns"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "private security report, filed 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
remedy: "Parse `function name { ...; }` (and `function name() {`) where `name()` is parsed and judge the body as command text, a body the tokenizer cannot read being a block; prove it with a guard verdict-table test (watched fail first) that `function f { git push --force origin main; }; f` and its newline form are block / git-push-force, `function f { git status; }; f` stays allow and the POSIX form stays a block; sweep siblings (other compound-command keywords the reserved-word walk does not step over)."
resolution: "The walk to command position steps over the function keyword and its name, as it steps over coproc NAME, so a function NAME { ... } body is judged like the other function forms and a blocker in it blocks; zsh's repeat COUNT, the same shape, is stepped too."
impact: fix
---

`abcd guard` only warns on a bash `function f { ...; }; f` whose body is a blocker, and the PreToolUse hook lets a warning run, so the function executes.

A private security report, fixed in this release; its advisory, with the full text and reproduction, is published with the release.

Evidence (lines at main 7549ca2d5): `function` is not in the reserved-word set the walk steps over before command position (internal/core/guard/match.go:158, `reserved`), so the keyword form falls to unrecognised-launcher while the POSIX form `f() { ...; }` has its body judged. The hook maps a warn to exit 1, which surfaces the message and lets the tool run; only a block (exit 2) stops it (internal/surface/cli/guard.go:455-466).