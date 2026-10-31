---
schema_version: 1
id: "iss-2610311242266907"
slug: "once-a-security-advisory-s-fix-has"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "post-cut advisory publication 2026-10-31"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/embargo.md"
remedy: "Before embargo publish, summarise each advisory into its published form (what is affected, the impact, the fixed version and the workaround, with no file:line references or reproduction steps), and run the summary through the existing toolchain: the secret scanner, both banlist layers, the home-path and hostname redaction, abcd lint outbound and embargo check. The plan's digest covers the summarised text, and a finding refuses the plan. The same summary feeds the ledger record a publish restores, so the record and the advisory say the same thing."
---

Once a security advisory's fix has shipped, its text goes public exactly as it was drafted. A draft is written while the finding is being worked, so it can carry detail that a published advisory doesn't need: source paths and line numbers, reproduction steps, local paths, hostnames or member logins. embargo publish already leaves out an advisory whose text carries a secret, a private name or another embargoed id. It does not summarise the text first, and it does not run the wider privacy checks over it.
