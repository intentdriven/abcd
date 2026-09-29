---
schema_version: 1
id: "iss-2609291731336469"
slug: "the-transcript-store-refuses-every-history-verb-reads"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/history/location.go"
remedy: "Waits on ruling CB1: in Resolve (internal/core/history/location.go): if (a) the refusal of every verb stands, word it per verb ('refusing to read or write transcripts in it') so history list no longer reports a write; if (b), a read skips the mode narrowing with a Note and only a write refuses. Prove the answer with a test that stubs the records leaf's owner lookup to a foreign uid and asserts list, show and capture each get the ruled outcome."
deferred_after: "v0.11.1"
deferral_reason: "owes ruling CB1 (rulings-owed 2026-09-25): whether a foreign-owned records leaf keeps refusing every verb, or a read skips the narrowing with a Note and only a write refuses; the choice is a trust-boundary ruling for the product thinker, not a lane's call, and no working setup is lost meanwhile (the owner's writes into a root-owned 0o700 leaf already failed)."
---

The transcript store refuses every history verb, reads included, when its records leaf is owned by another uid, and says so with write wording: Resolve's owner check (internal/core/history/location.go:223) reports 'not owned by this account; refusing to write transcripts into it' even for history list. The realistic hit is a root container over a bind-mounted checkout with the local store declared, where reads and hook capture both refuse. Precedent runs both ways: rules/root.go and fsutil.go refuse foreign-owned reads, home.go admits a root-owned home. Fix-later shape from the review: on a foreign owner, skip the narrowing with a Note on a read and refuse only a write.

## Remedy grounds (2026-09-29)

- Why: CB1's two shapes, each with its change; the wording fault is real under both, so (a) still carries a fix. The ruling is unanswered and none is picked.
- Sources (consulted 2026-09-29): git's safe.directory 'will refuse to even parse a Git config of a repository owned by someone else', reads included, and admits another owner only through an explicit declaration or the SUDO_UID case (https://raw.githubusercontent.com/git/git/master/Documentation/config/safe.adoc). That precedent supports (a) for data a later session reads back as context, and it is the evidence the ruling should weigh against (b)'s bind-mount convenience.
- Rejected: admitting a root-owned leaf silently, which would read records any process with root could have planted.
