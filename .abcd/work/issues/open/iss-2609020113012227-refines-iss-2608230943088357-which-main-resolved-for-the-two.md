---
schema_version: 1
id: "iss-2609020113012227"
slug: "refines-iss-2608230943088357-which-main-resolved-for-the-two"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/version.md"
---

Refines iss-2608230943088357, which main resolved for the two LOUD shapes (an unknown flag, an unknown command) by naming the stale binary in the refusal. This record carries the third shape, which is SILENT and which that fix cannot reach: the verb exists, is served by the old plugin root, and answers confidently. Observed 2026-09-01 immediately after the v0.7.0 release; the mechanism is that plugin roots are keyed by the commit they were installed from, so a hash-pinned path baked into a skill page is designed to expire, and the old root stays on disk ready to answer. The section that follows is the observation as recorded on the day.

## A third failure shape, and it is silent (observed 2026-09-01, v0.7.0 release)

Both shapes above are LOUD: an unknown flag, an unknown command. Immediately
after v0.7.0 published, the same root cause produced a shape that is not.

`/plugin` updated abcd and created a new plugin root while leaving the old one
in place. The `/abcd:version` page interpolates an absolute, hash-pinned binary
path into its own prose, and the copy loaded in-session still named the OLD
root. Following the page exactly ran the old binary, which answered normally,
exit 0, no error of any kind:

```
version: v0.6.8   vintage: 0e22abfd6739
staleness: stale — behind the checkout tip (976575f9c8b6)
```

on a machine where v0.7.0 had just been published and verified. After
`/reload-plugins` the new root provisioned and reported `v0.7.0` correctly.
Nothing was broken. The answer was wrong because the question named a stale
path, and the surface asked was the one whose entire job is reporting version.

**This bears on the two directions already recorded.** The second one — have the
binary answer an unknown verb by naming its own version and vintage — does not
reach this case: the verb exists, is served, and answers confidently. There is
no unknown-verb path to instrument. The first direction, having the command
pages check before invoking, only helps if the check is against something other
than the hash-pinned path the page itself supplies.

**Three aggravating observations from the same session:**

- The old root held **v0.6.8 while v0.6.9 was the published release**, so the
  plugin had already crossed a release boundary on a stale binary before this
  update. Provisioning skips the fetch when a binary is present, so nothing
  would have corrected it.
- The `staleness` field compared against "the checkout tip (976575f9c8b6)",
  which was THIS checkout's HEAD while it sat 396 commits behind `origin/main`.
  A stale binary judged by a stale tree, reported as fact. The v0.7.0 binary
  reports `staleness: unknown` for the same directory, which is at least honest.
  Worth deciding what `version` should compare against, and saying so in the
  output: "stale relative to what" is currently unstated.
- **Six plugin roots have accumulated** (12 Jul to 1 Sep), four holding an ~18MB
  binary each at v0.6.1, v0.6.1, v0.6.6 and v0.6.8. Nothing prunes them. Their
  persistence is what makes a hash-pinned path in documentation hazardous rather
  than merely ugly: the wrong root is always still on disk, ready to answer.

**A third direction this suggests:** have the skill pages name the binary through
a stable indirection rather than baking a hash-pinned absolute path into prose
that outlives the root it names. The path is the defect's carrier here, not the
binary.

### The mechanism: plugin roots are keyed by the commit they were installed from

Established after the v0.7.0 merge landed, which makes the third direction the
obvious remedy rather than one of three. Each root directory is named for a
commit sha:

```
8f68ffb34558  ->  8f68ffb3  Merge PR #568 from release/cold-reading  (v0.7.0)
9a41f97a63b7  ->  9a41f97a  Merge PR #558 from chore/v0.6.8-follow-ups
2571d62f3ddf  ->  2571d62f  Merge PR #505 from contrib/pr294-isnull-coverage
b06fa80b50a5  ->  b06fa80b  Merge PR #470 from release/0.6.2-cutover
```

Three consequences follow, and they change the shape of the problem:

- **The path is guaranteed to change on every update, by construction.** It is
  not occasionally stale; a new root is minted per install because the name IS
  the commit. Documentation that pins the path is documenting a value designed
  to expire. A stable indirection is therefore the fix, not a mitigation.
- **The roots accumulate because each names a distinct commit.** Nothing
  supersedes anything, so every earlier root stays on disk holding its own
  binary, ready to answer. That is what turns a stale path from a broken link
  into a WRONG ANSWER: the old binary is still there and still runs.
- **A root does not necessarily correspond to a tagged release.** `9a41f97a` is
  a follow-ups merge, not a release merge, so the root tracked whatever commit
  was current at install time. That is consistent with it holding v0.6.8 while
  v0.6.9 was the published release, and it means "which release am I on" cannot
  be answered from the root name either.

Note the window this narrows the defect to: the skill page re-interpolates the
path on `/reload-plugins`, so the pinned path is CORRECT after a reload and
WRONG between the update and the reload. That is precisely the interval in which
someone asks what version they are on, having just updated.

Related, and worth deciding together: in a source checkout of abcd itself the
documented resolution ladder reaches the plugin path FIRST, and that path
exists, so it never falls through to `go run ./cmd/abcd` — the rung written for
exactly that case is unreachable in exactly that case. Same session: driving the
plugin binary against a moved tree made `launch ship` refuse on the surface
guard (loud, correct) and made `changelog --json` return an EMPTY cut against a
repo with 181 shipped records (silent, wrong).
