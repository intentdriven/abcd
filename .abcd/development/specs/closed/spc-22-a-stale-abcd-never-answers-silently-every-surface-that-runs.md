---
id: spc-22
slug: a-stale-abcd-never-answers-silently-every-surface-that-runs
intent: itd-111
---
# a-stale-abcd-never-answers-silently-every-surface-that-runs

## Summary

Staleness detection for the abcd binary, detection-only: every surface knows
its own vintage and says so, and the one verb where stale logic writes state
(`ahoy install`) refuses through a stale or unknown-vintage binary. The
intent implements — never redefines — the network posture extracted at its
planning decomposition:
[adr-38](../../decisions/adrs/0038-implicit-checks-are-disk-only.md) and
brief invariant 7 (implicit checks disk-only; the network answers only an
explicit ask; provisioning fetches only the pinned artifact). Planning
rulings of record (2026-08-15, in the intent's Design Decisions): explicit
check at `abcd version --check`; refusal narrow (`ahoy install` only);
unknown vintage fails closed; harness portability of the session-start
channel explicitly deferred to the itd-22 lineage.

## Scope

- **The comparison seam** (core): a version-source **provider interface**
  feeding one comparator — `compare(current, expected)` — per the upheld
  fit-challenge. Disk providers only in this cut: embedded build revision
  (`runtime/debug.BuildInfo` VCS stamping), dogfood checkout tip, and the
  plugin-cache manifest pin. The comparator's outcomes are fresh / stale /
  **unknown** — unknown is a first-class result (unstamped build,
  `-buildvcs=false`, `.git`-less build, dirty `vcs.modified` rebuild), never
  silently mapped to fresh.
- **Warning surfaces** (adapters over the seam): the SessionStart gap-notice
  channel names a stale binary, its vintage, the reference it trails, and
  the one-command fix; `abcd version` and `abcd ahoy` always print install
  mode + vintage + staleness; ordinary verbs stay silent.
- **The refusal**: `ahoy install` evaluated through a stale-against-tip or
  unknown-vintage binary refuses before any write, naming the mismatch (or
  the unknown state) and the rebuild fix. Whether an explicit override flag
  exists — and its name — is decided in this spec's implementation review,
  not improvised.
- **The explicit check**: `abcd version --check` fetches the latest release
  once, compares through the same comparator via a network provider used
  only here, and reports with the source named.
- **Transition reporting**: after provisioning (itd-105/108's mechanism) has
  fetched a new pinned binary, the next session reports the version
  transition performed.

Out of scope: any healing (rebuild is the dev shim's job, re-download is
provisioning's); any second warning cadence (anti-wallpaper re-surfacing is
iss-230's seed); refusal on verbs other than `ahoy install`; any
version-discovery request (adr-38 tier 1); non-default-harness session-start
wiring (deferred, itd-22 lineage).

## Approach

One core package owns the provider interface and comparator; front doors
consume results, never re-derive them (transport-agnostic core). The
SessionStart notice rides the existing gap-notice channel — no new hook
machinery. The refusal is a precondition check inside the install verb's
existing gate sequence, before the first apply step. The `--check` network
provider is the only tier-2 network touch this intent adds, and it lives
behind the explicit flag per adr-38; every other path must pass the
zero-network test harness.

## Acceptance-criteria satisfaction

- **Session-start notice (AC 1):** the checkout-tip provider supplies
  `expected`; the notice renders vintage, trailed tip, and the rebuild
  command via the gap-notice channel.
- **Refusal (AC 2) and unknown-vintage refusal (AC 7):** the install verb's
  precondition consumes the comparator result; stale and unknown both
  refuse before any write, fresh proceeds. Tests watch both refusals fail
  before the change and pass after, per the TDD gate.
- **Vintage output (AC 3):** `version` and `ahoy` render install mode +
  vintage + staleness from the same comparator — one source of truth, two
  renders.
- **Zero-network invariant (AC 4):** every path except `version --check`
  runs under the zero-network harness the citation gate already uses;
  adr-38 tier 1 is the asserted property.
- **Explicit check (AC 5):** `version --check` fetches once, names its
  source, and reuses the comparator through the network provider.
- **Transition report (AC 6):** the session compares the running binary's
  vintage against the last-reported one recorded beside the plugin-cache
  metadata and reports a transition when they differ; the fetch itself is
  provisioning's, out of this intent.
- **Platform parity (AC 8):** the staleness scenarios enter the itd-109
  calibration set as machine-class criteria, human-verified per platform —
  this spec delivers the scenario list, itd-109 delivers the harness.
