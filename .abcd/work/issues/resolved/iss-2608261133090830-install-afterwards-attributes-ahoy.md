---
schema_version: 1
id: "iss-2608261133090830"
slug: "install-afterwards-attributes-ahoy"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-round-8"
found_at: "docs/how-to/install.md:119"
resolution: "the Afterwards sentence gets its abcd ahoy install subject back and the section states the one-liners take no options"
impact: fix
resolved_by:
  commit: "5599337a"
---

the install guide Afterwards section attributes a PATH warning and bin-dir option to the CLI one-liners, which parse no arguments and print no warning; the sentence lost its abcd ahoy install subject in a README rewrite