---
schema_version: 1
id: "iss-2610141706063703"
slug: "docs-how-to-install-md-also-the-0-14-1"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "v0.14.1 release review, docs-currency-reviewer (Opus), content commit 74165cf97"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/how-to/install.md"
remedy: "Add: provided GitHub API names the release commit; when it cannot, nothing is installed whether or not gh is signed in (install.md and README)."
---

docs/how-to/install.md (also the 0.14.1 changelog and README) says a gh that is not signed in installs on the checksum alone with a warning; in every path a gh with --source-digest first resolves the release commit through the unauthenticated GitHub API and refuses when that lookup fails, before sign-in is tested, so a rate limit or outage gives a refusal, not the checksum-only install (fails closed).
