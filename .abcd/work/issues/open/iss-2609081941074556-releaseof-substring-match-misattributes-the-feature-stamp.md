---
schema_version: 1
id: "iss-2609081941074556"
slug: "releaseof-substring-match-misattributes-the-feature-stamp"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bughunt-triage"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/compose.go"
---

releaseOf stamps the homepage feature block by strings.Contains of the record id against each CHANGELOG line, with no word boundary, walking newest dated section first. A short id therefore inherits a longer one: on ec7f40d6 releaseOf(itd-9) returns 0.4.1 off the itd-93 credit and releaseOf(itd-1) returns 0.7.1 off itd-130, and a future superstring landing in a newer section restamps the featured id. site check does not validate the stamp. The same package already carries bodyHandleRe. Fix: match the record id at a word boundary (reuse bodyHandleRe, or split the line into tokens); newest-section-first can stay. Detector: a changelog whose newer dated section mentions itd-1990 must not stamp featured itd-199, while a line naming itd-199 as its own id still matches. Independent of the Audit Notes rollup scanner. Reported as GitHub issue 624 against ec7f40d6.
