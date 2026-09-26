---
schema_version: 1
id: "iss-2609260057117878"
slug: "the-install-summary-s-plain-language-copy-says-three-untrue"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/install_summary.go"
resolution: "The identity-pin item names .abcd/config/identity.json as the correction, the skipped-pin item names git's user.name/user.email and the config-change question, and each visibility meaning names every path its .gitignore block fences, memory/ included."
impact: fix
resolved_by:
  commit: "d44954d5"
---

The install summary's plain-language copy says three untrue things. The identity-pin item tells a person with a wrong recorded name or email to run abcd ahoy install again, but stepIdentityPin writes only while no pin exists, so a re-run changes nothing; the skipped-pin item says to answer y if the name and email shown are yours, but nothing shows a name or email (the pin rides on the settings confirm); and the public visibility meaning says only .abcd/ is kept out of git, omitting the /memory/ fence the public block also writes (gitignore.go visibilityEntries).

## Grounds

- pursued: the three sentences match the code; TestIdentityPinActionNamesWhatCorrectsAWrongPin, TestSkippedPinActionNamesWhatThePersonChecks and TestVisibilityMeaningNamesEveryFencedPath would fail if a re-run began correcting a pin without the copy changing, if the prompt stopped naming config-change, or if a fenced path went unnamed
