---
schema_version: 1
id: "iss-2610121505153157"
slug: "readdrain-internal-core-implement-loop"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "fixing iss-2610121206163414 on fix/sec-build-next-series"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain.go"
remedy: "When readDrain decodes a drain state, refuse any stored hand-back or judgement text carrying a control or bidi rune (termsafe.Sanitize(v) != v, the loop's predicate), with the drain stage's restore-or-remove remedy and without echoing the value; tests forge each field and expect the refusal, with nothing raw on drain --json."
refines: [iss-2610121206163414]
resolution: "readDrain now refuses a drain state whose stored rule, lane outcome, hand-back or judgement text (reason, home, question, rule, draft, wrote, note, kind, route, from, ids and paths) is not termsafe.Sanitize-equal, at the drain stage with the restore-or-remove remedy and without echoing the value, so no control, bidi or zero-width rune from drain.json reaches drain --json."
impact: fix
resolved_by:
  commit: "9ed7e476a751e20ff2844e3bf9b7b9534e5f4d4e"
---

readDrain (internal/core/implement/loop/drain.go) accepts the free text a drain state stores in .abcd/.work.local/run/drain.json unchecked: each hand-back's reason, home, question, rule, draft and wrote fields and the judgements are copied into 'abcd drain --json' as stored (res.HandBacks = st.HandBacks), so a hand-written or repository-shipped drain.json can carry control or bidi runes (U+202E, U+009B) onto the JSON stream; the same display-spoofing class as iss-2610121206163414, which hardened the drain's stored pace. Found by the implementer of that fix.

Security scope (abcd-fixes-what-it-adds): the route does not exist without abcd. The drain state file, its reader and the --json stream are abcd's own, so abcd added it and fixes it; a fix refuses or masks a malformed file abcd itself reads and changes nothing about how the repository works.
