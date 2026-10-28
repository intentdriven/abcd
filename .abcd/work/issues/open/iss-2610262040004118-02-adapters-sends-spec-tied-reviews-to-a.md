---
schema_version: 1
id: "iss-2610262040004118"
slug: "02-adapters-sends-spec-tied-reviews-to-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 902 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/02-adapters.md:284"
remedy: "none (filed automatically)"
related_intents: [itd-28]
finding_ref: cfk-7659910dd3ea55be
---

02-adapters sends spec-tied reviews to a native spec review store; itd-28 dropped that store, so the dated folders and gate receipts are the store.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/02-adapters.md:284` — "spec-tied reviews go straight to the native spec"
- End B: `.abcd/development/intents/shipped/itd-28-rp-reviews-into-flow.md:112` — "the dated review folders and the gate receipts are the store"

02-adapters' RepoPrompt harvesting section says spec-tied reviews go straight to the native spec review store and are never swept, and its reviews reader reads spec-tied reviews beside the charter folders. itd-28's first decision drops the review store its scope described, so the dated review folders and the gate receipts are the store. Its third decision says no abcd verb files a dated folder. The brief names a store that the shipped intent removed.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 902.
