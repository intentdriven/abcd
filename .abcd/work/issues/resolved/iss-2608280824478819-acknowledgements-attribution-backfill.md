---
schema_version: 1
id: "iss-2608280824478819"
slug: "acknowledgements-attribution-backfill"
severity: "major"
category: "documentation"
source: "user-observation"
found_during: "attribution-review-2026-08-28"
found_at: "ACKNOWLEDGEMENTS.md"
resolution: "Backfilled the canonical credit surface from the 2026-08-28 attribution review: Inspirations entries for CARL, PAUL, SpecStory, the Karpathy LLM-wiki gist, gitleaks, TruffleHog and Rams; the Diataxis entry completed with author and CC-BY-SA 4.0; the mattpocock entry widened to four adaptations; Dell'Acqua et al. 2023 added to the references and CSL; the Horn entry carries an honest URL caveat; the Weng essay is source-linked from all three principles; the Iacob and GLM-5 citations carry arXiv ids; the skills' third-party authorship is recorded in the skills-evaluation note; the itd-27 to-prd link no longer dangles."
impact: fix
---

The canonical credit surface is incomplete: design-shaping inspirations fully credited in the committed record never propagated to ACKNOWLEDGEMENTS.md. Backfill set from the 2026-08-28 attribution review: add Inspirations entries for CARL and PAUL (both Kahler, MIT; mechanisms adopted per itd-3 and itd-1), SpecStory (adr-29 seam, specstory-import ships in CLI help), the Karpathy LLM-wiki gist (memory-substrate pattern), gitleaks and TruffleHog (integrated scanners, one self-declared adapted regex), and Dieter Rams; complete the Diataxis entry with the author and CC-BY-SA 4.0; widen the mattpocock entry (ADR three-clause test, PRD shape); add Dell'Acqua et al. 2023 to the CSL references; add source URLs to the Horn and Weng credits and the arXiv id to the Iacob citation; record the skills' authorship in the socratic/first-principles research note; fix the dead to-prd deep link in itd-27.