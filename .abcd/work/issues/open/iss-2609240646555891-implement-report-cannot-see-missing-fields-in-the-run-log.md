---
schema_version: 1
id: "iss-2609240646555891"
slug: "implement-report-cannot-see-missing-fields-in-the-run-log"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/report.go"
---

`abcd implement report` derives its figures from fields no writer is required to supply, and says nothing when they are missing. It counts a lane as landed only when its lane_close carries `outcome` merged or landed, and agent minutes only from agent_end lines, while `abcd implement log` accepts any set of fields on any event. In autonomous run A the orchestrator that took over at the first rotation (16:15Z on 2026-09-23) wrote lane_close lines with `pr` and `merge` fields and no outcome, used lane_open and lane_close for review and fix rounds, and wrote no agent_start, agent_end, gate_run or ceiling_wait line for the rest of the run. Over both days the report counts ten lanes landed for the first session and one for the second, where thirty pull requests merged, and its agent minutes stop at 16:08Z on the first day. None of this is flagged, because every line parses. Wanted: per-event required fields (an outcome on lane_close; role, model and minutes on agent_end) refused at `implement log` time, and a report line naming any event kind whose coverage stops partway through the run.
