---
schema_version: 1
id: "iss-2608241659573856"
slug: "the-org-rename-breaks-the-plugin"
severity: "critical"
category: "bug"
source: "agent-finding"
found_during: "v0.6.4 release validation 2026-08-24"
found_at: "hooks/bootstrap.sh"
resolution: "hooks/bootstrap.sh, hooks/hooks.json, the plugin manifest and the issue-template config name the current organisation, so the redirect from releases/latest is a tag URL again and resolved_tag parses. The bootstrap test's pinned origin constants move with it."
impact: fix
---

the org rename breaks the plugin bootstrap's tag resolution, so a fresh install refuses with a message blaming the network. hooks/bootstrap.sh pins repo_url and api_url to the old organisation. It resolves the latest release by reading (not following) the redirect from $releases_url/latest and parsing a tag out of it with sed 's|.*/releases/tag/...|'. After the rename that URL returns 301 to https://github.com/<new-org>/abcd/releases/latest — the org hop, NOT a tag URL — so the sed matches nothing and resolved_tag is empty. Verified live 2026-08-24: old org gives redirect_url=.../abcd/releases/latest (301), new org gives .../releases/tag/v0.6.3 (302). Consequences: with a cached binary the script takes use_cache=yes with cache_trust=offline and provisions from an unauthenticated cache while claiming offline, which is wrong about the cause but degrades loudly; with NO cache the download path hits its guard '[ -n "$resolved_tag" ] || refuse the latest release tag could not be resolved ... there may be no network' and refuses, so every fresh plugin install fails and the message misdirects to the network. bootstrap.sh ships from the repository tree rather than a release asset, so merging the corrected URLs to the default branch fixes installs without waiting for a release cut.