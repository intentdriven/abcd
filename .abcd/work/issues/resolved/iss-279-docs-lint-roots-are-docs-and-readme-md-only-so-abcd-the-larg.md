---
schema_version: 1
id: "iss-279"
slug: "docs-lint-roots-are-docs-and-readme-md-only-so-abcd-the-larg"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "manual-capture"
found_at: ".abcd/docs-lint.json"
resolution: "docs-lint name_roots carry the names/ banned tokens over .abcd, AGENTS.md, CONTRIBUTING.md and scripts, every text file, with a coverage test."
impact: fix
resolved_by:
  commit: "dd43ce1f"
---

docs-lint roots are docs/ and README.md only, so .abcd/** — the largest public surface — is scanned by no name gate; retire-the-name's banned_tokens cannot reach CONTRIBUTING.md, AGENTS.md, or scripts/ either

## Grounds

- pursued: a names/ ban placed in AGENTS.md, a script, or a record under .abcd is reported by docs lint while non-name tokens stay confined to the roots; a name ban passing in one of those trees would show it wrong
