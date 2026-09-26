---
id: spc-34
slug: managed-repo-identity-gate
intent: itd-131
---
# The managed-repo identity gate — detect the wrong author/committer, establish only on confirm

## Summary

spc-34 delivers itd-131: extend `ahoy`'s identity check from author-only to the
**effective committer** as well, and add a **propose-and-confirm establish**
path that never silently rewrites git config. Detection lives in
`internal/core/identity`; the gap and the guided fix live in
`internal/core/ahoy`; the CLI renders and prompts (adr-23). Scope is exactly the
delta the reviews confirmed against the code — the author-env case is already
handled by `identity.EffectiveIdentity` and is not rebuilt.

## Scope

- **`internal/core/identity`:**
  - `EffectiveCommitter(root)` — resolve the committer identity env-first
    (`GIT_COMMITTER_NAME`/`GIT_COMMITTER_EMAIL`) then committer config, mirroring
    the existing `EffectiveIdentity` for the author. Deliberately NOT `git var`,
    which fabricates a gecos+hostname identity when unset and would collapse the
    `StatusUnset` state the pre-commit hook blocks on (review Finding 2).
  - `Check` gains a committer comparison against the pin, and a distinct
    committer-divergence result (author≠committer, or committer≠pin), preserving
    the existing `StatusUnset`/`StatusMismatch`/`StatusNoPin`/`StatusOK` author
    semantics unchanged.
  - `IsToolIdentity(name, email)` — recognise an AI/tool identity (the harness
    default `Claude <noreply@anthropic.com>`, and the vendor-domain ban
    `scripts/check-attribution.sh` already encodes) so the routine case is
    machine-visible. Shared predicate, not a third copy (one-canonical-primitive).
- **`internal/core/ahoy`:**
  - `detectGitIdentity` emits a new `git_identity.committer` gap (required,
    resolvable) when the committer diverges, alongside the existing author gaps.
  - A propose-and-confirm apply step: at a TTY, propose the pinned identity (else
    the global git identity — disk-only, no `gh`/network per adr-38) and write
    repo-local `user.name`/`user.email` only on confirm, through the existing
    ConfigChange approval seam; never under `--yes`, never privileged.
  - **Non-TTY / routine context: fail-closed** — report the gap, write nothing
    unprompted, never block on an unanswerable prompt.
- **CLI (`internal/surface/cli`):** render the committer gap and the propose
  prompt; a routine-context render names the tool identity and points at the
  runner (iss-2608210932052003) for establishment, not a silent write.

## How each acceptance criterion is satisfied

1. **TTY propose-and-confirm on a repo-local override** — the apply step
   proposes pinned→global and writes only on confirm; test drives a repo-local
   override differing from the pin and asserts no write until confirmation.
2. **Committer divergence detected (the delta)** — `EffectiveCommitter` +
   `Check`'s committer comparison; test sets author≠committer and a
   `GIT_COMMITTER_*` override and watches the author-only check miss it (fail),
   then the new check catch it (pass).
3. **Non-TTY fail-closed** — the apply step detects no TTY and reports without
   writing; test runs the gate non-interactively against a divergence and
   asserts (a) no config write, (b) no block/hang, (c) the gap is reported.
4. **Routine tool-identity detected here, established at the runner** —
   `IsToolIdentity` flags the harness default as a divergence; the test asserts
   detection. Establishment is spc-out (verified against iss-2608210932052003),
   and the render says so rather than writing.
5. **Never silently rewrite, never escalate** — every write path is
   confirm-gated and unprivileged; a zero-write test asserts a declined confirm
   (and any non-TTY run) leaves git config untouched.

## Deliberately out (carried from the intent)

The routine launcher and launch-time identity establishment
(iss-2608210932052003); a `gh`-authenticated proposal fallback (adr-38: no
network in the implicit path); the `check_text` trailer/footer enforcement
(itd-91 owns it — this intent owns `check_ident`, the identity-field half);
the CI-only-attribution-gate and working-tree-vs-commit gate gaps (separate
plumbing captures); history rewrite of already-produced wrong-identity commits.

## Test-first order

`EffectiveCommitter` and the `Check` committer path land under failing tests
first (author≠committer, `GIT_COMMITTER_*`, and the preserved `StatusUnset`
case as a regression guard); then the ahoy gap; then the propose-and-confirm
and non-TTY-fail-closed apply behaviour; then the CLI render. No new
dependency. The `StatusUnset`-preservation test is the explicit guard against
the `git var` regression the review found.
