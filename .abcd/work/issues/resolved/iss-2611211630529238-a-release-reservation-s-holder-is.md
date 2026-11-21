---
schema_version: 1
id: "iss-2611211630529238"
slug: "a-release-reservation-s-holder-is"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "review-of-reservations"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/withdrawchain.go"
remedy: "Read the release branch on in pages past 64, to a hard limit, and name the recovery step in the refusal past it"
resolution: "The release reservation's chain read goes on in pages of 64 to a hard limit of 512 commits to find the holder's newest push, and past the limit the refusal names the manual re-arm step."
impact: additive
resolved_by:
  commit: "dac096cc3"
---

A release reservation's holder is refused every act once other accounts' commits fill the 64 read above the holder's newest push, with no way back but re-arming withdrawn pull requests by hand.
