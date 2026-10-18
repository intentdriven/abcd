---
schema_version: 1
id: "iss-2610181122454939"
slug: "two-tests-origin-that-would-hang-never"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "fixing iss-2610180735158998"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/status_test.go"
remedy: "Write the origin as 'ext::sh -c sleep% 30' so it really hangs, and in the hook test also leave a marker file from the origin's command, so reaching origin is decided without the clock."
---

Two tests' 'origin that would hang' never hung: TestThePromptHookShowsEachAnnouncementOnce (internal/surface/cli/status_hook_test.go) and TestThePromptNeverWaitsOnTheNetwork (internal/core/claim/status_test.go) set origin to 'ext::sh -c sleep\ 30', but git-remote-ext escapes a space only as '% ', so sh -c got 'sleep' alone, which printed its usage and exited at once. A network call from the code under test would have failed fast instead of hanging, so the hook test's no-network claim rested on its one-second clock alone; the claim test also traces git's subcommands, which held its claim.
