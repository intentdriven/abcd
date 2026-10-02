---
schema_version: 1
id: "iss-2609301557191251"
slug: "the-runner-s-admit-internal-core-runner-proc-go-starts-a"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner/proc.go"
remedy: "After EvalSymlinks, stat the resolved binary, its directory and the PATH entry's directory, and refuse (absent, so the route falls back) any whose mode carries a group or other write bit, through the one fsutil test CallersAlone applies, exported as fsutil.WritableByOthers; ownership is not required, since a root-owned system binary is legitimate."
resolution: "admit stats the resolved binary, its directory and the PATH entry's directory and refuses any group or other can write, through fsutil.WritableByOthers, the one test CallersAlone applies."
impact: fix
resolved_by:
  commit: "70d49494f"
---

The runner's admit (internal/core/runner/proc.go) starts a harness binary resolved on PATH even when the binary, or the directory PATH reaches it through, is writable by group or other: a chmod 777 directory holding claude was admitted, so any account that can write there chooses what the loop runs with the person's credentials.

## Grounds

- pursued: a chmod 777 PATH directory holding the fake, or a mode 0772 fake binary, is refused as absent naming group or other and never launched; a launch from either, or a 0755 system harness refused, would show it wrong
