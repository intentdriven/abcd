---
schema_version: 1
id: "iss-2608311504353427"
slug: "the-intent-audit-ingest-does-not-neutralise-markdown-link-sy"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "ingesting the itd-187 fidelity verdict"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/audit.go"
resolution: "termsafe.cleanProse, the canonical untrusted-prose cleaner, now breaks the bracket-then-parenthesis and bracket-then-bracket adjacencies (a single space, matching the comment-delimiter treatment) so a faithful quotation such as items[0](itm-0001) can no longer land as a live link; the audit ingest's oneLine now routes through termsafe.CleanProseLine instead of keeping a second sanitiser, and an ingest test proves the links_resolve gate passes over the record it writes."
impact: fix
resolved_by:
  commit: "19735f87"
---

The intent-audit ingest does not neutralise markdown link syntax in untrusted verdict text, so a verdict that quotes code containing a bracket-then-parenthesis sequence writes a live markdown link into the committed record and record-lint then refuses the whole tree with a links_resolve blocker. Hit for real: an auditor quoting an assembled-input element path of the form items[0](itm-0001) produced a link whose target itm-0001 resolves to nothing, and the gate failed on a record the ingest had just written. The oneLine sanitiser already neutralises the two shapes that matter for spoofing -- newlines, so injected content cannot break out of its line, and HTML comment delimiters, so untrusted text cannot forge a review marker -- and link syntax belongs in the same set for a different reason: not spoofing but a self-inflicted gate failure that no amount of care in the auditor can prevent, because the offending text is a faithful quotation of the code under audit. The repair available today is a hand edit of the rendered line, which is the same accepted repair as the grounds body-lockout.

## Grounds

- pursued: we expect a verdict quoting bracket-then-parenthesis code to ingest without tripping the links_resolve gate, and a record-lint blocker on a freshly ingested Audit Notes block would show it wrong
