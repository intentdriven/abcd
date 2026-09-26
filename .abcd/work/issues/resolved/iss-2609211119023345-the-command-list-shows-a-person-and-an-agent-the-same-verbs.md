---
schema_version: 1
id: "iss-2609211119023345"
slug: "the-command-list-shows-a-person-and-an-agent-the-same-verbs"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "product thinker's interview for abcd build next and abcd drain, 2026-09-21"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli (the command list abcd --help renders); commands/*.md"
resolution: "Already fixed at base by 4bac5e5d (itd-146): abcd --help lists the person's verbs in five labelled groups, abcd --help --agent adds the block of verbs agents and hosts call, and each command page's block: frontmatter says which list it belongs to, held to the tree by a test."
impact: internal
resolved_by:
  intent: "itd-146"
  commit: "4bac5e5d6c9052d055275f5533c82c9ac16a771d"
---

The command list shows a person and an agent the same verbs. abcd --help and the plugin's command pages list every verb alike, but the verbs split by who types them: a person types abcd build <itd-N>, abcd build next, abcd drain, abcd intent, abcd capture; a driving host or an agent calls abcd implement step, abcd implement receipt, abcd reading assemble, abcd intent audit ingest and the other machinery verbs a person never types. A product thinker reading the list today cannot tell which half is theirs, and the ruling of 2026-09-21 that build is the verb for people and implement the loop for the machine (itd-2609201916151817, decision 8) needs a place to show. Wanted: the command list distinguishes the two, the person's verbs by default and the agent's behind a flag such as --agent (or a section), with each command page saying which it is; the product thinker recalls an existing record asking for this cleanup, so the first step is to find and link it rather than file twice.

## Grounds

- pursued: abcd --help at 6e4eca9a shows the person's groups and names --agent for the rest; a person-facing group listing implement, reading or the other machinery verbs would show it wrong
