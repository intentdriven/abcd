---
id: dcn-2610140000000003
slug: before-1-0-only-the-newest-release-is
date: 2026-10-14
---
2026-10-14 — Before 1.0 only the newest release is offered: the product thinker ruled, after the v0.14.1 cut, "We don't want to offer them prior to 1.0", choosing "Only v0.14.1" over the recorded newest-per-line rule (brief, launch chapter; draft itd-70), which would also have kept v0.13.4, v0.12.0 and v0.11.1. The eight earlier GitHub Releases (v0.14.0, v0.13.4, v0.13.3, v0.13.2, v0.13.1, v0.13.0, v0.12.0, v0.11.1) and their assets were deleted by hand that evening; their tags and the git history stay, so each is still buildable from its tag. Automating the removal at every cut, and changing the recorded rule to match, is iss-2610141938299842 ("auto remove previous ones"); iss-282 holds the older newest-per-line executor gap. Not chosen: newest per line, which keeps older builds on offer.
