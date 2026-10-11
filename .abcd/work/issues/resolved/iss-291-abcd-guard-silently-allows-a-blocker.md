---
schema_version: 1
id: "iss-291"
slug: "abcd-guard-silently-allows-a-blocker"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: "internal/core/guard/execstring.go"
resolution: "Thread value flags into clusteredPayload; a value-taking short flag before the payload letter aborts the cluster read, so script -Tc out.txt -c blocks. Watched-fail test added."
impact: fix
---

abcd guard silently allows a blocker whose exec-string verb carries a value-taking short-flag cluster before the payload flag (script -Tc out.txt -c '<blocker>'): clusteredPayload accepts the cluster without consulting the verb's value flags, mis-reads -Tc as -T -c, resolves a bogus payload and switches the Tier-2 fail-safe off for the segment
## Evidence

- `internal/core/guard/execstring.go` — `clusteredPayload(tok, payloadFlags)` accepts a
  short cluster when every letter before the payload letter passes `isShortOptionLetter`
  (`[A-Za-z0-9]`), never consulting `execStringOtherValueFlags` (which lists `script`'s
  value flags `-o -O -B -m -I -T`). For `-Tc` it treats `-T` as a boolean and returns a
  glued/next-token payload, short-circuiting `scanExecString` past the real later `-c`.
- `internal/core/guard/speculate.go` — because the segment now reports `carriesPayload`,
  the Tier-2 speculative fail-safe skips it, so neither tier produces a verdict.
- Reproduced against a build of `./cmd/abcd` (stdin form):
  `script -c '…' out.txt` blocks (exit 1); `script -Tc out.txt -c 'git push --force origin main'`
  allows (exit 0); `su -lc '…'` still blocks. `script -Tc out.txt -c '/bin/echo M'` prints `M`,
  so the payload genuinely runs — a true-positive miss.

## Adversarial review

CONFIRMED (substantive, modest end) by an independent refuter: the `-O`/`flock -w` example
spellings are false alarms (the tool itself rejects them), but `script -Tc` is a working
silent-allow bypass; distinct from the filed tokenizer/wrapper gaps and from iss-272's
long-spelling table. Fix: thread the verb's value flags into `clusteredPayload` and decline
the cluster when a pre-payload letter is a value-taking short flag.
