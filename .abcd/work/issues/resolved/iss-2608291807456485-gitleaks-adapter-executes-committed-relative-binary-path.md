---
schema_version: 1
id: "iss-2608291807456485"
slug: "gitleaks-adapter-executes-committed-relative-binary-path"
severity: "critical"
category: "security"
source: "agent-finding"
found_during: "v0.6.9-security-pass"
found_at: "internal/adapter/gitleaks/gitleaks.go"
resolution: "Binary admission rule: a configured or PATH-located gitleaks binary runs only when absolute, resolved outside the repository (lexically and after symlink resolution), regular, executable and named gitleaks (a wrapper or renamed binary is refused); refusals are loud and never fall back to PATH."
impact: fix
---

GHSA-fg9r-3f8g-89m6: the gitleaks adapter executes an attacker-controlled binary from a committed relative path. resolveBinary accepts a relative cfg.Path from the committed .abcd/config/gitleaks.json, validates it only with os.Stat (symlinks follow), resolves it against the working directory, and the runner executes it under a #nosec G204 waiver; the LookPath fallback also honours inherited PATH (CWE-426). Any SessionStart drain or history capture/drain in a hostile checkout runs the attacker's binary with the victim's privileges.
