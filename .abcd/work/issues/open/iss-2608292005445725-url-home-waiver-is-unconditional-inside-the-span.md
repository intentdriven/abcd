---
schema_version: 1
id: "iss-2608292005445725"
slug: "url-home-waiver-is-unconditional-inside-the-span"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "internal/adapter/scanner/residual.go"
---

ultra-v0.6.8 follow-up (confirmation pass): homeSweepable in internal/adapter/scanner/residual.go waives the leading half of the home anchor unconditionally inside a URL span (if inAnySpan(at, urls) { return !nameContinues(text, end) }), so with HOME=/root (Docker, CI-as-root) ordinary URLs are corrupted and false hard-fails raised: https://docs.example.com/root/index.html is committed as https://docs.example.com~/index.html, and git@github.com:acme/root/tool.git and https://pkg.go.dev/example.com/mod/root scan as home_path_self hard_fail. Proposed narrowing: waive the leading anchor only when the match begins at the first '/' after the URL authority (scheme://host[:port]), so a home that is the URL's path root is swept and one buried deeper in the path is judged by the ordinary anchor.
