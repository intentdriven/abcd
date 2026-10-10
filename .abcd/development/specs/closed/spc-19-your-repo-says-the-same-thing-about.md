---
id: spc-19
slug: your-repo-says-the-same-thing-about
intent: itd-102
---
# your-repo-says-the-same-thing-about-itself-everywhere-becaus

## Summary

spc-19 delivers itd-102's repo-identity system: one canonical, parseable
markdown identity block (title, tagline, pitch), an onboarding interview that
writes it, and a deterministic positioning check that holds every registered
surface to it — highlighting the exact drifted line, warn-tier by default,
and never rewriting anything autonomously. The design decisions below were
settled by the 2026-07-27 grill; this spec records them together with the
mechanism — it does not reopen them.

## Settled constraints (from the grill)

- **The identity home is a parseable markdown block, not a structured file.**
  Markdown stays the single source of truth; the repo's committed abcd
  configuration records only where the block lives.
- **The check is warn-tier by default** — highlight, never gate — and
  per-repo upgradeable to blocker. **Autonomous rewriting is permanently
  out**: re-rendering from the block is always a proposed diff a human
  adopts.

## Mechanism

### Identity block

The block is three fixed bullets under a recognisable heading, exactly the
shape the abcd-cli brief already carries (`.abcd/development/brief/01-product/README.md`, "Identity (canonical)"):

- `- **Title:** …` (required)
- `- **Tagline:** …` (required)
- `- **Pitch:** …` (optional at onboarding; title and tagline are required)

The parser accepts a multi-line pitch bullet. The block's location — file
path plus heading — is recorded in the repo's committed abcd configuration
alongside the other check configuration; abcd-cli's own configuration points
at the existing brief section unchanged (AC 4).

### Onboarding interview

The prepare flow asks once: title, tagline, pitch (pitch may be skipped).
The host conducts the interview (the prepare skill carries the wording); the
binary scaffolds the block into the recorded location and the pointer into
the configuration. A repo that already has a block is never re-interviewed —
onboarding detects and registers it instead.

### Positioning check

A deterministic check, run by the audit verb, compares each registered
surface against the block:

- **Default registered surfaces (three):** the README strapline, the
  plugin/package manifest description field, and the conventions file's
  (AGENTS.md) opening line. Further surfaces are registrable per-repo in the
  same configuration; none are registered silently.
- A diverging surface yields a warn-tier finding naming the file and the
  exact drifted line, and quoting the canonical line it should carry.
  Per-repo configuration may upgrade the family to blocker.
- **Acceptance corpus:** iss-143's three-variant tagline drift — the
  recorded real instance this check exists to catch. The check's fixtures
  reproduce that drift shape and must all be caught.

### Re-render as a proposal

`abcd identity render` (verb name final at implementation; the behaviour is
the contract) prints the proposed corrected surface content as a diff for
the human to adopt. It writes nothing. A deliberate identity change is an
edit to the block itself, after which the same proposal flow chases the
surfaces.

### Wiring

Both planes at delivery: the audit verb's check and the identity verbs reach
the CLI and the plugin markdown surface; the prepare skill gains the
interview step. abcd-cli's own audit runs the check against its brief block
and passes clean (or its findings are real drift, fixed by re-render or
recorded change — never by weakening the check).

## Acceptance-criteria mapping

- AC 1 (interview lands a parseable fixed-shape block; config records the
  location; markdown is the source of truth) → Identity block + Onboarding
  interview.
- AC 2 (divergence yields a warn finding naming the exact line; per-repo
  upgrade to blocker) → Positioning check.
- AC 3 (never rewrites autonomously; re-render is a proposed diff) →
  Re-render as a proposal.
- AC 4 (abcd-cli points at the existing brief Identity section unchanged;
  iss-143 is the acceptance corpus) → Identity block + Positioning check.

## Out of scope

- Registering surfaces beyond the canonical three by default (per-repo
  addition exists; defaults stay at three).
- Any autonomous write to a rendered surface, under any flag.
- Interview wording beyond the prepare skill's text (host-side, adjustable
  without a binary change).
