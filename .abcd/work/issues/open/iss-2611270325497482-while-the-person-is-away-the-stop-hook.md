---
schema_version: 1
id: "iss-2611270325497482"
slug: "while-the-person-is-away-the-stop-hook"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous surge run, coordinating session, 27/11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli"
remedy: "Judge each check by its newest run per name and workflow (the latest startedAt), as the forge's merge gate does, and count a check as failing only when that run failed; add a test with a rollup that holds a failure followed by a success for the same check"
---

While the person is away, the stop hook reports a pull request as failing its checks after the failed check was rerun and passed. The forge's status rollup keeps both runs of a check, the superseded failure and the newer success, and the hook reads the failure. It then tells the session to fix checks that already pass, and every reply after that carries the false item. Seen on 27/11: an attribution check failed on a pull-request body, the body was edited, the rerun passed, and the hook went on naming the pull request as failing.
