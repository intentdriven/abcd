---
schema_version: 1
id: "iss-2608211134281912"
slug: "stdin-operand-readers-truncate-instead"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bughunt-b-round-4"
found_at: "internal/surface/cli/cli.go"
resolution: "Added readCappedStdin (a cap+1 probe that refuses an over-cap payload whole, message paralleling readGuardedOperand) and substituted it for the bare LimitReader at all four stdin readers. Watched-fail: TestStdinOperandReadersRefuseOverCapWhole over all four readers."
impact: fix
---

Four untrusted-operand stdin readers (readIdeatePayload, readLessonsPayload, readSynthesisPayload, readSource) read exactly cap bytes via a bare io.LimitReader(cap), so an over-cap '-' payload is silently truncated into a severed prefix instead of refused, while each reader's file transport (readGuardedOperand) refuses the same bytes with fsutil.ErrTooBig. Worst on the history capture path: history.Capture performs no parse and no cap check, so an over-cap stdin transcript is stored as an 8 MiB prefix, reported 'stored', under a sha256 idempotency key computed over the prefix — breaking spc-4's refuse-whole invariant. Also lets an over-cap ideate verdict past both the MaxPayloadBytes cap gate and the dec.More() trailing-data gate at a crafted boundary. Unswept siblings of resolved iss-201/iss-347; every site's doc comment claims stdin parity with the file cap.