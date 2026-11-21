# abcd Plug-in — Design Brief

This is the canonical design brief for the abcd plug-in. The brief reflects the project's *current* state — it is not versioned, snapshotted, or archived in this directory. History lives in `git log`; inflection-point rationale lives in [`../decisions/adrs/`](../decisions/adrs) (ADRs); see [adr-5](../decisions/adrs/0005-brief-is-current-state.md) for why.

It is split across numbered folders for concurrent editing, diff legibility, and agent-context-budget friendliness.

> **Delivery-state provenance.** `spc-N` references throughout the brief trace
> the predecessor implementation's spec store, carried as design input for this
> from-scratch Go rebuild. They attribute *which spec owns which behaviour*;
> they are not delivery claims. In this repo nothing is shipped until its
> intent reaches `intents/shipped/` (see
> [`intents/README.md`](../intents/README.md)). Current delivery state splits
> by kind (per [adr-5](../decisions/adrs/0005-brief-is-current-state.md)).
> What exists is stated in the brief: each surface chapter's `shipped` /
> `staged` table and the Status column of
> [`04-surfaces/README.md`](04-surfaces/README.md), which `surface_coverage`
> checks against the command tree, and the verification matrix
> ([`06-delivery/02-verification-matrix.md`](06-delivery/02-verification-matrix.md)),
> whose design-target rows open **(staged)**. What is scheduled or in flight
> is the status board's Now / Next / Later block (on the bare `abcd` board
> and the site's Status page), never brief prose.

> **Naming and structural conventions:** see [`00-meta.md`](00-meta.md). It explains why this directory uses numbered folders and the brief↔lifeboat shape contract.

## Navigation

| Folder | Purpose |
|---|---|
| [`01-product/`](01-product) | The why and what — press release, context, mental model, scope, personas |
| [`02-constraints/`](02-constraints) | Hard locked decisions — platform, dependencies, invariants, naming |
| [`03-evidence/`](03-evidence) | What worked / what didn't / open questions / tradeoffs — live; lifeboat extraction reads it, never solely populates it |
| [`04-surfaces/`](04-surfaces) | One file per user-facing command surface |
| [`05-internals/`](05-internals) | Plumbing the surfaces depend on — indexed in [`05-internals/README.md`](05-internals/README.md) |
| [`06-delivery/`](06-delivery) | Build sequence, verification matrix, out-of-scope |
| [`glossary/`](glossary) | The canonical terminology glossary — one file per term per bounded context |

The directory layout (each folder's `README.md` — where present — indexes its files):

```
brief/
├── 00-meta.md                   # naming convention, archive policy, structure rationale
├── README.md                    # this index
├── 01-product/                  # press release, context, mental model, scope, personas
├── 02-constraints/              # platform, dependencies, invariants, naming
├── 03-evidence/                 # what worked / what didn't / open questions / tradeoffs
├── 04-surfaces/                 # one file per user-facing command surface (see 04-surfaces/README.md)
├── 05-internals/                # agents, adapters, configuration, universal patterns, prompt quality, lint, memory, skills, provenance, site (see 05-internals/README.md)
├── 06-delivery/                 # build sequence, verification matrix, out-of-scope
└── glossary/                    # the canonical terminology glossary, by bounded context (see glossary/README.md)
```

## Three-layer reading guide

For a fast orientation:

1. **Start with [`01-product/03-mental-model.md`](01-product/03-mental-model.md)** — the brief / intents / specs distinction underpins every other section.
2. **Then [`02-constraints/01-platform.md`](02-constraints/01-platform.md)** — what's locked, what's not.
3. **Skim [`04-surfaces/README.md`](04-surfaces/README.md)** — the command surfaces at a glance.
4. **Drill into [`05-internals/`](05-internals) only when implementing** — its README indexes the ten chapters.
5. **Finish with [`06-delivery/01-build-sequence.md`](06-delivery/01-build-sequence.md)** — the order things ship.
