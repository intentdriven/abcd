---
id: itd-111
slug: a-stale-abcd-never-answers-silently-every-surface-that-runs
spec_id: spc-22
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-108]
severity: minor
impact: additive
---

# A Stale abcd Never Answers Silently

## Press Release

> **Every abcd surface now knows its own vintage and says so.** The binary
> carries its build revision; at session start the plugin compares it against
> what should be running — the source tip in a dogfood checkout, the
> plugin manifest's pinned version everywhere else — and a mismatch is
> announced with the one-command fix, never discovered a session later
> through misbehaviour. `abcd version` and `abcd ahoy` always show install
> mode and vintage. And the one verb where stale logic writes state refuses
> outright: `ahoy install` run through a binary older than its own source
> declines to touch the machine until the binary is rebuilt. abcd never asks
> the network "what's new?" on its own: implicit checks read only what is on
> disk, the network answers only an explicit `--check`, and when a plugin
> update names a new binary version, provisioning fetches exactly that
> pinned, checksum-verified artifact — completing an update the user already
> chose, not phoning home.
>
> "A month-stale binary spent a morning confidently applying month-stale
> install logic at my machine," said Alice, a maintainer. "Now the session
> opens by telling me the binary is behind the tip, and the install verb
> won't even run through it."

## Why This Matters

The 2026-08-15 install session is the evidence (iss-228): the repo-root
plugin binary predated the no-sudo install work, so it targeted root-owned
`/usr/local/bin`, rejected the `--bin-dir` flag its own skill documentation
described (iss-225/226 — the docs were current, the binary was not), and sent
a whole session into detective work before `make build` revealed the trivial
root cause. Nothing in the system knew — or could say — that the binary was a
month behind the surface fronting it. The failure class is invisible by
construction: a stale binary behaves plausibly, just wrongly, and the newer
the documentation the more misleading the combination. iss-227 (error paths
that report partial with no note) compounds it: silence on top of silence.

Detection is cheap and already half-present — Go stamps the VCS revision into
the binary, the checkout knows its tip, the plugin cache knows its pinned
version. What is missing is only the comparison and the refusal to stay quiet.

## Design Decisions (grilled 2026-08-15)

1. **Separate intent, detection-only.** itd-108 ships a distribution channel;
   this ships a standing invariant across both channels. itd-111 never
   writes and never heals — rebuild is the dev shim's job, re-download is
   itd-105/108's provisioning job. It detects and refuses to be silent.
2. **The network trichotomy is a system-wide trust rule, extracted.** The
   itd-84 decomposition at planning (2026-08-15) routed it out of this intent:
   the rule now lives as
   [adr-38](../../decisions/adrs/0038-implicit-checks-are-disk-only.md) and
   brief invariant 7, which this intent cites and implements — implicit
   checks disk-only; the network answers only an explicit ask; provisioning
   fetches only the manifest-named, checksum-verified version.
3. **Warning surfaces.** SessionStart notice (the existing gap-notice
   channel) names the stale binary, its vintage, and the one-command fix.
   `abcd version` and `abcd ahoy` always print install mode + vintage.
   Ordinary verbs stay silent — with one deliberate exception: `ahoy
   install` run through a stale-against-tip binary refuses with the mismatch
   named and writes nothing, because stale install logic mutating the
   machine is the trap the evidence session actually fell into.
4. **Doc/binary version-match is a corollary, not a mechanism.** The plugin
   ships docs and binary as one unit, so drift exists only where vintage
   drift exists (dev checkouts, failed provisioning); the vintage warning is
   the doc-drift warning. No separate doc-version machinery.
5. **Platform parity is verified, not assumed.** macOS and Linux share the
   same paths and semantics by design; the claim becomes a machine-class
   criterion in the itd-109 calibration set (fresh Linux box on the (b)
   page), not an assumption in prose.
6. **The explicit check lives at `abcd version --check`** (planning ruling,
   2026-08-15). Vintage is `version`'s domain; `ahoy` stays disk-only.
