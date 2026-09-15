---
schema_version: 1
id: "iss-2609081940475662"
slug: "site-manifest-page-sources-accept-git-config"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bughunt-triage"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/manifest.go"
resolution: "home.hero.page and home.chapters[].page are now closed to docs/ and site-src/, and every manifest path field refuses the git directory through the shared fsutil.InsideGitDir gate; .abcd/site.json is code-owned."
impact: fix
---

manifest.validate holds home.hero.page and home.chapters[].page to fsutil.ValidRelPath only, and that function accepts .git/config; loadPage then composes whatever the named file holds into index.html through ReadGuardedInRoot, which contains the read to the repo root but does not constrain the selection. The image pipeline documents and tests this exact threat (assetRootPrefix, TestAssetsRefuseAnythingOutsideTheAssetRoot) and positioning already refuses .git surfaces (iss-150), but the site composer did not copy the gate. identity.file and policy.file are not whole-file page sources (heading extract, one-bullet quote). A gitignored .env cannot merge because CI has no file and site-render goes red, and persist-credentials: false keeps the CI .git/config token-free, so the live exposure is bounded; the missing gate is not. .abcd/site.json is not in CODEOWNERS. Fix: refuse page sources under .git/ (case-folded, as the positioning denylist does) and restrict hero.page and chapters[].page to docs/ or site-src/, keeping identity.file and policy.file on their legitimate locations while applying the .git refusal; add .abcd/site.json to CODEOWNERS. Detector: LoadManifest with home.hero.page set to .git/config, to .env, and to a path outside docs//site-src/ must each fail validation, while the committed manifest still loads. Twin of iss-150. Reported as GitHub issue 622 against ec7f40d6.

## Grounds

- pursued: the composer's whole-file page sources need the closed-set gate the asset pipeline already has, because a denylist of .git and .env still leaves the local tier and every future file reachable; a page source that legitimately lives outside docs/ or site-src/ would show it wrong.
