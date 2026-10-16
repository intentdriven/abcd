---
id: dcn-2610020000000017
slug: ruling-oc1-of-the-technical-facilitator
date: 2026-10-02
---
2026-10-02 — Ruling OC1 of the technical facilitator (2026-10-02, asked in plain words by abcd-a2 [75597f] after a research pass on the opencode harness; same local-tier file as the H3-0 entry above; recorded by lane rd2), verbatim: "SEAL IT, AND TEST IT — runner adds OPENCODE_DISABLE_PROJECT_CONFIG=1, OPENCODE_DISABLE_CLAUDE_CODE=1, OPENCODE_DISABLE_EXTERNAL_SKILLS=1 beside --pure; plus a hostile-fixture test (planted instruction, setting, skill) that fails if any is read." Sources: opencode's published documentation (config, cli, rules, plugins, skills) and its source, read 2026-10-02; the project-config switch is documented only in the source. Not built at this entry's base, where `internal/core/runner/opencode.go` passes --pure alone; lane runnerSeal builds it.
