---
schema_version: 1
id: "iss-2610230716454430"
slug: "build-runner-subscription-default"
severity: "minor"
category: "inconsistency"
source: "review-followup"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner"
remedy: "Ask the product thinker whether the build runner takes the subscription-by-default route; if yes, extend the walled route and its live wall proof to runner.claude, and if no, record why the build runner stays key-only."
---

The build runner (runner.claude) still follows ruling RN2's key-only rule (dcn-2610020000000016): it runs claude with --bare on an API key, and a person with only a subscription runs the role in their own session. On 2026-10-23 the product thinker ruled that a terminal drive runs on the person's subscription by default behind a wall, with an API key optional per configuration (dcn-2610230716408975). The two routes now differ, and the product thinker has not said whether the build runner takes the same default.
