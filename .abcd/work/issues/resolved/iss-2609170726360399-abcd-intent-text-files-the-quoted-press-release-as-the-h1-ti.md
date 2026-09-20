---
schema_version: 1
id: "iss-2609170726360399"
slug: "abcd-intent-text-files-the-quoted-press-release-as-the-h1-ti"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "Gropius managed-repo session gropiusllm-56, relayed to abcd-17 on 2026-09-17"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/create.go"
resolution: "The quoted text now seeds the Press Release section as prose, the H1 is the text's first sentence cut at the slug cap, a new --title flag overrides it, and Why This Matters takes a prompt instead of the text repeated. The promote path is unchanged."
impact: fix
resolved_by:
  commit: "1c8520cd6b83263e7485397b3a9d261f6e5cb3a0"
---

abcd intent "<text>" files the quoted press release as the H1 title and leaves the Press Release section a placeholder stub. Reproduced at v0.9.0 (4ae6f221) in a scratch managed repo: abcd intent "Teams see who is waiting on whom without asking. Every session prints its owed answer." wrote a draft whose H1 is the whole quoted paragraph, whose Why This Matters section repeats it verbatim, and whose Press Release section holds only the seeded-from-quoted-text placeholder. createIntentFromText hands the text to titleLine, which collapses it into one heading line, and the slug truncates it at sixty characters, so a paragraph-length press release becomes an unreadable title and a stub narrative. Relayed from the Gropius managed-repo session gropiusllm-56 on 2026-09-17: a record-discipline reviewer there flagged every fresh draft as not press-release-first, because the one section the INTENTS rule says comes first is the one the verb leaves empty. Wanted: the quoted text seeds the Press Release section, and the title is derived from it (first sentence, or an explicit --title). The surface page only says the draft is seeded from the text, so it neither promises nor forbids either placement; a fix should say which.

**Corroboration (2026-09-20, Gropius session gropiusllm-97, relayed to
abcd-17).** A second record-discipline reviewer, on a different draft, flagged
the same shape: the H1 is the whole quoted paragraph and Why This Matters is
that paragraph pasted again, judged against the one-line H1s every shipped
intent carries. Two independent reviewers now read a fresh quoted-text draft
as malformed on sight.

## Grounds

- pursued: a fresh quoted-text draft is expected to read as press-release-first on sight, with a one-line H1 like every shipped intent; a record-discipline reviewer still flagging a quoted-text draft as malformed, or a promoted draft losing its seed note or by-id pointer, would show it wrong
