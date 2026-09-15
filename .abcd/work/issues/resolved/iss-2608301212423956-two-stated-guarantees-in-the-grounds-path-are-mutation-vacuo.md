---
schema_version: 1
id: "iss-2608301212423956"
slug: "two-stated-guarantees-in-the-grounds-path-are-mutation-vacuo"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "itd-179-round-2-ruthless"
found_at: "internal/core/intent/grounds.go"
resolution: "both guards now have a test that goes red when the guard is short-circuited: a fresh-issue wontfix asserting ErrGroundsRefused on a non-declined token, and an unclosed-fence record asserting the read-back refusal and a byte-identical file"
impact: internal
resolved_by:
  intent: "itd-179"
---

two stated guarantees in the grounds path are mutation-vacuous: the wontfix declined-token refusal and the read-back count check

Found by the round-2 adversarial ruthless review of build/itd-179, both proved
by mutation rather than argued.

Site 1 — `internal/core/capture/grounds.go:77`, the `declined`-only token
refusal on wontfix. Replacing `if g.Token != grounds.Declined {` with
`if false && g.Token != grounds.Declined {` leaves
`go test ./internal/core/capture/` and `go test ./internal/surface/cli/ -run
"Grounds|Capture"` GREEN. The apparent coverage is the last block of
`TestWontfixGroundsOverride`, which calls Wontfix on an issue the preceding
call already moved to `wontfix/`, so `transition` returns
`ErrTransitionConflict` and the `err == nil` assertion fails whether or not the
token is checked. `TestCaptureMalformedGroundsExit2`'s wontfix row only uses
operands `requireGrounds` rejects before the token check is reached.
Remedy: a fresh open issue and `errors.Is(err, ErrGroundsRefused)` on a
`pursued:` operand whose text clears the floor.

Site 2 — `internal/core/intent/grounds.go:144`, the read-back count check that
commit 6658ddb2's own message calls "the durable half". Short-circuiting the
comparison to `false && got != want` leaves `go test ./internal/core/intent/`
and `go test ./internal/surface/cli/` GREEN. It is NOT dead code: on a record
whose body ends inside an unclosed fence the appended `## Grounds` section is
masked and this check is the only thing that refuses. Confirmed against the
real binary: "the appended grounds entry does not read back (0 entries after
the append, expected 1); nothing written", exit 2. With the check disabled that
write lands, the section is invisible to the gate forever, and the CLI prints
"recorded grounds on ... (0 entries)" and exits 0.
Remedy: the fixture the comment already describes — a record ending in an
unclosed fence — asserting the refusal and the byte-identical record.

The class, and why it is one record: a guard whose test passes for an
unrelated reason is a guard that reports success while doing nothing. The
builder's own note earlier in this cycle flagged exactly this as "the class to
watch" after it happened twice. Every guard this branch adds should be proved
by mutation before it is called covered.

## Grounds

- pursued: we expect a test that stays green when its guard is deleted to be measuring something other than the guard, and mutating each of these two turned both green tests red only after they were rewritten
