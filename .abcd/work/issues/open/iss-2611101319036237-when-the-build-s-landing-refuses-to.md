---
schema_version: 1
id: "iss-2611101319036237"
slug: "when-the-build-s-landing-refuses-to"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "Have the refusal list, for each open issue, the moves that settle it (resolve with the lane's commit, unlink, or move the link to another intent), and let a decision record written for that settlement ride in the landing's records commit; a test refuses a close, settles an issue by the offered move and finds the decision in the commit."
---

When the build's landing refuses to close a spec because the intent's related_issues hold open issues, it names the issues but offers none of the moves that settle them, and a decision record that settlement needs cannot enter the landing's commit, which takes only records the landing's own moves made. On 8 Nov at 01:02 the links were settled by hand (two issues resolved, four unlinked, one gap captured), and the decision record for that judgement was refused and held in the main checkout's scratch until a follow-up landed it. iss-2611030450356845 covers the related case where a shipped intent's open issues fail issue-drift at preflight.
