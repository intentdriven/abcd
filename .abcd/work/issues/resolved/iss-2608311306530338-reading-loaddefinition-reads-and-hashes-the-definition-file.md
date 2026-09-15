---
schema_version: 1
id: "iss-2608311306530338"
slug: "reading-loaddefinition-reads-and-hashes-the-definition-file"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "LoadDefinition reads through an os.Root at the repository root, so the hash it reports as the definition half of an instrument identity names a file this repository actually holds."
impact: fix
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

reading.LoadDefinition reads and hashes the definition file through a plain joined path rather than through the repository root, so a symlinked ancestor under agents/ can serve a definition from outside the repository. The read is harmless in itself, but the hash it returns is the definition half of an instrument identity sold as proving two runs read under the same instructions, and hashing a file that may not be in this repository makes that a claim about an unknown artefact.

## Grounds

- pursued: an instrument identity is a claim two runs are compared by, and a claim about an unknown artefact is worse than no claim; a definition hashed from outside the repository would show this wrong
