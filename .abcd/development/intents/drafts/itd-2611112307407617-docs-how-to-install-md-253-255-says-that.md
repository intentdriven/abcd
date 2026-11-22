---
id: itd-2611112307407617
slug: docs-how-to-install-md-253-255-says-that
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610251557502327]
origin: extracted-from-record
production_mode: hand-written
---

# docs/how-to/install.md:253-255 says that while ~/.abcd stands no tool runs at all in an agent session, because the safety check blocks every command, the rename included. That holds only while the plugin root is empty, the state a plugin update leaves, since hooks/pre-tool-use.sh exits 2 before any bootstrap. With a binary in the plugin root, abcd guard hook admits the exact rename command (internal/surface/cli/homestop.go:102-109, :143-160; internal/abcdhome/abcdhome.go:45-48), and the hook gates only the shell and question tools (hooks/hooks.json:29), so other tools still run.

## Press Release

> _Seeded by promotion from iss-2610251557502327. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610251557502327`: docs/how-to/install.md:253-255 says that while ~/.abcd stands no tool runs at all in an agent session, because the safety check blocks every command, the rename included. That holds only while the plugin root is empty, the state a plugin update leaves, since hooks/pre-tool-use.sh exits 2 before any bootstrap. With a binary in the plugin root, abcd guard hook admits the exact rename command (internal/surface/cli/homestop.go:102-109, :143-160; internal/abcdhome/abcdhome.go:45-48), and the hook gates only the shell and question tools (hooks/hooks.json:29), so other tools still run.. Read that issue record for the source observation.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
