---
schema_version: 1
id: "iss-170"
slug: "resolvepluginroot-s-ancestor-walk"
severity: "major"
category: "observation"
source: "user-observation"
found_during: "second-repo install session"
found_at: "internal/core/ahoy/store.go"
resolution: "resolvePluginRoot now canonicalises the executable path before the ancestor walk, so the pinned PATH symlink ahoy install writes no longer defeats plugin-root resolution"
impact: fix
---

resolvePluginRoot's ancestor-walk fallback defeats the install layout ahoy itself creates: os.Executable() (internal/core/ahoy/store.go:92) is used without filepath.EvalSymlinks, so when abcd is invoked through the pinned PATH symlink the walk scans the symlink's ancestors (~/.local/bin, ~, /) and never finds hooks/ — a permanent phantom 'plugin root not resolvable' gap on exactly the layout ahoy install writes. Neither env var is set in a normal shell (CLAUDE_PLUGIN_ROOT exists only inside harness hook invocations), so the fallback is the path that matters. Fix is EvalSymlinks before the walk, plus a test invoking through a symlink watched failing first.