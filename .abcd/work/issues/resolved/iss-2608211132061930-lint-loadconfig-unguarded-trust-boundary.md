---
schema_version: 1
id: "iss-2608211132061930"
slug: "lint-loadconfig-unguarded-trust-boundary"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "bughunt-b-round-4"
found_at: "internal/core/lint/config.go"
resolution: "Routed the read through fsutil.ReadGuarded (O_NOFOLLOW + regular-file-on-fd + O_NONBLOCK + 256KiB cap), with an Lstat symlink refusal on the config directory, mirroring guard.Load/rules.Load; ELOOP/ErrNotRegular/ErrTooBig folded into path-free typed errors and the os.IsNotExist branch preserved. Watched-fail: TestLoadConfigRefusesFIFO/SymlinkedLeaf/SymlinkedDir/Oversize."
impact: fix
---

lint.LoadConfig reads the docs-lint/record-lint config with a raw os.ReadFile — no O_NOFOLLOW, no regular-file check, no size cap — while its .abcd/*.json siblings guard.Load, rules.Load and positioning.LoadConfig all guard the read as a trust boundary. The read is hook-reachable (ahoy.Detect via hook session-start/end) and cross-repo-clonable, so a committed .abcd/docs-lint.json symlink to a FIFO wedges abcd docs lint / lint / ahoy / hook / cite refresh and the record-lint CI leg, a /dev/zero target OOMs the CLI, and an out-of-repo symlink target runs the entire ruleset from a file the repository does not own (gate substitution).