---
schema_version: 1
id: "iss-2609230851376499"
slug: "the-memory-lint-reports-an-ordinary-page-back-link-as-hostname-residue"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/memory/lint.go"
resolution: "The MR001 registry scan blanks every back-link that is a well-formed page name out of the registry text and judges it by the page-name rule alone, as the write side excludes it from the leaf walk, so an ordinary prose-shaped page name no longer lints as network residue"
impact: fix
resolved_by:
  commit: "6074755b80e538d2a35d881deee9cc7d375eb7c8"
---

The read-side MR001 lint reports an ordinary memory store as a blocker when a page name is prose-shaped. residueOfStoreFiles scans the sources registry bytes at the scanner.BlockingResidual bar, and the registry carries every page back-link, so a back-link such as topic_home_migrating-[redacted-hostname].md matches net_device_hostname at warn on the hyphen boundary and memory lint exits 1 with 'stored text carries a net:device_hostname span' on a store the write side produced without complaint. The write side already knows this: redactRegistryLeaves excludes the back-link list from the leaf walk for exactly this slug, and judgeFilename judges page names at the hard_fail bar alone. The read side holds a back-link to the free-text bar, so the two sides disagree on the same bytes. Reproduced at bad1c73e by ingesting a page with domain home and slug [redacted-hostname] and running Lint: one MR001 blocker on .sources_index.json. Remedy: judge back-link names by the page-name rule and keep them out of the registry's free-text scan.

## Grounds

- pursued: a back-link is an identifier the store resolves, so holding it to the filename bar on both sides makes the read side agree with the write side on the same bytes; an ordinary ingested store drawing MR001, or a secret in the registry's free-text fields going unreported, would show it wrong
