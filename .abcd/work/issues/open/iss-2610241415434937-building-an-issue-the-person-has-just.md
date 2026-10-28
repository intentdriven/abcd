---
schema_version: 1
id: "iss-2610241415434937"
slug: "building-an-issue-the-person-has-just"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "capture then build of the install one-liner issue, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement"
remedy: "Make capture say at filing time whether the drain rule would take the record (and why not, naming the field), and let build iss-N preflight every refusal at once, naming each field and the record-only pull request the issue lane needs, instead of refusing on the first ground found."
---

Building an issue the person has just filed takes a pull request round trip first, with nothing in the capture or build output saying so up front. On 2026-10-24 a person asked a session to capture an issue and then build it. abcd build refused three times in a row, each on a different ground found only by running it. First, the default category the capture gave the issue (observation) is outside the drain rule's fixable set. Second, the issue lane fetches origin's default branch, so a record that exists only in the working tree cannot be built at all, and the record had to go through its own branch, preflight, pull request and merge queue (a queue run that failed once on an unrelated flaky test), about three hours. Third, once it landed, the severity the capture gave it (major) is above the drain's nitpick and minor, so the issue was a person's all along. None of the three was visible when the issue was filed, and the person's intent, to file and then build, needed four round trips to learn that it could not be built.
