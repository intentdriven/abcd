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
resolution: "A project site on GitHub Pages now carries its content policy in a meta element in each page, generated from the declared headers, and emits no Cloudflare files. The protections a meta policy cannot carry (framing refusal, nosniff) are documented as lost on Pages. Landed with step 1 of spc-2610142146250176 in #1344."
impact: fix
resolved_by:
  spec: "spc-2610142146250176"
  commit: "b49f294f7b3f3e4dd74c96d9b901527a9ba5d68a"
---

On the github-pages hosting provider the site loses the security headers abcd's own site serves: the build copies site-src/headers to _headers, which Cloudflare reads and GitHub Pages ignores, so a project site on GitHub Pages is served without the Content-Security-Policy and the other headers that file declares.

## Grounds

- pursued: each rendered project page carries the policy element before any link and the site check fails a page without it; a project page served with no policy would show it wrong
