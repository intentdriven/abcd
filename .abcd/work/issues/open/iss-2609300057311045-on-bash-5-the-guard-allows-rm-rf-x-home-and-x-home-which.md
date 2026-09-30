---
schema_version: 1
id: "iss-2609300057311045"
slug: "on-bash-5-the-guard-allows-rm-rf-x-home-and-x-home-which"
severity: "major"
category: "security"
source: "impl-review"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/unknown.go"
remedy: "replacementTexts reads the pattern both ways (ending at a quoted / as bash 3.2 does, and past it as bash 5 does) and unions the texts; readPattern reads $\" outside double quotes as the double-quoted string it opens; grounds: printf under /bin/bash 3.2.57 and bash 5.3.9 on the reviewed forms."
---

On bash 5 the guard allows rm -rf ${X/"/"*/$HOME} and ${X//"/"*/$HOME}, which bash 5.3 prints as the home: readPattern ends a replacement pattern at a quoted /, as bash 3.2 does, while bash 5 keeps a quoted / in the pattern (review-guardSet MAJOR-3). The same reader takes $"" in a pattern for a literal $, so ${X%%$""*}/ reads as not whole and is allowed though every shell prints /.
