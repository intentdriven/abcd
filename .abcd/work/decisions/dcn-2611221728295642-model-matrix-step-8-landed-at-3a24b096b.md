---
id: dcn-2611221728295642
slug: model-matrix-step-8-landed-at-3a24b096b
date: 2026-11-22
---
2026-11-22 — Model-matrix step 8 landed at 3a24b096b with two minor review findings left open, each filed as its own issue, and was not patched again before landing. iss-2611221728031097 is a comment fix: two comments understate what the network screen withholds. iss-2611221728048753 is a code fix: a refusal on a carried record names a line of abcd's re-encoding. Nine review rounds had run and both reviewers passed that head. The comment fix would have cost a further security re-read and a full check for two comment lines. The refusal fix needs a new field carried through the nine places that build the screened list, a change that needs a review of its own. This record is the deferral, naming its findings and its reasons, that .abcd/development/principles/pre-existing-is-not-a-defence.md asks for. That principle is written for a defect met while doing other work; it is applied here in spirit, to findings on the change itself.
