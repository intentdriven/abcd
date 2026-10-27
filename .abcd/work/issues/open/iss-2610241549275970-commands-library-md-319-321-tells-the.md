---
schema_version: 1
id: "iss-2610241549275970"
slug: "commands-library-md-319-321-tells-the"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "hand-back triage of 2026-10-24, batch b03 reading iss-2609211905347458"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/library.md"
remedy: "Reword commands/library.md:319-321, .abcd/development/brief/05-internals/07-memory.md:118 and the Memory ingest row at .abcd/development/brief/06-delivery/02-verification-matrix.md:86 to say a kept original stays in the checkout, launch never publishes .abcd/**, and no lifeboat gate exists yet because disembark packs no memory (iss-2609211905347458), as 04-surfaces/07-library.md:358-363 already says; proven by docs-lint and the docs-currency review."
---

commands/library.md:319-321 tells the agent that the lifeboat licence gate governs a kept original's export. The brief's 05-internals/07-memory.md:118 and the Memory ingest row of 06-delivery/02-verification-matrix.md:86 say that gate refuses to publish one without an allowlist entry. No such gate exists: disembark packs no memory (internal/core/lifeboat/plan.go copies only ADRs, issues, specs, retrospectives and intents), as the brief's own 04-surfaces/07-library.md:358-363 says.
