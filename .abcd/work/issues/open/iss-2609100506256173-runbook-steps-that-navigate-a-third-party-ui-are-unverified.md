---
schema_version: 1
id: "iss-2609100506256173"
slug: "runbook-steps-that-navigate-a-third-party-ui-are-unverified"
severity: "major"
category: "process"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-07/08; re-filed into abcd 2026-09-10"
origin: researcher-authored
production_mode: hand-written
deferred_after: "v0.8.0"
deferral_reason: "Runbook steps that navigate a third party's interface cannot be verified by anything abcd runs, and the record's own measurement shows doc-sourced instructions failing where screenshot-sourced ones held. What to do about instructions whose truth abcd cannot check is a question about what a runbook is allowed to claim, not a defect to patch."
found_at: "conventions (agent runbook guidance for managed repos)"
---

An agent walking an operator through a third-party hosting dashboard produced four successive sets of instructions, none of which matched the screen in front of them. The task — create a hosting API token, put it in two forge secrets, run a workflow — is mechanically trivial and took roughly ten exchanges, most of them the operator saying the instruction did not match what they could see.

The root cause is not that the vendor's documentation was stale, though it was. It is that the agent treated documentation prose as evidence about a running UI. Fetching the vendor's guide felt like verification (it is a primary source, it was quoted accurately) but a doc sentence is evidence of what someone wrote, not of what the page renders today. The agent had the operator's screenshots available from the fourth exchange onward and only then began giving instructions that landed; it also had browser automation tooling in-session and never once offered to look at the page itself.

The two classes of claim can be scored separately, and the split is not subtle.

Claims sourced from vendor documentation: four made, four wrong. Two were recalled and described the wrong one of two similar token pages. One was fetched from the vendor's own current guide during the session and quoted exactly; it named a dropdown the page does not have. One, also fetched, named two form fields that do not exist in the current UI at all — the shipped page instead pre-fills two already-scoped policy boxes, and there is nothing to fill in. The operator asked directly, "you asked me to fill out those, where are they?" The third instruction is the instructive one: by every ordinary test it was verified, and it was still wrong about the screen, because the page had been rebuilt and the prose had not.

Claims sourced from a screenshot: four made, four correct on first attempt. Same agent, same task, minutes apart. 0/4 before, 4/4 after.

A compounding failure with the same shape. The agent withheld the hosting account id for four exchanges as a "secret value", per a standing instruction that named it as one of the two secrets the operator alone must create. The id was already public in the repository's own pull-request check links, and was visible in the operator's address bar throughout. Every URL the agent gave therefore arrived as a template with a placeholder, which is precisely what made them unusable as instructions. Withholding it protected nothing and cost most of the confusion. The rule's purpose is to keep the agent from handling credentials; an account identifier that grants nothing on its own, that the operator owns, and that is already published, is not the thing the rule is for. A privacy rule applied without reading its purpose degraded the work while protecting nobody. The cost landed as unusable instructions rather than as a visible refusal, so nothing flagged it.

The finding, stated so it can be tested: fetching a vendor's current documentation is not verification of a vendor's current UI. It is verification of the documentation. The two drift independently and the doc is the slower of the pair, so for any UI that has been redesigned recently the doc is reliably wrong rather than randomly wrong, which makes it worse than no source at all, because it carries the felt authority of a primary source and the agent stops looking.

Proposed rule for a managed repo. An instruction that navigates a third-party UI carries a verification tag: "seen this session" or "from vendor docs, unverified". Where the operator is already in front of the UI, one screenshot is taken before the SECOND guess, never after the fourth. Where the agent has browser tooling and the operator is logged in, looking at the page is cheaper than three rounds of correction and should be the first move. And a redaction rule needs a stated purpose alongside its pattern, so an agent can tell an identifier from an authenticator; a rule listing a public account id as a secret, with no note of which it is, will keep producing this.

What would falsify it: if UI-navigation instructions sourced from current vendor docs land, say, four times in five across a handful of vendors, the tag is unnecessary ceremony and should be dropped. The prediction here is the opposite — that redesigned dashboards make doc-sourced navigation fail most of the time, and that the failure is invisible to the agent, which is what makes a tag worth carrying.

Residue worth keeping: the corrected sequence is now known-good and was established empirically, not from any document. A verified runbook is worth committing precisely because it rots — the value is the date stamp and the screenshots, not the prose — and it should be re-verified rather than trusted on next use.
