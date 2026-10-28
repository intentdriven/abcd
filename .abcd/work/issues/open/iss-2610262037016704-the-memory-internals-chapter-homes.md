---
schema_version: 1
id: "iss-2610262037016704"
slug: "the-memory-internals-chapter-homes"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 692 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/07-memory.md:14"
remedy: "none (filed automatically)"
finding_ref: cfk-4c3c52a11a36d0b8
---

The memory internals chapter homes keyword-recalled personal and project memory in ~/.abcd.noindex/memory/ and .abcd/memory/; the memory verb homes the same in two memory.json layers.

A scope leakage (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/07-memory.md:14` — "| **user** | `~/.abcd.noindex/memory/` | **Personal preferences** and cross-project principles that have no single repo home"
- End B: `.abcd/development/brief/04-surfaces/37-memory.md:26` — "The machine's `~/.abcd.noindex/memory.json` holds the person's own notes; a repository's `.abcd/memory.json` holds the notes a person promoted into the project."

07-memory.md puts personal preferences and lessons that apply to every project in the user scope `~/.abcd.noindex/memory/`, with project knowledge in `.abcd/memory/`, curated by principle-distiller and injected by the prompt hook through each page's recall keywords. 37-memory.md puts the person's own lessons in `~/.abcd.noindex/memory.json` and promoted project notes in `.abcd/memory.json`, written by agents with recall words for the same prompt hook. Two stores, with different writers and different precedence, claim the same ground.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 692.
