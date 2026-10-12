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
remedy: "Decide, with the technical facilitator, between masking repository and tool text with termsafe where the loop stores it (keeping the run readable) and refusing a forged state.json at read time; then apply the choice to every field listed, with tests that forge each field and expect nothing raw on implement status --json and implement record --json."
---

A run's state.json stores text that legitimately carries repository or tool content, unchecked at read time and echoed on abcd implement status --json and implement record --json: spec step titles (stored raw from the spec), the pick text and excluded reasons, record notes quoting step titles and git errors, kept (git error text), sync paths (repository file names), the landing's forge URL, the runner's reported model and transcripts; ids, shas, stages, branch and worktree names, paths, verdict words and audit fields are unchecked too, though abcd writes clean values there. Refusing at read time would refuse runs abcd itself wrote (a spec in a right-to-left language, a coloured hook failure), so the fix needs a design choice: mask such text where it is stored, or refuse at read time and accept false refusals.

Security scope (abcd-fixes-what-it-adds): the route does not exist without abcd. The text is echoed by abcd's own --json surface, so abcd added the route and fixes it at the boundary the text enters by; a fix changes nothing about how the repository works.
