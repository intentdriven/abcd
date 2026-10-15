---
schema_version: 1
id: "iss-2610151422155612"
slug: "at-changelog-md-5-24-judgement-call"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "v0.14.2 release review, docs-currency pair, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: "CHANGELOG.md"
remedy: "Correct CHANGELOG.md:5-24 so it states what the shipped binary does, as the finding describes."
---

At CHANGELOG.md:5-24: Judgement call under the derived-changelog convention, which gives partly delivered intents no changelog line. This binary ships user-visible verbs that the 0.14.2 section never mentions: `abcd embargo` (listed in the default root help under Release, with a full command page), `abcd reserve` and `abcd memory` (in --help --agent), `rules --prompt`, drain reservations, and consistency chunking with --max-chunks. Line 5 says itd-2610112239022766 (embargo) is 'targeted and not shipped', but `abcd embargo` is in the binary. It also reuses the name `abcd memory`: in v0.14.0 and v0.14.1 bare `abcd memory` refused with exit 2; it now exits 0 with a different payload, and the section has no line saying so (the preamble says a break is stated as an Added line). Only docs/how-to/upgrade-to-v0.14.0.md mentions the reuse. / The pages shipped in this range document verbs the changelog and release note never announce: abcd reserve, abcd memory (add/list/remove), abcd embargo, `abcd rules --prompt`, reservation fields on `abcd peers` and the bare board. They exist in the binary and the pages match it, so nothing misleads; the changelog is record-derived, so this follows from which records were… (v0.14.2 release docs-currency review, raised by opus+sonnet.)
