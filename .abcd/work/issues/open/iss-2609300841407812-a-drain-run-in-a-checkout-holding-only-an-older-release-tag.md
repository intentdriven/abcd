---
schema_version: 1
id: "iss-2609300841407812"
slug: "a-drain-run-in-a-checkout-holding-only-an-older-release-tag"
severity: "major"
category: "security"
source: "user-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
remedy: "Compare each deferred_after with the local anchor through launch.CoreGreater, the canonical version comparison: one newer than the local tag names a tag the checkout lacks, so hand the record back as 'anchor stale', naming the tag and 'git fetch --tags', the way the anchor-unknown path does, with no remote call; name the stale tag in the dry run and --json; treat a deferral past the local tag as lapsed once a newer one is named; hand back a deferred_after that does not parse as vMAJOR.MINOR.PATCH."
---

A drain run in a checkout holding only an older release tag takes a record a person deferred past a newer one. liveDeferralAnchor (internal/core/capture/eligible.go) reads the checkout's newest local tag as the anchor, and eligibility compared deferred_after to it by equality, so a clone not fetched since the last cut (a stale worktree, a --no-tags remote) read a deferral past the newer tag as lapsed and made the record eligible. A deferred_after that is not a release tag at all (hand-written 'next') was let through the same way. Reproduced on b93f4cdd3 by reverify-drainOwnRule: clone holding v0.1.0 only, record deferred_after v0.2.0 reads eligible.
