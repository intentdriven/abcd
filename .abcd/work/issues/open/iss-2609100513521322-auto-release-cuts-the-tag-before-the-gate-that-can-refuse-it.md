---
schema_version: 1
id: "iss-2609100513521322"
slug: "auto-release-cuts-the-tag-before-the-gate-that-can-refuse-it"
severity: "major"
category: "process"
source: "agent-finding"
found_during: "v0.8.0 release, runs 34403815697"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/auto-release.yml"
deferred_after: "v0.7.1"
deferral_reason: "Found while cutting v0.8.0, and the remedy changes the release pipeline itself, which is not a thing to reorder underneath a release that is mid-flight. The workaround is recorded and was exercised end to end, so the next cut is not blocked on this. The waiver lapses at v0.8.0 and the finding returns to the gate, which is the right moment: the reorder wants to be the first change of a cycle, proven by a release, not the last change of one."
---

`auto-release` cuts the version tag **before** the gate that could refuse the
release. A red gate therefore leaves a tag that names a version with no Release
and no binaries: present in git, dated in the CHANGELOG, impossible to install.

This happened at v0.8.0 and cost a tag deletion to recover.

## Evidence

Job order in `.github/workflows/auto-release.yml`:

```
detect  ->  tag  ->  release / verify  ->  rehearsal  ->  release  ->  site
```

In run 34403815697, `tag` completed in **7s** and `release / verify` failed at
**11m**, on `TestHooksRunEveryShapeAhoyInstalls/dev_shim`
(`iss-2609092116378524`, since resolved). Three attempts, three failures, and
after each one the repository held a `v0.8.0` tag with no Release.

A gate placed after the act it guards cannot prevent the state it exists to
prevent. It can only report it.

## The recovery path is what turns a failure into a wedge

`auto-release` already handles "tag exists, Release missing". Its own header:

> If the tag exists but its Release is MISSING (e.g. a transient publish
> failure), `detect` sets need_release=true and re-invokes `release` ALONE —
> built from the tagged commit (release_ref), never the moved-on HEAD — so a
> flaky publish never permanently wedges the version.

Correct for a flaky publish. Wrong for a failing gate, and `detect` cannot tell
them apart, because both present as tag-without-Release. When the cause is a
defect:

1. The defect is fixed and merged to `main`.
2. The merge pushes to `main`, so `auto-release` fires.
3. `detect` finds tag present, Release missing, sets `need_release=true`, and
   rebuilds **from the tagged commit** — which does not contain the fix.
4. It fails again. **Landing the fix cannot rescue the release.**

The promise that it "never permanently wedges the version" holds only in the
transient case. In the non-transient case this path *is* the wedge.

## Workaround, exercised at v0.8.0

Delete the tag **before** the fixing merge lands, never after: `auto-release`
triggers on `push: branches: [main]`, so deleting a tag fires nothing, and
deleting afterwards leaves you needing a further push to re-trigger.

```
fix PR queued
  -> delete the tag while the merge queue is still validating
  -> the queue lands the merge -> push to main -> detect finds NO tag
  -> need_tag=true -> tags the NEW tip, which carries the fix
```

Record the target first so the deletion is reversible
(`gh api repos/<owner>/<repo>/git/refs/tags/vX.Y.Z --jq .object.sha`; v0.8.0 was
annotated object `9fecedae` on commit `9b3fbfab`).

Deleting a published ref is against our own rules and was acceptable here only
because nothing outward-facing had shipped from that tag: no Release, no
binaries, nothing fetchable, and it was under an hour old. **That does not
generalise.** Once a Release exists the route is closed and the only honest
option is a new version.

## Remedies, most fundamental first

1. **Reorder to `detect -> verify -> tag -> build -> publish`.** A red gate then
   leaves no tag and nothing to clean up: the release is simply not cut. This
   removes the wedge rather than making it recoverable. One property must be
   preserved deliberately: `release.yml` checks out `github.sha` so the gate
   exercises exactly the commit whose binaries ship, and with no tag yet to
   resolve, that sha has to be pinned through from `detect` to every later job.
2. **Teach `detect` the difference between a flaky publish and a failing gate.**
   If the previous `release / verify` for this tag concluded in failure, refuse
   loudly and say the tag must be re-cut rather than silently rebuilding the same
   commit. Worth doing regardless of (1), and it is what loud staging asks for: a
   recovery path that cannot work must say so instead of repeating.
3. **A `workflow_dispatch` `retag` input** that re-cuts the tag at HEAD once
   verify passes, making the recovery a reviewable operation rather than hand-run
   API calls.
4. **Bound the retry**, so one wedged tag cannot re-run the same failing build on
   every later push to main.

## Same shape, second instance this release

The `attribution` gate is a required check whose commit half exempts only merge
MESSAGES. A plain `git revert` ninety commits back carried no `Assisted-by:`
trailer, because that is what `git revert` writes by default, and the branch
could not merge until that one message was corrected: a history rewrite of 206
commits and a re-point of two semantic-gate receipts.

A gate that can only be satisfied by rewriting history is also running too late.
A commit template or a `prepare-commit-msg` hook would put the trailer in when
the message is written. Recorded here rather than separately because the
generalisation is the same one, and splitting them would lose it.

## Acceptance

- **Given** a release whose deterministic gate fails, **when** `auto-release`
  runs, **then** no tag exists afterwards and the version stays uncut.
- **Given** a tag whose gate failed, **when** a later push reaches `main`,
  **then** the workflow refuses loudly and names the re-cut as the remedy,
  rather than rebuilding the same commit.
- **Given** a `git revert` on this repository, **when** the message is written,
  **then** it carries an `Assisted-by:` trailer without a later rewrite.
