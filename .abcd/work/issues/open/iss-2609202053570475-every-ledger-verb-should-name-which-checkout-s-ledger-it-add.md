---
schema_version: 1
id: "iss-2609202053570475"
slug: "every-ledger-verb-should-name-which-checkout-s-ledger-it-add"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "record-discipline review of the peer-listing draft, 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
---

Every ledger verb should name which checkout's ledger it addressed. The ledger is per worktree and says so nowhere: a capture filed in one worktree is invisible to capture resolve in another, and the verb's refusal reads "not found" with no hint that another checkout holds the record. The cheapest always-on remedy, asked for in the 2026-09-18 corroboration on iss-2609020716570699 and routed out of the peer-listing intent by its record-discipline review on 2026-09-20 as a cross-cutting change: one line from every ledger verb (capture, resolve, wontfix, promote, list, the record dispatcher, intent audit) naming the checkout root and branch whose ledger it read or wrote, on stderr in the text render and as a member in --json. Distinct from the peer listing (itd-2609091416295622), which reads other checkouts; this is the verb saying which one it is in.
