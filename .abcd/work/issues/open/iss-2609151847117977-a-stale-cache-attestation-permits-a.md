---
schema_version: 1
id: "iss-2609151847117977"
slug: "a-stale-cache-attestation-permits-a"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "security review of the owned-copy attestation before the v0.9.0 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/cache_attestation.go, hooks/bootstrap.sh"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (renewed by run A 2026-09-29 after the v0.10.0 grant lapsed at the v0.11.0 anchor): Cache attestation freshness: a release_tag binding, an attested_at bound, or a rewrite on every manifest-reaching run?"
remedy: "Waits on ruling W (a release_tag binding, an attested_at bound, both, or kept): add the freshness term the ruling picks to the cache attestation in internal/core/ahoy/cache_attestation.go and hooks/bootstrap.sh: if a release binding, write release_tag and refuse promotion when the cache's binary-meta tag differs or is older than the newest tag this machine has recorded; if a time bound, read the already-written attested_at and refuse past the bound until an online session re-attests; if both, both; if kept, document the downgrade window in the threat model. Prove it with a test that replays the record's four steps (restore an older attested artefact) and expects the promotion refused."
---

The home-scoped cache attestation (GHSA-4q78-ccfv-f374) binds exactly two
things: a `data_dir` and a `binary_sha256`. It carries no release tag and no
freshness bound, and it is rewritten only when a bootstrap run provisions a
root with manifest trust established. So the record keeps vouching for the
hash it was written with, indefinitely, after the release it names has been
superseded.

## The mechanism

1. A session provisions the cache online. `hooks/bootstrap.sh` §9b writes
   `data_dir=<D>`, `binary_sha256=<H(v1)>`, `cache_trust=manifest`.
2. Releases move on. `v2` ships, `v1` is withdrawn — a bad build, a fixed
   vulnerability, whatever the reason a release stops being the one to run.
3. Whoever can write into `<D>` restores the `v1` artefact and its
   `binary-meta` there. Both are genuine bytes from a genuine release, so
   nothing about them is forged.
4. `abcd ahoy install` finds the attestation naming `<D>` and `H(v1)`, finds
   the co-located record carrying `H(v1)`, hashes the artefact to `H(v1)` —
   three agreements — and promotes `v1` to the owned PATH copy with its
   provenance recorded.

No step is a forgery. The attestation was written by the bootstrap after a
real manifest check, and every hash is a real release's. The record simply has
no way to say *which* release it attested or *when*, so it cannot distinguish
"the release this machine authenticated" from "a release this machine
authenticated once".

## The bound

This is a DOWNGRADE to a previously attested genuine release, never a promotion
of attacker-chosen bytes: the three-way agreement still holds the property the
advisory's fix established. An attacker who cannot produce an artefact hashing
to an attested value gains nothing here. What they can do, given write access
to the attested directory, is pin the machine to an older release that this
machine really did authenticate at some point in its past — and hold it there,
because the attestation is rewritten only by a bootstrap run that reaches the
manifest, and a machine that never goes online again never rewrites it.

The write access the step-3 restore needs is the same access the accepted
same-uid residual already grants (iss-2609012039107700). The distinct fact
here is not the write but the DURATION: the record survives the release it
describes, so the window is not "until the next session" but "forever".

## What a fix would have to add

Some freshness term the record does not carry today. Candidates, none chosen
here:

- a `release_tag` in the attestation, compared against the tag the cache's
  `binary-meta` records, so an artefact from another release fails the binding
  even when its hash was once attested;
- an `attested_at` freshness bound (the field is written but never read), past
  which the promotion declines and asks for a re-authenticating session;
- rewriting the attestation on every manifest-reaching run rather than only on
  a provision, so a machine that goes online re-anchors to the current release.

Not fixed in the cut this was found in: the advisory's own property holds, the
finding is a strictly narrower residual, and each candidate above changes when
the record moves — which is a decision about the record's contract, not a
patch.

## Remedy grounds (2026-09-29)

- The threat is a downgrade (rollback) or freeze to a previously attested release; TUF names exactly these two and answers them with a monotonic version check (spec section 5.4.3) and an expiry on the metadata (sections 1.5.2, 5.4): https://theupdateframework.github.io/specification/latest/ (consulted 2026-09-29). Each ruling option maps to one of those two defences, so the remedy states each and picks none (ruling W is unanswered).
- Rejected: adopting a TUF client library, which is a new dependency needing the person's sign-off and far heavier than one field and one comparison.
