# Script-first MVP

**The rule.** When a new capability's contract is uncertain, build it first as
a local script (bash and the standard Unix toolbox) operating on real data,
outside the Go core. The script's job is to *discover the contract* — the
on-disk shapes, the failure modes, the verbs worth having. The Go core absorbs
the proven contract, never the script; the script then serves as reference
implementation and dogfood until the native verb is wired.

**Why.** The single-binary boundary makes premature absorption expensive: a
wrong contract baked into `internal/core` costs a schema migration and a
release, while a script iterates at conversation speed against real data. What
stabilises under real use is the on-disk contract — folders, JSON shapes, exit
codes — which is exactly the part a Go port keeps verbatim. Learning happens
where iteration is cheap; permanence happens where the contract is proven.

**Bounds.**

- A script MVP lives in the personal/local tier (a user-home `bin/`, a repo's
  `scripts/`), never ships as product behaviour, and is never the thing "wired
  or it isn't done" is claimed against — that bar applies to the Go
  absorption.
- A plugin surface may front a script MVP early (one surface across the later
  swap), but it must stop gracefully where the script is absent.

**Live instance.** The sources-corpus tooling (user-tier registrar, ingest
wrapper, banlist projection, cite guard) proved the per-source-folder +
CSL-JSON + append-only-ledger contract as bash before any Go verb exists; its
absorption into the core is iss-27, converging with the itd-36 provenance
substrate. The consult and ingest skills that fronted it became command pages
and then retired into `/abcd:library`, which fronts it per the bounds above.

**Promotion.** This principle is the capability-specific instance of the
MVP → tool edge of the ladder in this directory's [README](README.md): the
script is the enabling MVP, the Go-core absorption the tool. No enforcement
hook exists; per the promotion path in this directory's README, this file
becomes a discipline-kind intent the moment one does.
