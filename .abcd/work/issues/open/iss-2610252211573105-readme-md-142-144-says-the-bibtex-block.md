---
schema_version: 1
id: "iss-2610252211573105"
slug: "readme-md-142-144-says-the-bibtex-block"
severity: "nitpick"
category: "drift"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D8"
origin: researcher-authored
production_mode: hand-written
found_at: "README.md"
remedy: "Either change README.md:149 to '  title   = {{abcd}},' so the block is what the CFF yields, or set CITATION.cff's title to the long name. The second is the owner's call, because it changes what the cite button hands out. Either way, add the title to the cases in internal/core/launch/citation_test.go so it is pinned like the other three fields."
---

README.md:142-144 says the BibTeX block 'is rendered from [CITATION.cff](CITATION.cff), which is the record GitHub's own *Cite this repository* button reads'. But README.md:149 reads '  title   = {{abcd}: {Agent-Based} {Configuration} for {Development}},' while CITATION.cff:4 declares title: 'abcd' (the long name is in abstract:, CITATION.cff:5). The cite button therefore hands out a different title from the README's. TestReadmeCitationAgreesWithCitationCFF pins only the author, the repository URL and the licence (internal/core/launch/citation_test.go:33-43), so nothing catches the title.
