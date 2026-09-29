---
schema_version: 1
id: "iss-2609290411321963"
slug: "the-memory-writer-s-page-filename-refusal-quotes-the-token"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/memory/redact.go"
resolution: "Fixed: judgeFilename still names the refused page, which is what the operator repairs, but every span the hard_fail findings over filenameJudgeTexts matched is sealed, marked byte by byte in the joined name so overlapping matches leave no raw tail; a match that cannot be located leaves the whole name described. The writer_filename tests pin the sealed name for the plain slug and all three separator spellings and assert the token is absent."
impact: fix
resolved_by:
  commit: "47a510f88"
---

The memory writer's page-filename refusal quotes the token it refuses. judgeFilename (internal/core/memory/redact.go, called from writer.go at the write boundary) refuses a page whose host-chosen type, domain or slug carries a hard_fail secret, and its message names the whole filename raw: a slug of ghp_ and forty characters is refused with the token in the error, so it reaches the terminal and the transcript, the class iss-2609290218032954 and iss-2609290300464268 closed elsewhere. The echo is deliberate and pinned: TestWriteRefusesASecretShapedFilename and the separator-spelling cases in writer_filename_test.go assert the refusal names the page, because the filename is the write's identity and the operator needs to know which page to repair. scanner.RedactRefusal cannot square the two: an underscore is a word character, so a token joined behind topic_auth_ has no word boundary and is not redacted in the joined name, and a token can straddle the type, domain and slug separators. Found by lane drainEcho3's sweep for iss-2609290300464268. Fix direction: seal, byte for byte in the joined name, every span the hard_fail findings over filenameJudgeTexts matched, and keep the rest of the name. That keeps both requirements, the page named and the token not echoed, so it needs no ruling; describing the whole name would give up the identity the test pins. Detector: a page whose slug, or a spelling across the separators, is a well-formed token is refused, nothing is written, and the refusal names the page with the token sealed.

## Grounds

- pursued: a page whose name carries a well-formed token, in the slug or across a separator, is refused naming the page with the token sealed; a refusal carrying the token would show it wrong
