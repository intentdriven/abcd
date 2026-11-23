---
schema_version: 1
id: "iss-2611060809597822"
slug: "abcd-development-brief-04-surfaces"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/17-guard.md"
remedy: "Add file writes through the host's own tools to the sentence."
resolution: "The guard chapter's sentence on what the shim's UNGUARDED warning names now lists file writes through the host's own tools beside shell commands and questions through the host's question tool, matching the three wordings hooks/pre-tool-use.sh prints."
impact: internal
resolved_by:
  commit: "ba958350e16b1df00a46cba49a9d7f97b7703ce2"
---

`.abcd/development/brief/04-surfaces/17-guard.md:459-460` says the shim's `UNGUARDED` warning names shell commands or questions through the host's question tool. `hooks/pre-tool-use.sh:4` has a third wording for the host's file-writing tools, `file writes through the host's own tools run UNGUARDED`.

## Grounds

- pursued: the sentence in 17-guard.md's fail-open-loud section was checked against the three UNGUARDED wordings hooks/pre-tool-use.sh prints (shell commands, questions, file writes through the host's own tools), and it now names all three; what would show it wrong is the script printing a wording this sentence does not list, or the sentence naming a case the script does not print.
