---
schema_version: 1
id: "iss-2610120601214392"
slug: "bootstrap-env-read-test-flags-self"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "security fix lane B2, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "hooks/bootstrap.sh"
remedy: "Make the allowlist test treat a parameter expansion of a variable the script itself assigned earlier as a local read, not an environment read; prove with a test case for x=${x#* }"
---

The bootstrap's environment-read allowlist test treats a variable reassigned from itself (x=${x#* }) as an environment read, so new shell code in hooks/bootstrap.sh has to use fresh variable names to pass it. Found by lane B2 on 2026-10-12.
