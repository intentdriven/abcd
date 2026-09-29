---
id: adr-38
slug: implicit-checks-are-disk-only
status: accepted
date: 2026-08-15
supersedes: null
superseded_by: null
related_intents: [itd-111, itd-105, itd-108]
related_rfcs: []
related_adrs: [adr-22, adr-23]
---

# ADR-38: Implicit checks are disk-only — the network answers only an explicit ask

## Context

itd-111 (staleness detection) declared a "network trichotomy" as one of its
design decisions: implicit checks read disk, the network answers only an
explicit `--check`, and provisioning fetches only pinned artifacts. The
itd-84 decomposition at that intent's planning interview (2026-08-15) found
the rule is not a feature decision at all — "no version-discovery request
exists anywhere in abcd" binds the whole system, and it would bind it even if
itd-111 were never built. A trust rule wearing a design-decision coat is the
monolith failure itd-84 exists to catch, so the rule was extracted here.

The posture it protects is already load-bearing: abcd positions itself as a
tool whose record stays on the machine, and this repo's own SOTA notes score
competing tools down for ambient network traffic. The update-notifier /
Homebrew pattern — a background version-discovery request on ordinary use —
fails that bar even when its UX grammar (cached comparison, gentle nudge,
one-command fix) is worth keeping. The pattern's own ecosystems offer no
disk-only mode, only full opt-out (verified at the itd-111 fit-challenge,
2026-08-15).

## Decision

Three tiers, exhaustive — every network touch in abcd falls into exactly one:

1. **Implicit operations never touch the network.** Anything abcd does
   without the user naming a network action — session-start checks, status
   renders, audits, ordinary verbs — reads only what is on disk (embedded
   build info, checkout state, plugin-cache manifests, committed config). No
   version-discovery request exists anywhere in abcd.
2. **The network answers only an explicit ask.** A fetch happens when the
   user invokes a verb or flag whose documented meaning is that fetch —
   `abcd version --check`, `abcd docs cite refresh`. The output names its
   source.
3. **Provisioning completes a chosen update, it never discovers one.** When
   a plugin update names a new pinned binary version, bootstrap fetches
   exactly that checksum-verified artifact from the named release — the
   user's update action was the ask; the pin and checksum bound what may
   arrive.

> **Spelling note (2026-09-30).** Tier 2's first example is spelt
> `abcd update --check`: `abcd version --check` moved there
> (itd-2609212130136102) and the old spelling is removed in v0.12.0 (BT1,
> iss-2609251324599468), so it is an unknown command. The rule is unchanged.

## Consequences

- The zero-network test harness the citation gate already uses is the
  enforcement seam: tier-1 paths must pass under it, and a new verb that
  fetches implicitly is a defect, not a preference.
- Update-notification UX is built over disk sources only (itd-111's
  comparison provider), and "check for updates" is forever a verb the user
  runs, not a thing abcd does.
- Any future adapter that wants ambient network behaviour (telemetry, auto
  update discovery) requires superseding this ADR in the open — it cannot
  ride in as an implementation detail.
