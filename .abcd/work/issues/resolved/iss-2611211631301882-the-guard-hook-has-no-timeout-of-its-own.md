---
schema_version: 1
id: "iss-2611211631301882"
slug: "the-guard-hook-has-no-timeout-of-its-own"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "review-of-guard-coverage"
origin: researcher-authored
production_mode: hand-written
found_at: "hooks/hooks.json"
remedy: "Give the PreToolUse hook a timeout and bound the hook's git calls with a deadline, saying when a call was not judged"
resolution: "The guard hook's first git calls (the checkout's top level for the session directory and the call's workdir, and the question route's lookup) run under a 5-second budget; when git has not answered, the hook says on stderr that it could not judge the call and why. The hook keeps the host's default timeout, per the ruling recorded on iss-323."
impact: additive
resolved_by:
  commit: "d5693ddee"
---

The guard hook has no timeout of its own and its git calls (such as finding the checkout's top level) have no deadline, so a repository config that includes a file which never finishes reading stalls every tool call until the host's timeout, after which the call runs unjudged.
