---
id: dcn-2607140000000018
slug: adopted-default-to-the-local-tier-when
date: 2026-07-14
---
2026-07-14 (layout): ADOPTED "default to the local tier when in doubt". An artefact
whose home is unclear (tool exports, oracle/review output, traces, intermediate
analysis) goes to .abcd/.work.local/scratch/ or logs/ FIRST — never the repo root,
never a tracked directory on a guess. Promotion to .abcd/work/ or
.abcd/development/ is cheap and always available later; demotion is not, because a
wrongly-committed artefact is already in the history. Guessing upward is
irreversible; guessing downward costs nothing. Prompted by a RepoPrompt oracle export
landing in a top-level prompt-exports/ during this session (moved to
.abcd/.work.local/scratch/prompt-exports/). Recorded in AGENTS.md § Working-tree
layout, the canonical home, rather than a new doc.
