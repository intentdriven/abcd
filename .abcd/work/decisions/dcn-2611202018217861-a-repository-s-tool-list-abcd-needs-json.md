---
id: dcn-2611202018217861
slug: a-repository-s-tool-list-abcd-needs-json
date: 2026-11-20
---
2026-11-20 — A repository's tool list (.abcd/needs.json) holds, under schema_version 1, exactly three fields per tool: name, reason and version_file, and nothing else (itd-2611041815429986, step 1). Grounds: adr-2611041815445203 rule 2 fixes the list as a tool's name, why it is needed and which file declares its version, and no path, checksum or version of its own; the needs-step1 Opus review accepted the three fields as an exact match to that rule, and making version_file required for Go and forbidden for a tool with no version set keeps the closed set explicit. Refs itd-2611041815429986.
