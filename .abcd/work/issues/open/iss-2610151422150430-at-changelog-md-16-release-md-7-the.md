---
schema_version: 1
id: "iss-2610151422150430"
slug: "at-changelog-md-16-release-md-7-the"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "v0.14.2 release review, docs-currency pair, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: "CHANGELOG.md"
remedy: "Correct CHANGELOG.md:16; RELEASE.md:7 so it states what the shipped binary does, as the finding describes."
---

At CHANGELOG.md:16; RELEASE.md:7: The entry says that with `oracle.review = rp` in the repository's or the machine's configuration, a build lane sends its reviews to RepoPrompt. In the code (internal/core/oracle/review.go) the route also needs `oracle.rp.url`, and only ~/.abcd.noindex/config.json may set it: a repository that sets it is refused. With only `oracle.review = rp`, every review falls back to the host. commands/build.md and commands/implement.md state this correctly; the changelog line does not mention the URL. (v0.14.2 release docs-currency review, raised by opus.)
