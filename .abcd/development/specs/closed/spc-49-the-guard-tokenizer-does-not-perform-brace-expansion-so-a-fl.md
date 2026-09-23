---
id: spc-49
slug: the-guard-tokenizer-does-not-perform-brace-expansion-so-a-fl
intent: itd-156
---
# the-guard-tokenizer-does-not-perform-brace-expansion-so-a-fl

## Summary

Closes a Tier-1 blocker miss in the command guard: `git push {--force,} origin
main` expands in bash to byte-identical `--force` argv, yet the guard tokenizer
reads the literal token `{--force,}` and allows it. This spec teaches the
tokenizer to recognise an unquoted brace group and refuse the command
fail-closed (a real `VerdictBlock`, not a fail-open tokenize error), while
leaving quoted brace sequences and `${VAR}` parameter expansions untouched.

## Scope

In:

- `internal/core/guard/tokenize.go` — the `tokenize` switch (function at line 33),
  a new case for a structural unquoted `{`.
- The block-signal path so the refusal reaches `VerdictBlock`: fold a synthetic
  block signal into `Registry.Check` (`guard.go:367–408`), mirroring how
  `expandPayloads` folds signals by severity.
- `internal/core/guard/tokenize_test.go` and `internal/core/guard/guard_test.go`
  — the four AC cases.

Out:

- No general bounded brace-expander (the intent scopes that out as larger than
  this round). The design refuses an unquoted brace *group* rather than fully
  enumerating its expansion — enough to block the mutate-the-flag-token shape.
- No change to the `$'...'` ANSI-C path (already closed, tokenize.go:244–259) or
  any quoting branch.

## Approach

**Why today allows it.** There is no `{`/`}` handling anywhere in the guard. An
unquoted `{--force,}` falls into the default word branch (tokenize.go:287–291),
so `git push {--force,}` tokenizes to `["git","push","{--force,}"]`; the literal
token does not match the `--force` blocker, and the command is a silent
fail-open. Quoted bytes never reach the default branch — the single-quote
(lines 77–88) and double-quote (89–118) branches append into `cur` directly — so
`'{--force,}'` already arrives as an inert literal token.

**The refusal posture must be fail-closed.** The two guard entrypoints treat a
tokenize *error* differently: the `guard check` verb maps it to exit 2
(blocking, `cli/guard.go:106–108`), but the pre-tool-use hook maps it to
`failOpen` — exit 1, non-blocking (`cli/guard.go:233–235`). So returning
`ErrUnparsableCommand` for a brace group would fail *open* on the hook, which the
intent forbids. The design therefore drives a `VerdictBlock`, not an error.

**The mechanism.** Add a new case in the `tokenize` switch, before the default at
line 287, that fires only for a *structural, unquoted* `{`:

1. It is reached only when `{` arrives as a top-level byte — routed through none
   of the quote branches — so quoted `'{--force,}'`/`"{--force,}"` are already
   exempt by construction.
2. `${VAR}` exemption: a `{` immediately preceded by `$` (i.e. `cur` non-empty
   and its last byte is `$`) is parameter expansion, not a brace group — skip the
   new case and let it fall through to the default branch, exactly as today.
3. Brace-group test: scan forward for a matching top-level `}` and check the body
   for a top-level comma or a `..` range. A lone `{` with no comma/range is an
   ordinary literal (bash leaves it unexpanded) and stays in the default branch.
   A body with a comma or range is the fail-closed target.

When a brace group is identified, `tokenize` records a **block signal** on the
segment (a bool/enum on `segment`, struct at tokenize.go:15–18), which
`Registry.Check` (guard.go:367) folds into the verdict at `VerdictBlock`
severity — the same folding `expandPayloads` signals use (guard.go:375). The
result is a genuine block on both entrypoints: exit 2 on the verb, and a blocking
verdict on the hook, not `failOpen`.

## How it satisfies each acceptance criterion

- *`git push {--force,} origin main` is refused fail-closed* — the new case
  identifies the unquoted brace group `{--force,}` (top-level comma), sets the
  block signal, and `Check` returns `VerdictBlock`. Test: assert the verdict is
  Block and the exit code is 2 on the verb and blocking on the hook path
  (`guard_hook_test.go` posture).
- *A quoted `'{--force,}'` is not treated as a brace expression* — the bytes
  arrive via the single-quote branch, never reaching the new case. Test: assert
  no false positive; the command's verdict is unchanged.
- *A `${VAR}` parameter expansion is not mistaken for a brace group* — the
  `$`-preceded exemption (step 2). Test: `echo ${HOME}` and `git ${x}` keep their
  prior verdicts.
- *An ordinary command with no unquoted brace group is unaffected* — the case
  only fires on a top-level `{` with a comma/range body; every other command
  falls through unchanged. Test: a corpus of ordinary commands asserts
  byte-identical verdicts before and after.

## Decisions

Refuse the group rather than expand it. A correct bounded brace-expander (Cartesian
product of alternatives, nested groups, `{a..z}` ranges) is out of scope for this
round; refusing an unquoted brace group fail-closed removes the mutate-the-flag
bypass without that machinery. This mirrors the round-6 redirection fix, which
also refused a token-mutation shape rather than fully modelling the shell. The
block-signal route (over an `ErrUnparsableCommand`) is chosen precisely because
the hook fails *open* on tokenize errors — only a `VerdictBlock` is genuinely
fail-closed there.
