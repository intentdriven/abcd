---
id: adr-2609151706587280
slug: the-cache-reaches-path-only-through-a-home-scoped-attestatio
status: accepted
date: 2026-09-15
supersedes: adr-46
superseded_by: null
related_intents: [itd-105, itd-132]
related_rfcs: []
related_adrs: [adr-38, adr-46]
---

# ADR-2609151706587280: The cache reaches PATH only through a home-scoped attestation the environment cannot write

## Context

[adr-46](0046-persistence-never-weakens-the-verification-posture.md) bound the
persistent cache's trust to the release's published manifest (decision 3) and
made ownership a home-scoped recorded provenance (decision 4). Both held at the
promotions the bootstrap performs. The one promotion the bootstrap does not
perform — cache → owned PATH copy, which `ahoy install` performs — inherited
only decision 2: it re-hashed the artefact against the `binary-meta` beside it.

GHSA-4q78-ccfv-f374 (CWE-345, iss-2609012039102770) is what that leaves open.
`pluginDataDir` takes `CLAUDE_PLUGIN_DATA` from the environment as given, and
the record it re-verifies against lives in the same directory as the artefact,
equally writable by whoever chose that directory. So the check is the very
shape adr-46 decision 3 rejects for the bootstrap — a file agreeing with its
own neighbour — and it was reproduced at v0.7.0: with the variable and
`ABCD_PLUGIN_ROOT` pointed at attacker-chosen directories, `ahoy install --yes
--adopt` installed a one-byte file 0755 as `~/.local/bin/abcd`, recorded its
hash and the fake root in `~/.abcd/path-entry`, and the classifier then vouched
for it as an owned copy.

No mechanical fix closes the class without moving a documented contract. spc-35
says the data dir is taken from the harness and never derived; adr-38 says
implicit checks are disk-only; the ahoy brief documents `install` as a
no-network verb. The record held three options open for a ruling:

- **A.** Authenticate at install time against the published `checksums.txt`,
  putting a network GET on `ahoy install`.
- **B.** Bind the cache to a record the environment does not choose alone: the
  bootstrap, which runs with the harness's real data dir and has just
  authenticated the cache against the manifest, writes a home-scoped
  attestation, and the PATH promotion accepts a data dir only when it matches.
- **C.** Accept the finding as a documented residual — environment control is
  already a PATH foothold — and record it beside adr-46's offline residual.

The maintainer ruled **B** on 2026-09-15 at an interactive question. A
mechanical partial was rejected in the same record so nobody re-derives it:
cross-checking the recorded hash against the plugin-root binary defeats the
literal recipe only and breaks dogfood installs, because a source checkout used
as the plugin root with a locally built binary never matches the cache.

An ADR is never amended, always superseded, so this record carries adr-46's
five decisions forward unchanged and adds the sixth. adr-46 is retained rather
than pruned because later records cite its numbered decisions.

## Decision

Decisions 1–5 are adr-46's, in force exactly as written there:

1. **The fetch-time posture carries over verbatim, wherever the download
   lands.** Same-origin checksums pinned to one release, the HTTPS pin on every
   call, `-q`-first curl with the proxy/CA scrub, and no environment seam for
   origins.
2. **Every promotion of a persisted artefact re-verifies against its recorded
   `binary_sha256` and refuses loudly on a mismatch.** Cache → plugin root,
   cache → PATH copy, PATH-copy refresh, and the migration seed each re-hash
   the bytes being moved; a mismatch installs nothing and leaves the evidence.
3. **The cache's own trust is established against the release's published
   manifest, not against its co-located record.** Online, the cached digest is
   checked against the published `checksums.txt` before the cache is preferred
   over a download; offline, the cache is trusted at corruption-evidence only,
   and the success notice names which trust it rests on.
4. **Ownership is recorded provenance, never content-guessing**, and the record
   lives in a home-scoped, abcd-owned location reachable without any harness
   variable, because the verbs that consult it run from a terminal.
5. **SessionEnd performs no network work.**

