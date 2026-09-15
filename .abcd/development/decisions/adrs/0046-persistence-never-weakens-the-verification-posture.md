---
id: adr-46
slug: persistence-never-weakens-the-verification-posture
status: superseded
date: 2026-08-21
supersedes: null
superseded_by: adr-2609151706587280
related_intents: [itd-105, itd-132]
related_rfcs: []
related_adrs: [adr-38]
---

# ADR-46: Persisting the hook binary never weakens the verification posture — every promotion re-verifies, and SessionEnd performs no network work

## Context

itd-105 (spc-21) bought the hook binary's provenance dearly: same-origin
checksums.txt pinned to one resolved release, HTTPS-only transport on every
fetch including redirects, `-q`-first curl with the proxy and CA environment
scrubbed before any request, and — after two adversarial-review defeats — no
environment seam of any kind for the fetch origins. Two properties bounded the
residual risk: the artefact lived in the harness's commit-stamped plugin directory,
which every plugin update replaced, so a tampered binary survived at most one
update cycle; and every trusted byte had just come off the verified download
path.

itd-132 (spc-35) moves the artefact into the harness's persistent per-plugin
data directory, where it survives every update and is deleted only on full
uninstall. Persistence is the point — it is what ends the re-download and the
missing-binary window on every update — but it changes the threat shape twice
over: the at-rest tampering window extends from one update cycle to unbounded,
and the artefact now travels again after fetch time, promoted from the cache
into fresh plugin roots, onto PATH, and (at migration) from a plugin root into
the cache. Each promotion is a fresh opportunity for the trusted path to pick
up bytes the verification never saw. The trust rule outlives any one feature
— it binds the bootstrap, `ahoy install`, `abcd update`, and every later verb
that moves the artefact — so per the itd-84 SPLIT routing it lives here rather
than in the intent's prose (iss-2608210934566226).

## Decision

1. **The fetch-time posture carries over verbatim, wherever the download
   lands.** Same-origin checksums pinned to one release, the HTTPS pin on
   every call, `-q`-first curl with the proxy/CA scrub, and no environment
   seam for origins are invariants of fetching the binary, not of the
   directory it lands in. Relocating the artefact's home changes none of them.
2. **Every promotion of a persisted artefact re-verifies against its recorded
   `binary_sha256` and refuses loudly on a mismatch.** Cache → plugin root,
   cache → PATH copy, PATH-copy refresh, and the one-way migration seed
   (plugin root → cache) each re-hash the bytes being moved against the
   recorded provenance before anything lands in a trusted location; a
   mismatch installs nothing and leaves the mismatching artefact in place as
   evidence. This catches corruption and substitution *in transit between two
   trusted points* — a truncated copy, a half-written rename.
3. **The cache's own trust is established against the release's published
   manifest, not against its co-located record.** The recorded `binary_sha256`
   lives in the same data directory as the artefact it vouches for and is
   equally writable by any same-UID process; re-hashing the artefact against
   *that* record proves only that the file matches its own neighbour, which an
   attacker who writes both satisfies trivially. So before the cache is
   preferred over a fresh download, its recorded digest is checked against the
   published `checksums.txt` for the resolved release — the same same-origin,
   HTTPS-pinned manifest fetch spc-21 already trusts, and the tag is already
   being resolved on the cache path, so this is one extra small GET, not a new
   network posture. Online, this makes the cache tamper-evident against
   everything short of control of the release origin itself. **Offline, no
   published manifest is reachable, so the cache is trusted at
   corruption-evidence only** (rule 2): this is the deliberate
   availability trade — a machine that cannot reach the origin still boots its
   guard from a cache that has not been corrupted in place — and it is stated
   here rather than hidden, because the whole point of this ADR is that the
   posture is never *silently* weaker than it claims. The success notice says
   which trust it rests on: verified against the published manifest, or
   provisioned from an unauthenticated cache while offline.
4. **Ownership is recorded provenance, never content-guessing.** A persisted
   artefact or PATH entry is abcd's to refresh or remove only when a recorded
   hash vouches for its exact bytes; anything that stopped matching is
   foreign and is never replaced, refreshed, or deleted. The provenance record
   must be reachable wherever the verb that consults it runs — `ahoy install`,
   `ahoy uninstall`, and `abcd update` run from a terminal, where
   `CLAUDE_PLUGIN_DATA` is not exported, so a record readable only from a hook
   process cannot establish ownership at all and silently reclassifies abcd's
   own binary as foreign. The record lives in a home-scoped, abcd-owned
   location derivable without the harness environment.
5. **SessionEnd performs no network work.** The exit path never bootstraps,
   fetches, or refreshes anything — a download belongs to session start and
   the explicit verbs, never to the moment the user is leaving
   (iss-2608210934566223). This clause becomes true in the tree only when that
   fix lands; the brief invariant that mirrors it is accepted together with
   the SessionEnd hook change, never ahead of it.

## Alternatives Considered

- **Trust the persistent location because the fetch was verified.** Rejected:
  that inference held (weakly) when the harness deleted the directory every
  update; a persistent directory extends the at-rest window without bound,
  so the recorded hash must be re-checked at every point where the artefact
  is promoted into a place something executes from.
- **Trust the cache on its co-located record alone (re-hash artefact against
  the neighbouring `binary-meta`).** Rejected: this was the first shape, and
  adversarial review (2026-08-21, iss-2608210934566228) demonstrated it
  end-to-end — an attacker who writes both the artefact and its record passes
  the check, and because the cache is now preferred over the network across
  every update, the implant persists rather than healing at the next update as
  it did under spc-21. The co-located record proves corruption, never tamper;
  authenticating the cache against the published manifest (decision 3) is what
  closes it online, and the offline residual is named rather than papered over.
- **Re-verify on every execution (hash in the hook fast path).** Rejected:
  the steady-state contract is one file test and no network (spc-21, kept by
  spc-35); hashing ~11 MB on every hook event buys little — the promotion
  points are where bytes move into trusted locations — and would be paid on
  every prompt.
- **Relocate execution into the data dir instead of copying per root.**
  Rejected in adversarial review (2026-08-21, recorded in itd-132): one
  shared mutable binary across roots creates cross-root skew and locking
  semantics, collides with the dev shim, and widens the at-rest tampering
  blast radius to every session at once.

## Consequences

- spc-35's bootstrap, `ahoy install`'s owned PATH copy, and the migration
  seed each carry an explicit re-verify step; the corrupt-cache refusal is
  pinned by tests at every promotion point, and the cache-tamper case (a
  poisoned artefact whose co-located record matches it) is pinned by a test
  that rewrites artefact *and* record together and asserts the published
  manifest check rejects it online.
- `abcd update` remains the one verb that legitimately changes the PATH
  copy's bytes, and it re-stamps the recorded digest only after proving the
  new content against the release's own checksums.
- The corresponding brief invariant is invariant 12 in
  [`02-constraints/03-invariants.md`](../../brief/02-constraints/03-invariants.md).
  Its cache-authentication and promotion clauses are accepted with this
  record; its SessionEnd clause (decision 5) is accepted only when the
  SessionEnd hook change (iss-2608210934566223) lands, and the two are
  sequenced so the invariant is never present-tense-false in the tree.
  That condition has since been met: the fix landed in `baf0551`, is
  pinned by `TestSessionEndNeverBootstraps`, and the register asserts
  the clause.
- Any future verb that moves the persisted artefact (a repair verb, a
  multi-machine sync) inherits rules 2–4 as requirements, not suggestions.
