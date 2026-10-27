---
schema_version: 1
id: "iss-2610251558434724"
slug: "docs-reference-terminology-md-three"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D3"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/reference/terminology.md"
remedy: "Rule each position: Durable execution to ADAPTS (file-based checkpoint and resume) or keep WATCHING and name the record that tracks it; Agent2Agent to REJECTS on the principle that coordination lives on the shared remote, or file a watch issue and cite it; Orchestration reworded to say abcd's implement loop sequences a run's agents while the host session or a configured runner executes each. Edit the three rows and run abcd lint docs."
---

docs/reference/terminology.md: three crosswalk positions no longer fit their own label definitions once the stale facts are corrected, and need a ruling rather than an edit. Durable execution stays WATCHING although the pacing intent (itd-2609201925079472, shipped) checkpoints lanes to their branches and abcd build resumes a run from its state file (internal/core/implement/loop/ratelimit.go:3-20, loop.go:825; internal/surface/cli/build.go:103). Agent2Agent stays WATCHING although its tracking record itd-33 is superseded by itd-2609091034175565, so no live entry tracks it as the label requires (terminology.md:13-14). Orchestration says the orchestration-substrate role is declined while abcd's own implement loop sequences a run's agents and gates each stage on its receipt (internal/core/implement/loop/budget.go:11-13).
