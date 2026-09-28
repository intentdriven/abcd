---
schema_version: 1
id: "iss-2609251645376219"
slug: "docs-reference-cli-readme-md-says-abcd-help-documents-itself"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
resolution: "docs/reference/cli/README.md says abcd --help lists the verbs a person types and abcd --help --agent adds the agents-and-hosts verbs"
impact: fix
resolved_by:
  commit: "07af2eb092552739a5360c39e5659457e7b3c734"
---

docs/reference/cli/README.md says abcd --help documents itself, but the default list now omits the agents-and-hosts verbs and nothing in the prose names --help --agent, so a person who finds abcd version in commands.md does not see it under abcd --help and is not told the list expands (review-helpgroups 1).

## Grounds

- pursued: a person who finds an agent verb in commands.md learns how to list it; a hidden verb the README does not account for would show it wrong
