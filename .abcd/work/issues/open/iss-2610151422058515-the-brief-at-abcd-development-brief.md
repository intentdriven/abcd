---
schema_version: 1
id: "iss-2610151422058515"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/03-embark.md"
remedy: "Rewrite .abcd/development/brief/04-surfaces/03-embark.md:17 to match the shipped surface: Three embark sub-verbs ship: `abcd embark --help` lists from, lessons and probe; the chapter's own Sub-verbs table (line 42) and generated appendix list `lessons` as shipped too. The callout's enumeration omits `embark lessons`."
---

The brief at .abcd/development/brief/04-surfaces/03-embark.md:17 says: "What ships is the unpack and the probe" — the shipped-surface callout enumerates two sub-verbs. The shipped surface differs: Three embark sub-verbs ship: `abcd embark --help` lists from, lessons and probe; the chapter's own Sub-verbs table (line 42) and generated appendix list `lessons` as shipped too. The callout's enumeration omits `embark lessons`. (v0.14.2 release crosscheck, false-claim, raised by opus in a02.)
