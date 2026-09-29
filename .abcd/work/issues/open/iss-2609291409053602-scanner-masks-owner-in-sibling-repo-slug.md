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
---

The scanner still masks the repository owner's public GitHub handle when it names a sibling repository of the same owner, e.g. <owner>/<sibling> written in a record of <owner>/<repo>. isOwnRepoSlug (internal/adapter/scanner/identity.go) exempts the owner only in the repository's own owner/repo slug, so a managed repo that publishes to a sibling repo loses the handle in every record naming it, and each one is restored by hand. Extend the exemption to the remote's owner followed by '/<any repo name>', since that owner is public by the same argument; the owner alone and other owners stay masked. Reported from a managed repository that publishes to a sibling repository of the same owner.
