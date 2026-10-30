---
schema_version: 1
id: "iss-2610230816405438"
slug: "the-release-hold-s-full-loop-an-armed"
severity: "minor"
category: "process"
source: "manual-test"
found_during: "iss-2610222014550692 live comment test"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/withdrawforge.go"
remedy: "Prove the loop on the first real release cut on this repository with a watched checklist: one armed pull request before the reservation; check its withdrawal, the withdrawn comment, and its re-arm and comment once the tag is on origin."
---

The release hold's full loop (an armed merge withdrawn at reserve, commented on, and re-armed after the tag) has no private live test: on a free personal plan a private repository can take neither branch protection nor auto-merge, so a scratch repository has nothing to arm. Only the comment writer itself (post and read-back) has run live, on 2026-10-23; the loop runs on fakes only.
