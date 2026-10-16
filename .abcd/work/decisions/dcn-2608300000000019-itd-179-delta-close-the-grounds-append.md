---
id: dcn-2608300000000019
slug: itd-179-delta-close-the-grounds-append
date: 2026-08-30
---
2026-08-30 — itd-179 delta close: the grounds APPEND and its read-back run over the record BODY, spliced back onto the frontmatter, rather than over the whole file. `frontmatter.Split` is the one lossless answer to where the block stops and `grounds.Body` is what both halves ask, so the writer judges the bytes the reader consults and the writer/reader disagreement is unrepresentable rather than banned one spelling at a time (iss-2608301805069999). The intent half's reader moves to the same scope, where the whole-file reading would have found an empty pseudo-section. Rejected: refusing a frontmatter `# Grounds` comment, which bans the one spelling that was reported and leaves the scope split that produced it.
