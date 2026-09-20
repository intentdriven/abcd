---
schema_version: 1
id: "iss-2609181020519987"
slug: "the-intent-surface-page-does-not-say-whether-abcd-intent-rea"
severity: "nitpick"
category: "documentation"
source: "agent-observation"
found_during: "Gropius managed-repo session gropiusllm-56, relayed to abcd-17 on 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/intent.md"
resolution: "The Grounds section of commands/intent.md now states that grounds are recorded on a draft or a planned intent alike, and refused on a shipped or superseded one — the bucket rule RecordGrounds enforces."
impact: fix
resolved_by:
  commit: "3e9da5b4d3a8ee720150fdc7338d93f40bae9edc"
---

The intent surface page does not say whether abcd intent ready --grounds records on a draft. The Grounds paragraph of commands/intent.md describes the flag on the readiness gate and never states which buckets accept the write; the code (RecordGrounds) refuses only a shipped or superseded record, so a draft in drafts/ takes grounds before it is planned, and the resolved iss-2608300930057882 states that forward-only exemption in its resolution. Relayed from the Gropius managed-repo session gropiusllm-56 on 2026-09-18, which recorded grounds on a draft, found it worked, and could not tell from the page whether that was sanctioned. One sentence on the page closes it: grounds are recorded on a draft or a planned intent alike, and refused on a shipped or superseded one.

## Grounds

- pursued: a session reading the page is expected to know before writing that grounds on a draft are sanctioned; a reader still asking whether a draft takes grounds, or the sentence disagreeing with what RecordGrounds refuses, would show it wrong
