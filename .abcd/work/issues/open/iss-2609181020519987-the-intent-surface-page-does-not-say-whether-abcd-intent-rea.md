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
---

The intent surface page does not say whether abcd intent ready --grounds records on a draft. The Grounds paragraph of commands/intent.md describes the flag on the readiness gate and never states which buckets accept the write; the code (RecordGrounds) refuses only a shipped or superseded record, so a draft in drafts/ takes grounds before it is planned, and the resolved iss-2608300930057882 states that forward-only exemption in its resolution. Relayed from the Gropius managed-repo session gropiusllm-56 on 2026-09-18, which recorded grounds on a draft, found it worked, and could not tell from the page whether that was sanctioned. One sentence on the page closes it: grounds are recorded on a draft or a planned intent alike, and refused on a shipped or superseded one.
