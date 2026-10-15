---
schema_version: 1
id: "iss-2610121542240257"
slug: "a-run-s-state-json-stores-text-that"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "the build loop's stored-text sweep on fix/sec-build-next-series, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/state.go"
remedy: "The technical facilitator ruled on 2026-10-12, in the person's words: \"both: mask when stored and when shown\". Neutralise repository and tool text with termsafe before the loop writes it to state.json (step titles, pick text and excluded reasons, record notes, kept, sync paths, the landing URL, the runner's reported model, transcripts), AND mask it at every render that echoes state.json text (implement status, implement record, build, drain, refusal reasons), as the other output paths do; a run abcd wrote stays readable, so no read-time refusal for these fields. Tests forge each field in a stored state and plant raw runes at each write site, expecting nothing raw on any --json stream."
resolution: "Per the ruling \"both: mask when stored and when shown\", State.masked passes the run's repository and tool text through termsafe.Sanitize where writeState stores it and where readStateIn returns it, so every surface shows it masked and nothing is refused at read time; the checkpoint kept reason, the pick's exclusion reasons and a transcript capture are masked where they are composed."
impact: fix
resolved_by:
  commit: "ffee62d99aa0f14396606ccc0ad5622d11c6f3d3"
---

A run's state.json stores text that legitimately carries repository or tool content, unchecked at read time and echoed on abcd implement status --json and implement record --json: spec step titles (stored raw from the spec), the pick text and excluded reasons, record notes quoting step titles and git errors, kept (git error text), sync paths (repository file names), the landing's forge URL, the runner's reported model and transcripts; ids, shas, stages, branch and worktree names, paths, verdict words and audit fields are unchecked too, though abcd writes clean values there. Refusing at read time would refuse runs abcd itself wrote (a spec in a right-to-left language, a coloured hook failure), so the fix needs a design choice: mask such text where it is stored, or refuse at read time and accept false refusals.

Security scope (abcd-fixes-what-it-adds): the route does not exist without abcd. The text is echoed by abcd's own --json surface, so abcd added the route and fixes it at the boundary the text enters by; a fix changes nothing about how the repository works.
