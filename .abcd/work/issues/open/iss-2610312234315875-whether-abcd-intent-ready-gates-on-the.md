---
schema_version: 1
id: "iss-2610312234315875"
slug: "whether-abcd-intent-ready-gates-on-the"
severity: "minor"
category: "inconsistency"
source: "agent-observation"
found_during: "splitting a remedy after review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/ready.go"
remedy: "Gate abcd intent ready on the claim_sections start check, per the product thinker's ruling of 2026-11-01 (dcn-2611011332291422), which reverses the iss-2609091009111294 advisory ruling for the scope-conditions row alone: ReadyWithQuestions (internal/core/intent/ready.go) appends the claim_sections check beside the open-questions one, so a record without a Scope Conditions section reads NOT READY naming the section, and a section holding None stated. reads READY; proven by a test that a record lacking the section is refused by intent ready exactly where abcd build refuses it, and one that None stated. passes both."
---

Whether abcd intent ready gates on the claim_sections start check is a ruling the product thinker has not given: gating it reverses their ruling on iss-2609091009111294, which kept the mechanism and scope-condition rows advisory, so intent ready can report READY on a record that abcd build then refuses at claim_sections, as itd-39 showed on 2026-10-31. Split from iss-2610311501473953 so its drain-buildable half carries no ruling.
