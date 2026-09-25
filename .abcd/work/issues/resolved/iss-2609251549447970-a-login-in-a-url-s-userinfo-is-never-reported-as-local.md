---
schema_version: 1
id: "iss-2609251549447970"
slug: "a-login-in-a-url-s-userinfo-is-never-reported-as-local"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/identity.go"
resolution: "Each URL span carries its userinfo (urlUserinfo: the bytes between scheme:// and the authority's last '@'), and the username matcher's URL suppression stops at it (urlSet.suppressing); a generic login inside it stands as an account name (urlSet.inUserinfo). The forge's ssh service account keeps its suppression. TestLocalUsernameReportedInAURLUserinfo was watched failing on five userinfo shapes, named and generic, with and without a password, before the change and passes after."
impact: fix
resolved_by:
  commit: "077b3d75"
---

A login in a URL's userinfo is never reported as local_username, for any login. localSuppressionSpans (internal/adapter/scanner/identity.go) suppresses the username matcher inside every URL span, so https://<login>@host/ and https://<login>:<password>@host/ pass the scanner with the caller's login intact; the userinfo is the one part of a URL that names an account rather than a resource, and a clone URL or a proxy setting quoted in a transcript carries it. Pre-existing: the suppression predates the scanner-cluster lane, whose review found it (item 6).

## Grounds

- pursued: a login in a URL's userinfo is reported as local_username while the rest of the URL stays suppressed; a scheme://<login>@host URL passing ScanText with no finding, or a URL path segment being flagged, would show it wrong.
