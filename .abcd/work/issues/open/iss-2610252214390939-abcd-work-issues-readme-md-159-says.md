---
schema_version: 1
id: "iss-2610252214390939"
slug: "abcd-work-issues-readme-md-159-says"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S07"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/issues/README.md"
remedy: "Say exit 2, or replace the README's verb paragraph (:145-160) with a pointer to commands/capture.md."
---

.abcd/work/issues/README.md:159 says resolving without --impact 'exits 1'; it exits 2 (internal/core/capture/workflow.go:373-376 refuses with ErrRequestRefused, mapped to code 2 at internal/surface/cli/cli.go:5989), as 06-capture.md:296-302 and commands/capture.md:408-411 say.
