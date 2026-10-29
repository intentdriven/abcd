---
schema_version: 1
id: "iss-2610290913450522"
slug: "required-checks-accept-any-source"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "follow-up to the settings-audit change, 2026-10-29"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/remote_baseline.go"
remedy: "Pin each required check to the app observed producing it in the check runs of recent commits, looking the app id up, never hard-coding it; leave a check no recent run produced unpinned and name it; report and keep on any source a check produced by two apps or by a commit status; treat pinning an existing required check as a tightening applied on the yes; test each case, record the decision with GitHub's rulesets and statuses-API grounds, and refresh main-protection.json from live"
refines: [itd-2610290746077716]
---

The settings baseline requires each status check from any source. GitHub lets any person or integration with write access set the state of any status check, and the statuses API lets a push-access token post any context, so a required check left on any source can be set to passing by someone other than the workflow that runs it. Choosing the app as the source refuses a status set by anyone else. Assessed against the abcd-fixes-what-it-adds principle: the route exists without abcd (it is GitHub's default for any required check), and abcd's apply adds no weaker state, since without it the repository requires no check at all; pinning is a tightening of a setting abcd itself writes, applied only on the person's yes, and changes nothing else about how the repository works. No released abcd has ever applied a required check: the apply that writes one ships for the first time in the change itd-2610290746077716 delivers, so abcd has put no managed repository in this state.
