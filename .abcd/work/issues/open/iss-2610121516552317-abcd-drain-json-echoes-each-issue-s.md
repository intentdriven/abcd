---
schema_version: 1
id: "iss-2610121516552317"
slug: "abcd-drain-json-echoes-each-issue-s"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "fixing iss-2610121505153157 on fix/sec-build-next-series"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain.go"
remedy: "Where the drain builds an issue's disposition reason from ledger text, quote each record path and the deferred_after value with %q (or pass them through termsafe.Sanitize) so no raw control or bidi rune reaches drain --json or a stored judgement note; a test plants a ledger record whose name and deferred_after carry U+202E and U+009B and expects nothing raw on drain --json and a drain.json that still reads back."
---

abcd drain --json echoes each issue's disposition reason straight from the ledger reader: that text includes the reader's error messages with record paths unquoted and a record's raw deferred_after value, so a repository whose ledger file name or deferred_after carries a control or bidi rune (U+202E, U+009B) puts it raw on the --json stream, and the drain copies the same text into a judgement's note in drain.json, where readDrain (since iss-2610121505153157) then refuses the whole file. Found by the implementer of iss-2610121505153157.

Security scope (abcd-fixes-what-it-adds): the route does not exist without abcd. The reader, the text it builds and the --json stream that carries it are abcd's own, so abcd added it and fixes it, at the boundary the text enters by; the fix changes nothing about how the repository works.
