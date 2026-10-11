---
schema_version: 1
id: "iss-2608211143185943"
slug: "readme-capture-mint-describes-retired"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-b-round-4"
found_at: "README.md"
resolution: "Rewrote the README capture-mint description to name the shipped collision-proof scheme (iss-<yymmddHHMMSS><4 random digits>, never renumbered), per adr-45."
impact: internal
---

README.md described abcd capture as minting records 'with the next free id' — the proper name of the max+1 allocation protocol that adr-45/spc-33/itd-114 abolished after live collisions (one costing a release re-cut). 'Next free id' is the codebase's own name for the retired mechanism (recordid.MaxAcrossRefs, lint.go). The README is the only public document still describing it, setting the wrong mental model (small sequential successor-comparable ids) and legitimising the hand-add-at-max+1 path iss-74 identified as a collision source. Written 2026-08-17, pre-rollout.