7. **Unknown vintage fails closed at the refusal gate** (planning ruling,
   2026-08-15, from the fit-challenge's stamping-gaps finding). A binary
   whose vintage cannot be determined — unstamped build, `-buildvcs=false`,
   or a dirty (`vcs.modified`) rebuild — is reported as **unknown**, never
   silently treated as fresh; `ahoy install` refuses through it, naming a
   rebuild (or an explicit override flag, specced there) as the out.
8. **Decomposition record (itd-84 hand-run, 2026-08-15).** Verdict SPLIT:
   the network posture → adr-38 + brief invariant 7 (above); the anti-
   wallpaper micro-prompt seed → iss-230. Typed links: `refines` itd-105
   (this intent reports the transition provisioning performs), `refines`
   itd-109 (parity as a machine-class calibration criterion), `refines` the
   iss-206 skew-notice retirement — a scoped replacement, not a reversal:
   steady-state skew machinery stays retired; this intent covers the
   non-steady states (dev checkouts, failed provisioning) pinned
   provisioning cannot reach.

## SOTA

Anchors: the `update-notifier` pattern (npm ecosystem), Homebrew's
auto-update-on-use, and Go's embedded build info (`runtime/debug.BuildInfo`
VCS stamping). **Declared path: 2 — native floor.** The anchors' implicit
network checks fail the fit-challenge outright (a privacy-positioned tool
making background HTTP calls is the posture this record scores against
elsewhere); what survives is their UX grammar — cached comparison, gentle
nudge, one-command fix — implemented over disk-only sources.

**Fit-challenge (independent, 2026-08-15): UPHELD**, with three recorded
caveats. (a) The seam is a **version-source provider interface feeding one
comparator** — `compare(current, expected)` with `expected` drawn from a
provider (disk providers now: embedded revision, checkout tip, manifest pin;
a network/selfupdate provider is a drop-in later, behind the new-dependency
gate). A fixed-arity function over three disk sources would be reuse, not
swappability. (b) The closest disk-only precedent joins the anchors: git's
own "behind upstream" notice — comparison against locally cached refs,
refreshed only by explicit fetch — is exactly the grammar built here.
(c) Go's VCS stamping has known holes (`go run`/`go test` binaries,
`-buildvcs=false`, `.git`-less builds, `go install module@version` history,
`vcs.modified` dirty rebuilds), so the comparator has an explicit
**unknown** outcome — see design decision 7. The challenge also confirmed
no adoptable path-1 candidate exists (every checker in the class is
network-based) and that the pattern's ecosystems offer opt-out, not a
disk-only mode.

## Scope Conditions

None stated.

## Acceptance Criteria

- Given a dogfood checkout whose plugin-root binary predates the source tip,
  when a session starts, then the SessionStart notice names the binary, its
  embedded revision, the tip it trails, and the one-command rebuild fix.
- Given a binary stale against its checkout tip, when `abcd ahoy install`
  runs through it, then the install refuses before any write, naming the
  vintage mismatch; a fresh binary proceeds normally.
- Given any install mode, when `abcd version` or `abcd ahoy` runs, then the
  output includes install mode and vintage (embedded revision or pinned
  version) and staleness relative to the disk references.
- Given any verb without an explicit check flag, when it runs, then no
  network request for version discovery is made — verifiable in the
  zero-network test harness the citation gate already uses.
- Given the explicit check (`abcd version --check`), when the user invokes
  it, then the latest release is fetched once, compared, and reported with
  its source named.
- Given provisioning has fetched a new pinned binary (itd-105/108's job),
  when the next session starts, then the session reports the version
  transition performed.
- Given a binary whose vintage cannot be determined (unstamped build, dirty
  `vcs.modified` rebuild), when staleness is evaluated, then the state is
  reported as unknown — never as fresh — and `abcd ahoy install` run through
  it refuses before any write, naming the rebuild fix.
- Given the same staleness scenarios on macOS and on Linux, when the itd-109
  calibration runs, then observed behaviour matches — recorded as a
  machine-class criterion, human-verified per platform.

## Open Questions

_All resolved or explicitly deferred at planning (2026-08-15):_

- **Sampled re-surfacing / one-tap micro-prompt** — graduated to its own
  capture, iss-230 (facilitator-experience plan); struck from this intent.
- **Explicit check naming** — resolved: `abcd version --check` (design
  decision 6).
- **Harness portability of the SessionStart channel** — **explicit
  deferral** to the itd-22 lineage: the comparison provider is host-agnostic
  core; each harness adapter wires its own session-start channel when that
  harness lands. Not a criterion of this intent.
- **Refusal breadth** — resolved: deliberately narrow, `ahoy install` only
  (design decision 3); extending to other state-writing verbs is a later
  intent if evidence arrives.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
