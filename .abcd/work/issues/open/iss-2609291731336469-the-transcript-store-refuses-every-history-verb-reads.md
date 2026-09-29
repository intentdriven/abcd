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
deferred_after: "v0.11.1"
deferral_reason: "owes ruling CB1 (rulings-owed 2026-09-25): whether a foreign-owned records leaf keeps refusing every verb, or a read skips the narrowing with a Note and only a write refuses; the choice is a trust-boundary ruling for the product thinker, not a lane's call, and no working setup is lost meanwhile (the owner's writes into a root-owned 0o700 leaf already failed)."
---

The transcript store refuses every history verb, reads included, when its records leaf is owned by another uid, and says so with write wording: Resolve's owner check (internal/core/history/location.go:223) reports 'not owned by this account; refusing to write transcripts into it' even for history list. The realistic hit is a root container over a bind-mounted checkout with the local store declared, where reads and hook capture both refuse. Precedent runs both ways: rules/root.go and fsutil.go refuse foreign-owned reads, home.go admits a root-owned home. Fix-later shape from the review: on a foreign owner, skip the narrowing with a Note on a read and refuse only a write.
