---
schema_version: 1
id: "iss-2609300057311045"
slug: "on-bash-5-the-guard-allows-rm-rf-x-home"
severity: "major"
category: "security"
source: "impl-review"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/unknown.go"
remedy: "replacementTexts reads the pattern both ways (ending at a quoted / as bash 3.2 does, and past it as bash 5 does) and unions the texts; readPattern reads $\" outside double quotes as the double-quoted string it opens; grounds: printf under /bin/bash 3.2.57 and bash 5.3.9 on the reviewed forms."
resolution: "replacementTexts unions the bash 3.2 and bash 5 pattern boundaries, and readPattern reads $\" as a double-quoted string; TestQuotedSlashReplacementsTheWrittenCompareReads"
impact: fix
resolved_by:
  commit: "598f47756"
---

On bash 5 the guard allows rm -rf ${X/"/"*/$HOME} and ${X//"/"*/$HOME}, which bash 5.3 prints as the home: readPattern ends a replacement pattern at a quoted /, as bash 3.2 does, while bash 5 keeps a quoted / in the pattern (review-guardSet MAJOR-3). The same reader takes $"" in a pattern for a literal $, so ${X%%$""*}/ reads as not whole and is allowed though every shell prints /.

## Grounds

- pursued: ${X/"/"*/$HOME} and ${X%%$""*}/ block while ${X/[/]*/$HOME} and ./${X/"/"/_} stay allowed; a bash 5 pattern boundary other than a quoted slash that still allows would show it wrong
