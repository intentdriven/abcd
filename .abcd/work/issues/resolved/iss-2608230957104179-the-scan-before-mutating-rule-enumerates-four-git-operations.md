---
schema_version: 1
id: "iss-2608230957104179"
slug: "the-scan-before-mutating-rule-enumerates-four-git-operations"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "record-review"
found_at: "CLAUDE.md"
details: "AGENTS.md's concurrency rule reads 'Before a commit, branch switch, stash, or rebase in a checkout that might be shared, check for peer sessions... and announce the mutation'. That is a closed list of four git operations. On 2026-08-23 a session rebuilt the PATH binary and swapped the artefact every session on the machine executes as `abcd`, which is none of the four and has a wider blast radius than a branch switch in one checkout. The session announced it anyway, on its own judgement; no rule required it. The gap is the enumeration, not the operator."
suggested_fix: "State the rule by blast radius rather than by operation list: announce before mutating anything a peer session reads or executes, naming the four git operations and the shared build artefacts as examples rather than as the set. Same shape as the narrowed subject set recorded against the proxy-gate class, so weigh the two together."
related_issues: ["iss-2608230847432285", "iss-2608220750029993", "iss-2608230847432286"]
resolution: "AGENTS.md's concurrency rule is restated by blast radius ('Scan before mutating anything a peer reads or runs'), with the git operations and a replaced build artefact outside git as examples rather than the set. The peers surface test cuts the step on its new heading. Planned itd-148 lists this rewrite in its scope; that bullet is now already met."
impact: internal
resolved_by:
  commit: "be5c7ffeb"
---

the scan-before-mutating rule enumerates four git operations and misses shared artefacts outside git

AGENTS.md § Concurrent sessions says:

> **Scan before mutating git state.** Before a commit, branch switch, stash,
> or rebase in a checkout that might be shared, check for peer sessions via
> the harness's session listing, and announce the mutation to any peer found.

Four operations, enumerated. Everything else is outside the rule.

## The mutation that is not on the list

On 2026-08-23 a peer session rebuilt the abcd binary and replaced the artefact
on this machine's PATH. Verified on disk:

- `~/.local/bin/abcd` is a symlink to `<repo>/bin/abcd-darwin-arm64`.
- That path is gitignored (`/bin/`), so it is not git state and no git
  operation touches it.
- The artefact changed under every session on the machine at once, including
  sessions in their own worktrees, which is a wider blast radius than any of
  the four listed operations. A branch switch harms one checkout; this reaches
  all of them.

**The session announced it anyway, and no rule required it.** That is the
evidence the scope is short rather than the assertion that it is: the
convention was applied correctly by someone reasoning past its stated bounds,
which is not a property anyone can rely on.

## Why the rule cannot be read as already covering it

"Mutating git state" is the rule's own framing, and a gitignored build artefact
is definitionally not git state. A reader checking whether the rule applies
gets a clear no. The failure is not ambiguity, it is confident exclusion.

## The shape this shares

This is a subject set narrowed by a list. The rule measures the right property
— does this mutation affect a peer — over a subject set that is four items
long. Anything outside the list is invisible rather than merely unhandled, and
a reader who consults the rule is told the case does not apply. The same shape
appears in prose here and as a Go slice in the exclusion recorded against the
proxy-gate class; the medium differs and the failure does not.

## What the artefact does and does not tell you afterwards

Relevant because a peer arriving later has only the artefact to reason from.
`abcd version` on the swapped binary reports three fields at three different
trust levels:

```
abcd dev
  vintage:   5d864c42bad9
  staleness: stale — behind the checkout tip (ac46c7768823)
```

- **Version is empty.** `make build` stamps `core.Version` only when `VERSION`
  is passed, so a dev build reports `dev` and carries no identity.
- **Vintage is baked in and reliable.** The peer confirmed this by running the
  old binary from the same working directory and getting a different vintage,
  which rules out the reading that it is derived from the surrounding checkout.
- **Staleness is measured against a moving target.** This binary was built from
  `origin/main` roughly an hour before the reading above, and already reports
  stale, because the comparison is to the local checkout tip rather than to
  what the artefact was built from. It decays immediately and says nothing
  about whether the artefact is the one intended.

So the surface is not uniformly untrustworthy, which would be easier. One field
is reliable, one is empty, and one answers a question the reader is not asking.

## A second artefact hazard, recorded because it is cheap to fix

`<repo>/bin/` now holds four cross-compiled binaries at two vintages —
`abcd-darwin-arm64` from 2026-08-23 and the other three from 2026-08-20 — with
nothing in the directory indicating which is which, and exactly one of them on
anyone's PATH. `make build` writes all four; the swap copied one. A later
reader has no way to tell the current artefact from the stale ones without
running each.

Recorded with the home path written as `~`. The absolute form names the operator's account, and `abcd capture` does not run the redaction scanner: the scanner is wired into `launch`, `repolint` and `history` only, so the ledger write path has no PII gate. The first draft of this very record carried the absolute path and every lint gate passed on it.

## Grounds

- pursued: a session asking whether rebuilding the PATH binary needs an announcement now gets a yes from the rule's own words; shown wrong if a mutation a peer executes still reads as outside the rule
