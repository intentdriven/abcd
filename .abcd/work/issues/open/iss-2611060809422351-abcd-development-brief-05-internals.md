---
schema_version: 1
id: "iss-2611060809422351"
slug: "abcd-development-brief-05-internals"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Correct the passage to say what the code does."
---

`.abcd/development/brief/05-internals/03-configuration.md:1629`: The prompt router's documented `--json` envelope and stderr contract omit fields and a line the binary emits. The envelope also carries fill, bracket and system_message (internal/surface/cli/cli.go:1737-1752; run output included "fill":{"source":"unknown",...} and "bracket":"unknown"), none in the table. recalled also holds page keys (run: "page:fact_eng_repo-page.md"). stderr also carries a once-a-session line when the fill is unknown (run: 'abcd rules: how full this session is, is unknown ...').
