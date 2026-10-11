---
schema_version: 1
id: "iss-2608241612087533"
slug: "the-release-derivation-cannot-tell-a"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "v0.6.4 release validation 2026-08-24"
found_at: "internal/core/release/emit.go"
resolution: "the derivation honours a stated shipped_in field: a record naming the release that already carried its work leaves the cut, so it neither drives the version nor earns a changelog line. Written by abcd capture resolve --shipped-in, bare like impact. Nothing is inferred — an earlier reachability design could have judged 85 of 376 resolved records and guessed at the rest. Every rejection keeps the record IN the cut and reports why: unreadable, names no tag, or names a tag the anchor cannot reach. Shape alone is refused, because a mistyped but well-formed version would otherwise drop a record silently. The Removed side is deliberately unfiltered: a withdrawal is this cut's event whenever the work first shipped."
impact: additive
---

the release derivation cannot tell a back-filled resolution from a fresh one, so a ledger hygiene sweep re-announces old work as this release's content. internal/core/release/emit.go and internal/core/changelog/shipped.go read only folder membership and the record's impact field; neither reads resolved_by. The cut therefore equates 'record entered resolved/ since the base tag' with 'shipped in this release'. That equation holds only because RS001 now lands resolution inside the fixing commit, and a bulk hygiene sweep is the legitimate exception. Live instance: the 2026-08-24 sweep (089c0d61) closed 33 already-fixed records, and the v0.6.4 cut renders 14 of them as changelog entries for work first tagged v0.4.0, v0.4.1, v0.4.2 and v0.6.2 — four of the 14 are already cited by id under the 0.6.2 heading. The four records deciding the additive impact class (iss-98, iss-99, iss-100, iss-158) all describe pre-v0.6.3 work, so the version and the receipt tier rest on them too. Candidate fix: exclude an added record whose resolved_by.commit is reachable from the base tag, which needs the stamp back-filled on the six swept records that carry none.