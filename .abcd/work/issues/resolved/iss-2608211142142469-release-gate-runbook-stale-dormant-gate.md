---
schema_version: 1
id: "iss-2608211142142469"
slug: "release-gate-runbook-stale-dormant-gate"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-b-round-4"
found_at: ".abcd/development/release-gate/README.md"
resolution: "Corrected the runbook's stale dormancy framing: the blockquote now reads 'Live, and visibility-gated' and records the two real fail-closes (v0.3.0/iss-108, v0.6.0/iss-326); procedure step 4 states receipt_gate is armed fail-closed and that a receiptless tag is permanently unpublishable (anti-tag-move); the self-reference parenthetical corrected to name the v0.3.0 firing. Docs-only."
impact: internal
---

The release-gate runbook describes the semantic-receipt gate as 'Dormant until the public flip' (README.md blockquote) and phrases arming as a future condition ('Once the fail-closed verify rule is armed, the tag is rejected'), but the repo is public and the gate is armed and fed receipts on every release — it has fail-closed two: v0.3.0 (iss-108, the self-reference defect) and v0.6.0 (iss-326, a missing receipts commit, leaving a permanently unpublishable tag). The runbook a maintainer follows at tag time thus tells them receipts are not yet required, which is iss-326's proximate cause. It also misstates the failure mode ('the tag is rejected' — the tag survives, anti-tag-move; the publication is refused) and self-contradicts at the adjacent parenthetical ('dormant... never exercised' vs iss-108's v0.3.0 firing). Inverts the enforcement-claims-are-facts principle (an under-claimed live gate).