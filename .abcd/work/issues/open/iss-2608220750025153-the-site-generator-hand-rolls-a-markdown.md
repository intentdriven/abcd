---
schema_version: 1
id: "iss-2608220750025153"
slug: "the-site-generator-hand-rolls-a-markdown"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "agent-observation"
found_at: "internal/core/site"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed C): Sign off goldmark and x/net/html as the site generator's dependencies?"
remedy: "Waits on ruling C (sign off goldmark and x/net/html): if signed off, adopt goldmark alone for Markdown behind the site package's render seam with raw HTML left off, proven by the site package's existing tests passing unchanged plus the CommonMark spec examples for the constructs the site uses, and keep the hand-written strict tokenizer rather than x/net/html; if refused, record the subset renderer as the decision in an ADR line and add those CommonMark examples as a conformance test of the subset."
---

the site generator hand-rolls a Markdown-subset renderer, a strict HTML tokenizer and a CSL formatter to stay dependency-free; adopting goldmark and x/net/html instead is a maintainer dependency decision

## Remedy grounds (2026-09-29)

- goldmark states CommonMark 0.31.2 compliance, depends only on the standard library and renders no raw HTML or dangerous links by default (https://github.com/yuin/goldmark, consulted 2026-09-29). The Go vulnerability database lists sixteen reports against golang.org/x/net/html, seven of them in 2026, chiefly parser denial of service and render-tree XSS (https://pkg.go.dev/search?q=golang.org%2Fx%2Fnet%2Fhtml&m=vuln, consulted 2026-09-29), while the generator only tokenises repository content strictly, so the full HTML5 parser buys little and adds that surface.
- Rejected: taking both dependencies as one decision; they carry different cost and different benefit.
