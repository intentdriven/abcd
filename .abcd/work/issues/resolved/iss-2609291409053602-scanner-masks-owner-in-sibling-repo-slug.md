---
schema_version: 1
id: "iss-2609291409053602"
slug: "scanner-masks-owner-in-sibling-repo-slug"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "managed-repo report, 2026-09-29"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/identity.go"
remedy: "Extend the exemption to the remote's owner followed by '/<any repo name>', since that owner is public by the same argument; the owner alone and other owners stay masked."
resolution: "github_username spares the remote's owner as the owner half of any owner/repo slug on the forge (isOwnerSlug, replacing isOwnRepoSlug): '/' and a GitHub repository name of 1-100 bytes follow the owner, the owner is a whole owner segment, and a host written before it must be github.com or a subdomain. The owner alone, the owner with no name, other owners and a slug after another host stay findings. The same anchoring now holds for the repository's own slug, and the byte scan applies the rule identically to text."
impact: fix
resolved_by:
  commit: "674cfa152828f1de691d59fc1f1ea3ba7312c723"
---

The scanner still masks the repository owner's public GitHub handle when it names a sibling repository of the same owner, e.g. <owner>/<sibling> written in a record of <owner>/<repo>. isOwnRepoSlug (internal/adapter/scanner/identity.go) exempts the owner only in the repository's own owner/repo slug, so a managed repo that publishes to a sibling repo loses the handle in every record naming it, and each one is restored by hand. Extend the exemption to the remote's owner followed by '/<any repo name>', since that owner is public by the same argument; the owner alone and other owners stay masked. Reported from a managed repository that publishes to a sibling repository of the same owner.

## Grounds

- pursued: a record of <owner>/<repo> naming <owner>/<sibling> keeps the owner's handle through every scanner-backed redactor (TestGithubUsernameSparesTheOwnersSiblingSlug); a sibling slug still rewritten, or the owner alone, another owner, or a slug after a non-forge host spared (TestGithubUsernameSlugExemptionIsAnchored), would show it wrong
