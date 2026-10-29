---
schema_version: 1
id: "iss-2610170713227840"
slug: "abcd-has-no-established-procedure-for"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "answer-screen mockups, 2026-10-17"
origin: researcher-authored
production_mode: hand-written
found_at: "design work run in labs (no command page)"
remedy: "Make the design round an abcd procedure: a command page and verb that mint a design lab for one question, scaffold the round brief from a template carrying the hard rules, ship the render script and the pairwise tool with abcd instead of copying them between labs, build the gallery from the round's files, record each round's verdict and the comparison export as probe records, and harvest the adopted design and the process measures; draw the brief template from the two labs' briefs, and decide from their findings whether the pairwise step is a default or an option."
related_intents: [itd-2610171228577260]
---

abcd has no established procedure for design work, so each visual design question is run from memory. Three times now the same process has been rebuilt by hand: the control-panel lab in another managed repository, the brief session page for abcdesign.app (lab-261016142621-a46cdaa, three rounds), and the phone and tablet answer screen (lab-261017071130-6fd3032). The process: a written round brief with hard rules (self-contained HTML, mock data, light and dark, both device sizes, a mock-only screen switcher, URL-selectable screens); five distinct directions from one agent; a render script that screenshots each screen with headless Chromium; a gallery page opened for the product thinker; the product thinker's pick; five variations of the pick; and a pairwise comparison tool whose export is the evidence. Each time the brief, the render script, the gallery and the tool were copied from the previous lab by hand, and the first lab also showed the product thinker can choose without the pairwise step, so that step's place is still open. The product thinker asked on 2026-10-17 for this to become an established procedure.

The product thinker added on 2026-10-17, after ranking round 3 of the answer-screen mockups: they do not want to download results. Two steps follow from that. First, abcd serves the comparison tool from the person's own computer and saves each judgement straight into the lab as it is made, so nothing is exported or copied. Later, the whole design round runs on abcdesign.app: abcd publishes a round's mockups to the project's private space there, the person ranks them from any device (a tablet included), a team member can be invited to rank too through the same team sign-in, and the results land in the project's record. The live brief session's relay and team sign-in are the base for the second step.

## Grounds

- pursued: we expect a product thinker to reach an adopted design in fewer rounds and with no hand steps once abcd runs the round from a template and builds each next round itself from their feedback; a design that still needs files copied, results downloaded or a person to start the next round would show it wrong

## Severity

Re-graded from major to minor by the product thinker on 2026-10-23: this record asks for a new capability, a design-round procedure that the draft intent itd-2610171228577260 carries, rather than reporting a defect in something abcd ships, so it is graded as the feature request it is.

## Checked at the default branch 2026-10-28

Still true. A managed repository's report filed the same need from the other side as iss-2610180913376757 (a design verb with design rounds, a person-only assessment and a committed design history). The two describe one capability from two origins and are carried together by the draft itd-2610171228577260.
