---
schema_version: 1
id: "iss-2610121516559163"
slug: "readstate-internal-core-implement-loop"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "fixing iss-2610121505153157 on fix/sec-build-next-series"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/state.go"
remedy: "When ReadState decodes a run's state, refuse a lane whose stored hand-back text (reason, home, question and any other free-text field of HandBack) is not equal to termsafe.Sanitize of itself, with the run state's restore-the-file remedy and without echoing the value, as readDrain now refuses its own stored text; a test forges each field and expects the refusal, with nothing raw on drain --json or implement record --json."
refines: [iss-2610121505153157]
resolution: "readStateIn now refuses a run state whose lane hand-back text (kind, reason, home, discarded, verdict, verdicts, findings, not_met, undecided) is not termsafe.Sanitize-equal, with the restore-the-run's-state-file remedy and without echoing the value, so no control or bidi rune from state.json reaches implement record --json, implement status --json or the drain's routing move."
impact: fix
resolved_by:
  commit: "44c1c8a455ee7159009489a3c008f7aff92b4465"
---

ReadState (internal/core/implement/loop/state.go) does not check a lane's stored hand-back text in a run's state.json, and on the call that routes a hand-back the drain echoes that text onto abcd drain --json before drain.json is ever re-read, so a hand-written or repository-shipped state.json whose hand-back reason, home or question carries a control or bidi rune (U+202E, U+009B) puts it raw on the stream. The receipt path sanitizes these fields when the loop takes them, so only a forged file carries them. Found by the implementer of iss-2610121505153157.

Security scope (abcd-fixes-what-it-adds): the route does not exist without abcd. The reader, the text it builds and the --json stream that carries it are abcd's own, so abcd added it and fixes it, at the boundary the text enters by; the fix changes nothing about how the repository works.
