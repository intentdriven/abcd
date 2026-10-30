---
schema_version: 1
id: "iss-2610291946475029"
slug: "test-packages-reach-the-machine-gh-login"
severity: "minor"
category: "tech-debt"
source: "review-followup"
found_during: "#1147 merge-queue failure follow-up, 2026-10-29"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/github"
remedy: "Give each of these packages the same TestMain guard #1147 adds to the landing, claim, implement loop and CLI test packages (HideGHLogin in internal/gittest: an empty GH_CONFIG_DIR and the gh token variables unset), and add a check that names any test package depending on internal/core/github without it."
---

Twenty-one test packages depend on internal/core/github but have no TestMain hiding the machine's gh login: afterupdate, ahoy, autonomy, capture, embargo, github, ideate, implement, lifeboat, reading, record, release, repolint, report, reservesweep, scribe, site, sweep, update, surface/session and surface/session/web. A test in any of them that reaches the forge reads the developer's real login and tokens, so it can pass locally and fail on CI, which has no login, as the landing body test did in the merge queue for #1147.
