---
schema_version: 1
id: "iss-2610252214522207"
slug: "the-generated-cli-reference-s-library"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S09"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/surface/examples.go"
remedy: "Change the example to an https HTML or Markdown address and regenerate docs/reference/cli/commands.md."
---

The generated CLI reference's library ingest example is https://example.com/paper.pdf (internal/core/surface/examples.go:111), a PDF the binary rejects with no extractor wired (internal/core/memory/ingest.go:985-987; 07-library.md:355-356, commands/library.md:297-299).
