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
found_at: "internal/core/autonomy/forge.go"
remedy: "Judge each check by its newest run per name and workflow (the latest startedAt), as the forge's merge gate does, and count a check as failing only when that run failed; add a test with a rollup that holds a failure followed by a success for the same check"
resolution: "The away mode's stop check now judges each check in a pull request's rollup (a check run by its workflow and name) by its newest run, so a failure rerun and passed no longer reads as failing. An earlier run of the same check is set aside only when it finished at or before a later run started, or while a later run is waiting to start; runs whose windows overlap are concurrent and are all judged, so matrix legs sharing a name cannot hide a failing leg. A run with no start time sets others aside only while it is unfinished; a finished one is placed at its completion, and one with neither time sets nothing aside. Runs that cannot be ordered are all kept. A commit status, which carries no completion time, keeps the newest-by-time rule per context. TestChecksJudgeEachCheckByItsNewestRun holds it."
impact: fix
---

While the person is away, the stop hook reports a pull request as failing its checks after the failed check was rerun and passed. The forge's status rollup keeps both runs of a check, the superseded failure and the newer success, and the hook reads the failure. It then tells the session to fix checks that already pass, and every reply after that carries the false item. Seen on 27/11: an attribution check failed on a pull-request body, the body was edited, the rerun passed, and the hook went on naming the pull request as failing.

Rerunning the failed run until it passes clears the false item, which shows the hook reads every run in the rollup rather than the newest one.