6. **The cache reaches PATH only through a home-scoped attestation the
   environment cannot write.** The bootstrap — the one process that holds the
   harness's real data dir and has just established manifest trust for the
   bytes in the cache, whether by authenticating a cache hit or by verifying a
   fresh download against the same-origin manifest — writes
   `~/.abcd/cache-attestation`, a sibling of `path-entry` in the same key=value
   form: `data_dir`, the manifest-authenticated `binary_sha256`, `cache_trust`
   (the bootstrap's own vocabulary; only `manifest` is ever written), and
   `attested_at`. It is written whole into a sibling temp file and renamed in,
   mode 0600, after the root install's own re-hash succeeded, and only on
   manifest trust: an offline run, which proved corruption evidence and nothing
   more, neither creates nor rewrites it, and a run that could not authenticate
   leaves any existing attestation exactly as it found it. `ahoy install`
   promotes a cache into the owned PATH copy only when three things agree —
   the attestation names the directory being promoted from, the co-located
   `binary-meta` carries the attested hash, and the artefact hashes to it
   (decision 2) — whichever route named the directory, the hook's environment
   variable or the plugin root's `.data-dir` stamp. Either route is a route and
   neither is trust. A present cache the attestation does not bind is refused
   with a note that names which of the three failed, in tilde paths only, and
   the install degrades to the pinned symlink exactly as it does with no cache
   at all; detection offers the owned-copy heal on the same predicate, so the
   two surfaces never disagree about one directory. The attestation binds the
   **cache**, never the plugin-root binary: a source checkout as the plugin
   root with a locally built binary is untouched.

   The trust floor for this promotion moves from "a value the environment
   supplies" to "a write into the caller's own home" — the same floor decision
   4 already sets for ownership. An attacker who can write `~/.abcd` already
   owns `path-entry` and everything it vouches for, so the attestation grants
   nothing that authority did not already hold; an attacker who can only set
   the environment, including through a committed project settings file's
   environment block should a harness honour one, reaches the data dir and
   nothing in the home.

## Alternatives Considered

- **A. Authenticate at install time against the published manifest.**
  Rejected by the ruling: it puts a network GET on a verb the brief documents
  as disk-only (adr-38), so offline it must either degrade to the symlink
  fallback anyway or invent a second trust vocabulary; it also re-fetches what
  the bootstrap has already fetched and proved in the same session.
- **C. Accept as a documented residual.** Rejected by the ruling: "environment
  control is already a PATH foothold" is weaker than it reads, because a
  harness that honours a committed project settings file's environment block
  lets a hostile checkout set the variable for its own sessions, and the
  owned-copy claim is what the hook shims trust before running an `abcd` off
  PATH — a foothold that is also a vouched-for binary is not a residual of the
  same size.
- **Cross-check the recorded hash against the plugin-root binary.** Rejected
  as a mechanical partial: it defeats the literal reproduction and breaks
  dogfood installs, whose plugin-root binary is a local build the cache never
  matches. The attestation binds the cache, not the root, for this reason.
- **A terminal route to the cache through the attestation alone** (when the
  environment is unset and the root's stamp is missing). Not adopted, although
  it falls out of the same record: it would turn a dogfood checkout's stable
  symlink into a `symlink.legacy` gap healed to a release copy on the next
  `ahoy install`, which is the behaviour change the rejected partial was
  refused for. The stamp remains the terminal's route; the attestation binds
  whatever the stamp names.
- **Extend `path-entry` with the attestation's fields** instead of a sibling
  record. Rejected: `path-entry` records exactly one thing, the owned copy,
  and is parsed by the hook shims on every PATH resolution; the attestation
  describes the cache, is written by a different process at a different time,
  and must survive an uninstall that removes `path-entry`. The
  `~/.abcd/trusted-roots` record follows the same sibling idiom for the same
  reason.

## Consequences

- `hooks/bootstrap.sh` writes the attestation in step 9b, after the plugin
  root's re-hashed install and the `.data-dir` stamp, and only when the run
  established manifest trust; the notice reports an attestation it could not
  write. `internal/core/ahoy/cache_attestation.go` is the one reader, through
  the guarded bounded read `path-entry` uses; `cacheBindingProblem` is the one
  predicate, consumed by `cacheSourceReady` for both detection and install.
- A cache provisioned offline, or migrated from a pre-cache root, is not
  promoted to PATH until a session with network access re-authenticates it and
  attests; the install says so and degrades to the symlink meanwhile. This is
  the availability cost of the ruling and it is stated rather than hidden.
- spc-35's data-dir contract is revised in place, dated: the data dir is still
  taken from the harness or the root's stamp and never derived, and it now
  reaches PATH only through the attestation. Brief invariant 12 gains the
  clause and cites this record.
- adr-46 is superseded and retained; every citation of "adr-46 decision N"
  in code and records keeps its meaning, because decisions 1–5 are carried
  here under the same numbers.
- Any future route to the cache — a repair verb, a multi-machine sync, a
  terminal rung — inherits decision 6 as a requirement: a route is never
  trust, and only a manifest-authenticating run may attest.
