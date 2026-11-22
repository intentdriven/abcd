---
schema_version: 1
id: "iss-2611060809343029"
slug: "abcd-development-brief-05-internals"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/05-prompt-quality.md"
remedy: "Reword line 101 to the later statement: no shipped prompt has a research file it can find by name."
resolution: "The prompt-quality chapter's SOTA-audit section no longer says research files do not exist for any shipped agent; it now says, as the chapter's research-driven section does, that no shipped prompt has a research file it can find by name. A lint test holds the section to that."
impact: internal
resolved_by:
  commit: "5dbd319688b58a313376921511ace2a5c8be2e60"
---

`.abcd/development/brief/05-internals/05-prompt-quality.md:101` says research files do not exist for any shipped agent. `.abcd/development/research/prompting/agents/intent-fidelity-reviewer.md` is research for the shipped intent auditor (`.abcd/development/research/prompting/agents/README.md:44`), and :151-155 of the same chapter says it is stranded under the old name, so that no shipped prompt has a research file it can find.

## Grounds

- pursued: the SOTA-audit section and the research-driven section of 05-prompt-quality.md agree that the intent auditor's research file exists but is stranded under its old name; TestBriefSOTAAuditAgreesResearchFilesAreStranded passing while the section still says the files do not exist, or a by-name lookup that finds a research file for a shipped prompt, would show it wrong
