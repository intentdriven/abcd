---
schema_version: 1
id: "iss-2610142036270003"
slug: "on-the-github-pages-hosting-provider-the"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "itd-2610091010348892 build (github-pages provider)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/hosting/githubpages/githubpages.go"
remedy: "Carry the declared headers a Pages host can honour into the rendered pages (a CSP meta element per page, generated from site-src/headers), or document that Pages serves none of them and drop _headers from a Pages build."
---

On the github-pages hosting provider the site loses the security headers abcd's own site serves: the build copies site-src/headers to _headers, which Cloudflare reads and GitHub Pages ignores, so a project site on GitHub Pages is served without the Content-Security-Policy and the other headers that file declares.
