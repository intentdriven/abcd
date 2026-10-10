---
schema_version: 1
id: "iss-2610101610446956"
slug: "abcd-intent-ready-reported-itd-82-ready-while-abcd-build"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "build of itd-82, 2026-10-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/build.go"
remedy: "make intent ready run the open-questions check that build runs, so READY means build will start"
---

abcd intent ready reported itd-82 READY while abcd build itd-82 refused it at check (open_questions) for two open questions in its record: the readiness verb does not run the check the build makes, so READY does not mean the build will start.
