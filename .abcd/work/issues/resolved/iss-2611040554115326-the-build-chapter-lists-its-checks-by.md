---
schema_version: 1
id: "iss-2611040554115326"
slug: "the-build-chapter-lists-its-checks-by"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/34-build.md"
remedy: "Name the `eligible` check in the chapter's checks section, in or beside the key bullet."
resolution: "The build chapter's key bullet names the eligible check beside the drain-rule sentence (and peers beside the peer sentence); a test holds every check name Check reports to the chapter's checks section."
impact: internal
resolved_by:
  commit: "f361fe6f8fe834684398c0e112ac470bd3ce8849"
---

The build chapter lists its checks by name and says a refusal names the check (.abcd/development/brief/04-surfaces/34-build.md:40-48,134). An issue id the repository's drain rule does not take is refused at a check named `eligible` (internal/core/implement/loop/check.go:41). The key bullet describes that rule at line 44 without the name, so a reader holding the refusal cannot find the check under the name it gives.

## Grounds

- pursued: a reader holding a refusal at `eligible` finds that name in 34-build.md's checks section; TestBuildChapterNamesEveryCheck failing, or a check constant whose name is absent from the section, would show it wrong
