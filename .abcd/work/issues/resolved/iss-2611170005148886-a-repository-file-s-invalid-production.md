---
schema_version: 1
id: "iss-2611170005148886"
slug: "a-repository-file-s-invalid-production"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "security review of the iss-2611162118486104 fix"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/provenance/declared.go"
remedy: "Have each reader of a repository file say the file is malformed and name the closed set without quoting the value, keeping ParseMode's quoting for a value the person typed on the command line; a test with a long prose value in each file shows the value stays out."
resolution: "The three readers of a repository file's production_mode, DeclaredMode's legacy-pin reading, ReadDeclaredModeFile and the identity migration's pin reader, refuse an out-of-set value through provenance.MalformedModeFile, which names the file and the closed set and leaves the value out; ParseMode's quoting stays for a value the person typed. TestAMalformedModeFileLeavesTheValueOut and TestMigration_AMalformedPinLeavesTheValueOut hold it."
impact: fix
---

A repository file's invalid production_mode value is quoted word for word into the refusals abcd shows the person and the agent: provenance.ParseMode's error quotes the value, and the three readers of in-tree files (the legacy identity pin in identity/migrate.go and provenance/declared.go's DeclaredMode, and the declared-mode file in ReadDeclaredModeFile) wrap it, so a hostile clone can put up to the read cap of chosen prose into an install note, escaped but read by the agent as written.

## Grounds

- pursued: no refusal abcd shows carries a value a repository file chose for production_mode, while it still names the file and the values it accepts; a refusal the person cannot act on would show it wrong
