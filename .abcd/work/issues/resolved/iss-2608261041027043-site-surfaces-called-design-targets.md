---
schema_version: 1
id: "iss-2608261041027043"
slug: "site-surfaces-called-design-targets"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-a/round-7"
found_at: ".abcd/development/brief/05-internals/README.md"
resolution: "Aligned the 05-internals row-10 and 04-surfaces row-22 cells with 10-site.md: the check gates, explorer and deploy ship; the design target is the abcdev.app MkDocs root."
impact: internal
resolved_by:
  commit: "2811321ba4b6adcd06bf05d14101c4bd76d26cdc"
---

The 05-internals index and the 04-surfaces row still call shipped site capabilities design targets, contradicting 10-site.md. 05-internals/README.md row 10 says the check gates, the explorer pages and the deploy workflow are design targets, and 04-surfaces/README.md row 22 says the deploy workflow remains a design target — but 10-site.md marks only the abcdev.app MkDocs root as the design target, abcd site check is registered shipped in surface.json, the explorer shipped (itd-136), and the deploy workflow ships (adr-48 accepted, release.yml calls site.yml, CHANGELOG 0.6.6). Both index cells are stale surviving copies of the earlier all-staged claim; outside the release-gate crosscheck's pinned briefDocs corpus, so no gate catches the record-vs-record contradiction. Fix: align both cells with 10-site.md.