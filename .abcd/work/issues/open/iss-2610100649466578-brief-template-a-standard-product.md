---
schema_version: 1
id: "iss-2610100649466578"
slug: "brief-template-a-standard-product"
severity: "minor"
category: "future-work-seed"
source: "managed-repo"
found_during: "abcd inbox report rpt-2610071653351920 from a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0)"
origin: researcher-authored
production_mode: hand-written
found_at: "brief template (01-product), disembark coverage pass, the vision refinement step"
remedy: "none (filed automatically)"
---

Brief template: a standard Product principles section beneath the press release, filled by the vision step

## What the lab found

A post-brief vision interview in a managed repository produced the owner's vision in their own words and six ranked trade-offs ("easy first minutes, even over every choice up front"; "playing together, even over the studio, when the build order has to choose"; and four more). Those trade-offs then settled 7 of 9 deduced brief changes with no further question to the owner.

Where they should live took three tries:

1. **A new vision page.** The owner asked what made it different from the press release; only the ranked trade-offs were new, and the page broke abcd's brief layout.
2. **The framing page.** It kept the layout, but buried the trade-offs on the sixth page, after context, mental model, scope and personas.
3. **Beneath the press release, as Product principles** (the owner's proposal). This is the working-backwards pattern, where internal material follows the release on the same page. It reads in order: the story, then what wins when goals clash.

## The naming point

abcd already uses "principles" for its engineering conventions (development record, principles folder: prefer SOTA, script-first, and so on). A brief section called just "Principles" would give one word two meanings in a managed repository that vendors abcd's principles. "Product principles" separates them: product principles decide what the product favours; abcd's principles decide how it is built.

## Proposed

- The template's press release page gains a Product principles section beneath the release: ranked statements in the X-even-over-Y form, each with a persona example, and a line saying which were ranked against which.
- The vision step fills it; the owner's vision words go into the release as their quote.
- The coverage pass reads the section as product-thinker-owned.

Remedy the reporter proposes: Add a Product principles section to the brief template's press release page, directly beneath the release: ranked X-even-over-Y statements, each with a persona example; name it product principles so it never collides with abcd's engineering principles; have the coverage pass treat it as product-thinker-owned.

Reported by a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0) through the abcd inbox as rpt-2610071653351920, a enhancement against abcd v0.13.1, surface brief template (01-product), disembark coverage pass, the vision refinement step.

Evidence:

- rpt-2610071653351920 (the report, kept in the inbox)
- lab-261002171217-c372ff6
- rpt-2610071611295053
