---
schema_version: 1
id: "iss-2608241347321757"
slug: "resolving-an-issue-is-a-convention"
severity: "major"
category: "architectural-insight"
source: "user-observation"
found_during: "post-merge-provenance-review"
found_at: "internal/core/capture/workflow.go:162,scripts/"
resolution: "Closed by the lint-issues gate (scripts/check-issue-resolution.sh): RS001 refuses a Resolves: iss-N trailer whose record does not leave open/ in the same diff, so resolution lands inside the fix and no post-merge step exists to forget; RS002/RS003 check that a resolved_by.commit stamp names a commit that is actually reachable, which the shape-only --commit validation never did. Wired into make preflight and the CI record-lint job, with a cases script proving each rule can fail. The open question the capture left to a maintainer — whether resolved_by.commit should be written at all or computed on demand — is deliberately NOT settled here: the stamp stays optional and is now merely verified when present."
impact: internal
resolved_by:
  commit: "2852c095"
---
`abcd capture resolve` exists and works. Nothing anywhere can fail when it is
not run, so an issue that has in fact been fixed stays in `open/` until a human
remembers, and a forgotten one leaves no marker to find it by. The measurement:
328 issues in `resolved/`, 78 carrying `resolved_by` — the provenance
convention holds about 24% of the time, and the resolution convention itself
has no denominator, because an unresolved-but-fixed issue is indistinguishable
from an open one.

The specific mechanic that forces the forgetting is `resolved_by.commit`. Every
other field in a resolution is knowable while the record is being edited; the
fixing commit's sha is not, because the record and the fix are in the same
change. That makes stamping a second step deferred past merge, and a second
step is what gets dropped.

Two working patterns exist in the tree, neither enforced:

- The bughunt routine resolves in the SAME pull request, citing the preceding
  commit on its own branch (b783352f cites 52988d9). This works and is the
  right shape: no post-merge step exists to forget.
- The `--commit` flag is shape-checked only, `^[0-9a-f]{7,64}$`
  (`internal/surface/cli`), so a stamp naming a commit that no longer exists —
  or never did — is accepted and reads exactly like a good one.

Audited 2026-08-24: all 76 stamped shas currently in the ledger ARE reachable
from `origin/main`, so the practice is correct wherever it has been applied.
The exposure is latent rather than realised, and it is not what the standing
practice note assumes. That note reasons that branch commits land unchanged
because the repository merges with merge commits; the repository in fact
allows all three methods (`merge_commit=true squash=true rebase=true`), the
method is a per-pull-request choice, and under squash or rebase the cited
branch sha is rewritten out of existence with nothing detecting it.

The deeper observation is that `resolved_by.commit` is denormalised: the
provenance is already in git and is derivable under every merge method,
because it is read from `main`'s own history rather than from a branch.
Verified on iss-97, which stores no `resolved_by` at all:

```
git log --diff-filter=A -- .abcd/work/issues/resolved/iss-97-*.md
  -> 51566371 fix: guard ahoy.Detect marker/config reads against FIFO and
     huge files (iss-97), merged by 94fac3ef (#105)
```

Storing the sha is what makes the step unschedulable; deriving it removes the
ordering problem rather than automating around it.

Direction, not a decision: a `Resolves: iss-N` commit trailer plus a gate in
the `scripts/check-*.sh` family — the shape `check-attribution.sh` already
uses to walk `base..head` — asserting that a declared trailer is accompanied
by the ledger move in the same diff, and that any `resolved_by.commit` present
names a commit reachable from `main`. That family runs in `make preflight` and
in CI, so it fails at push time rather than after a merge. The routing, the
trailer spelling, and whether `resolved_by.commit` should be written at all or
computed on demand are a maintainer's call.

Supersedes the live half of iss-245, which observed that the schema modelled
provenance the verbs could not write. Both halves of that observation have
since shipped: `capture resolve` takes `--intent/--spec/--commit`, and
`capture promote` mints and stamps `promoted_to` (3 issues now carry it). What
remains is one layer down — the verbs exist, and nothing compels or verifies
their use.
