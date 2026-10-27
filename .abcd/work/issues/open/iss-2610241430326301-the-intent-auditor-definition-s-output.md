---
schema_version: 1
id: "iss-2610241430326301"
slug: "the-intent-auditor-definition-s-output"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "owed fidelity-review drain, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "agents/intent-auditor.md"
remedy: "Make the file route read what the auditor is told to emit: have intent audit ingest and intent consistency ingest accept a file holding exactly one fenced json block, through one shared unfence helper moved out of the provider adapter, and fail the receipt closed on zero or two blocks as rule 1 says, keeping bare JSON accepted; or, if bare JSON stays the only input, reword rule 1 and the two Save steps in commands/intent.md to say the fence lines are dropped. Prove it with ingest tests over a fenced verdict, a bare verdict and a two-block reply. Grounds: internal, the auditor contract and the provider adapter's existing unfence."
---

The intent-auditor definition's output rule 1 (agents/intent-auditor.md) says a reply with zero or two fenced json blocks makes the ingest fail the receipt closed, every criterion recorded INCONCLUSIVE, but intent audit ingest --verdict-json decodes the file as bare JSON (internal/core/intent/audit.go, IngestVerdictBytes) and refuses a fenced reply as not parseable JSON, writing nothing, and commands/intent.md tells the host to save the single JSON block without saying the fence lines must go. Only the provider route strips a fence (internal/adapter/openaiapi/client.go, unfence). A host that saves the auditor's reply as it came back is refused, and the fail-closed path the auditor is told about does not exist on the harness route. The consistency ingest decodes bare JSON the same way.
