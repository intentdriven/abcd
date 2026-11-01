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
remedy: "Gate abcd intent ready on the claim_sections start check, per the product thinker's ruling of 2026-11-01 (dcn-2611011332291422), which reverses the iss-2609091009111294 advisory ruling for both claim rows: ReadyWithQuestions (internal/core/intent/ready.go) appends the claim_sections row from intent.StartChecksIn beside the open-questions one, so a record with an unanswered or empty Mechanism, or without a Scope Conditions section, reads NOT READY naming what is missing, and one whose Mechanism is answered and whose Scope Conditions holds None stated. reads READY; proven by tests that intent ready refuses each failing record exactly where abcd build refuses it at claim_sections, and passes a record both accept."
---

Whether abcd intent ready gates on the claim_sections start check was ruled on 2026-11-01 (dcn-2611011332291422): it gates. Before that ruling, gating it would have reversed their ruling on iss-2609091009111294, which kept the mechanism and scope-condition rows advisory, so intent ready can report READY on a record that abcd build then refuses at claim_sections, as itd-39 showed on 2026-10-31. Split from iss-2610311501473953 so its drain-buildable half carries no ruling.
