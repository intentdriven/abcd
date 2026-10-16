# CONTEXT

Shared team/agent orientation — what you need to know *right now* to be useful.
Short and pointer-heavy; durable design truth lives in
[`../development/`](../development/), personal session state in
`../.work.local/NEXT.md` (local).

This file is **status-free** (see [dcn-2607100000000005](decisions/dcn-2607100000000005-context-md-goes-status-free-it-keeps.md)): what is being
worked on, what is next, and what has shipped are read live from the surfaces
built for that — the issue ledger (`abcd capture list --open`), the roadmap
dashboard (`../development/roadmap/README.md`), and the intent buckets — never
written here, where they would only go stale. A record-lint rule
(`context_status_free`) enforces this.

A second rule (`context_citation_currency`) watches the sharp-edges list below:
it blocks a bullet that grounds a live constraint in a record which has since
resolved, shipped, closed, or been superseded, so a caveat cannot outlive the
record it rests on.

## What this repo is

`abcd-cli` is the from-scratch **Go** rebuild of abcd as a host-agnostic
configuration layer for development — a single `abcd` binary with a
transport-agnostic core, usable as a Claude Code plugin and in the companion
harness's ecosystem, depending on no external tools. It supersedes the frozen
Python reference implementation.

## Live constraints / sharp edges

- Trust the binary over the record where they disagree. Less of the
  brief↔surface agreement is machine-enforced than the record's five named
  surfaces suggest: `surface_coverage` holds Direction B (every shipped surface
  resolves to a brief row) on every run, and its sub-verb pass also reads the CLI
  command tree (via the committed `surface.json` snapshot) for surface-bearing
  verbs — but the agent and hook surfaces, the operator-internal verbs
  (`spec`/`rules`/`hook`/`completion`), and all of Direction A stay uncovered.
  Direction A — the brief's prose describing what the binary actually DOES — is
  irreducibly semantic. Everything outside that one structural check is carried
  by the periodic release-gate cross-check
  (`../development/release-gate/brief-surface-crosscheck.js`, pinned by its
  manifest and tiered by release impact), which runs at a release, not
  continuously. So a brief sentence about behaviour can be wrong between
  releases.
- Single repo, curated release (no dev→public mirror). `.abcd/**` is in every
  repository checkout — marketplace installs and release source archives
  included — and never in the released binaries. The launch bundler denies the
  namespace structurally, but that filter has not yet run on a cut release.
- Never commit/push without the maintainer asking; new deps need sign-off.
