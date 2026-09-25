---
schema_version: 1
id: "iss-2609091611458541"
slug: "parking-the-ledger-gates-will-re-drift-two-corrected-chapters"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "release-gate"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md"
resolution: "already fixed at tip: the parking change 667a57f4 corrected 05-intent (four gating rows, three advisory), the capture chapter and the command pages in the same diff"
impact: internal
resolved_by:
  commit: "667a57f4fad4198182c01e82d5b7106474064dd6"
---

The change that parks the ledger's human gates edits the intent and capture surface chapters, both command pages and the generated reference, and the maintainer has ruled it out of the v0.8.0 release. The brief correction this release performs therefore fixes those two chapters against behaviour that the parking will change immediately afterwards: the intent readiness report will gate on four rows rather than seven with three rendering advisory, the promote and resolve verbs will stop refusing an absent grounds entry, and a lapse capture without its instant will record none rather than refuse. Nothing is wrong with either change on its own; the cost is that the two chapters are corrected twice and the semantic cross-check that clears a release has to run again over them, because a receipt names the commit its reviewers read and cannot vouch for prose written after it. The order was chosen deliberately and with the cost stated, so this record exists to make the second pass expected rather than discovered. Whoever lands the parking should correct those two chapters and both command pages in the same change, the way a surface change is supposed to carry its own record, and should expect the release after it to want a fresh cross-check rather than the one this release earned. Detector: the chapters describing the intent readiness rows and the capture verbs' grounds requirement state what the shipped verbs do at the commit a release gate reads, and a change to either verb's refusal set arrives with its chapters in the same diff.

## Grounds

- pursued: the chapters describe the parked gates as they behave; shown wrong if 05-intent or the capture chapter still describes a grounds refusal that the binary no longer makes
