---
schema_version: 1
id: "iss-2610241300370199"
slug: "an-identity-pin-that-holds-only"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "review pair of feat/setup-redesign, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/identity"
remedy: "migrate production_mode on its own for any contributor at a terminal, and drop a pin that names no human identity with its own explained step"
---

an identity pin that holds only production_mode, or a machine identity, is owned by no one under the migration rule, so it is never removed and its production_mode never moves to .abcd/config/provenance.json, and every contributor is told the repository stores a commit identity that is not theirs.
