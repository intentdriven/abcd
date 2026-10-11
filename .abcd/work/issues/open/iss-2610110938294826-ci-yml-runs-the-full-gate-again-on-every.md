---
schema_version: 1
id: "iss-2610110938294826"
slug: "ci-yml-runs-the-full-gate-again-on-every"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "CI-cycles research, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
remedy: "drop ci.yml's push trigger on the default branch and add a nightly schedule run, which also keeps saving the default branch's Go build cache"
---

ci.yml runs the full gate again on every push to main, though with the merge queue every commit on main is the merge result the queue already tested: 68 push runs in the 6 days to 2026-10-11 against 71 merged pull requests, each taking a macOS slot.
