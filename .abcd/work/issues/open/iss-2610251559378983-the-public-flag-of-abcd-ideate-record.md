---
schema_version: 1
id: "iss-2610251559378983"
slug: "the-public-flag-of-abcd-ideate-record"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D5"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/ideate.go"
remedy: "Change the flag text in internal/surface/cli/ideate.go:93 to 'the idea is public: write the verdict into the committed research store, under .abcd/development/research/ideas/<root-sha>/, and record a decision pointing at it (exclusive with --private)', then run go generate ./internal/surface/cli."
---

The --public flag of abcd ideate record (internal/surface/cli/ideate.go:92-93; commands.md:2096) reads 'the idea is public: commit the verdict under .abcd/development/research/ideas/<root-sha>/ and record a decision pointing at it'. The verb writes the record and a decision record (internal/core/ideate/record.go:176-225) and makes no git commit: neither internal/core/ideate nor internal/core/decisions runs git beyond resolving the checkout root. Read literally, the help promises a commit the verb never makes, and the person is left with uncommitted files.
