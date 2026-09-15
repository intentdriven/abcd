---
schema_version: 1
id: "iss-2609012039117381"
slug: "hook-shim-path-rung-accepts-abcd-inside-working-tree-or-world-writable-dir"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "hooks/hooks.json"
resolution: "The hook shims' PATH rung now accepts an abcd only when command -v yields an absolute path whose directory — resolved physically — is neither under the shim's working directory nor world-writable; a resolution of any other shape is ignored with one line naming the binary and the reason, and the shim takes its existing loud degraded path (PreToolUse still says UNGUARDED and exits 1). All four events carry the same rung; SessionStart has none and is untouched. TestBinaryHooksRefuseAPathBinaryInsideTheWorkingTree, TestBinaryHooksRefuseARelativePathEntry and TestBinaryHooksRefuseAWorldWritablePathBinary pin the three shapes across all four events, each watched executing the planted stub before the change, and TestBinaryHooksFallBackToAPathBinary still pins the documented rescue through an ordinary directory. Two defects of the first cut, found by the second security review and fixed before merge: the rung resolved ls through the PATH it was auditing, so a working tree that controls a PATH element (a shipped vendor/bin/ls beside no abcd) ran code on every prompt where main ran none; the shim now invokes /bin/ls. And the path printed for an ignored binary was stripped of ESC and DEL but not newline or tab, so a directory name could forge a second stderr line; the sanitiser range is now the full C0 set plus DEL, as termsafe.Sanitize masks. TestBinaryHooksNeverRunAnLsFromThePathTheyAudit and TestBinaryHooksSanitiseANewlineInAnIgnoredPath pin both."
impact: fix
---

Sub-finding of GHSA-gx3m-3224-qqcv that does not need the design decision: the hook shims' PATH rung accepts an `abcd` that `command -v` resolves from inside the working tree or from a world-writable directory. With a dot or empty PATH entry — or a PATH entry naming a directory under the checkout — `command -v abcd` in `hooks/hooks.json` (UserPromptSubmit, PreToolUse, PreCompact, SessionEnd) returns a file the checkout controls, so a hostile clone carrying an executable `abcd` becomes the guard and rules loader for a session whose plugin-root binary is missing; a world-writable PATH directory hands the same role to any local user. The documented contract (docs/how-to/install.md: plugin root first, then an abcd on PATH, the one-liner installs into `~/.local/bin`) never places the binary in either location, so refusing both keeps the documented rescue intact. The fix must establish, for all four events, that a PATH abcd whose directory is under the shim's working directory or is world-writable is not executed, that the shim degrades to its existing loud line (PreToolUse: UNGUARDED, exit 1) plus one line saying the PATH binary was ignored and why, and that an abcd in an ordinary PATH directory still serves as the fallback. SessionStart is unchanged (no PATH rung). The full owned-only rung stays with the parent record.

## Grounds

- pursued: the documented install never puts the binary in a relative location, inside a checkout, or in a world-writable directory, so refusing those three shapes narrows the rung without touching the contract; requiring every PATH binary to be vouched for by ~/.abcd/path-entry would break the documented one-liner rescue and stays open on the parent record GHSA-gx3m-3224-qqcv (iss-2609012039107700)

## Correction, 2026-09-09

Two claims in this record were true when it was written and are false at this
tip. Both concern the parent record's rung, not this record's own fix, which
still stands exactly as the resolution describes it.

The resolution's closing clause — "`TestBinaryHooksFallBackToAPathBinary` still
pins the documented rescue through an ordinary directory" — names a test that no
longer exists. `c637a734` ("fix: the hook shims run only the abcd this machine
recorded") deleted it and put three tests in its place, which are what pins the
rung now: `TestBinaryHooksRunAnOwnedPathBinary` (the documented rescue, through
a directory `~/.abcd/path-entry` names), `TestBinaryHooksRefuseAnUnrecordedPathBinary`
(the advisory's own reproduction) and `TestBinaryHooksRefuseAPathBinaryTheRecordDoesNotName`,
all in `internal/surface/cli/hooks_selfprovision_test.go`. The three shapes this
record closed — `TestBinaryHooksRefuseAPathBinaryInsideTheWorkingTree`,
`TestBinaryHooksRefuseARelativePathEntry` and
`TestBinaryHooksRefuseAWorldWritablePathBinary` — are untouched by that commit
and still pin what this record fixed.

The `## Grounds` bullet says requiring `~/.abcd/path-entry` to vouch for a PATH
binary "would break the documented one-liner rescue and stays open on the parent
record". Neither half holds. The parent record, iss-2609012039107700, was
resolved by that same commit under its option A, and the rescue was not broken
but rewritten: the install one-liners in `README.md` and both forms in
`docs/how-to/install.md` now write `path=` and `binary_sha256=` into
`~/.abcd/path-entry`, so a binary the one-liner installs is a binary the record
names. The cost of that is stated on the parent, which is why it is `breaking`
rather than `fix`: a PATH copy installed by an earlier one-liner carries no
record and stops being accepted until the one-liner is re-run.

Nothing here changes this record's disposition. Its own narrowing — absolute
resolution, outside the shim's working directory, not world-writable — is the
first half of the rung the parent's ownership check now sits behind.
