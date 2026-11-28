---
schema_version: 1
id: "iss-2611280938411357"
slug: "review-of-the-ahoy-gap-step-fix-left-two"
severity: "nitpick"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli_test.go"
remedy: "Empty CLAUDE_CONFIG_DIR in hermeticEnv as hermeticRepo does, with a test that fails without it, and reword that resolution to say doctor prints the step under every diagnostic."
---

Review of the ahoy gap-step fix left two follow-ups: the hermeticEnv test helper leaves a set CLAUDE_CONFIG_DIR in place, so a test using it can read the developer's host configuration, unlike hermeticRepo; and the resolution of iss-2611252138558553 says doctor adds the step only under a diagnostic whose detail lacks it, while the code prints it under every diagnostic.
