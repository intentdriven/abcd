---
schema_version: 1
id: "iss-2609281613094952"
slug: "the-status-board-prints-the-checkout-s-absolute-path"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainRedact"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/core.go"
---

The status board prints the checkout's absolute path with no redaction at all: core.Status sets Dir to filepath.Abs(cwd) (internal/core/core.go:46), bare abcd renders it raw as its first line ('abcd — <dir>', internal/surface/cli/cli.go:285) and abcd --json carries it whole as dir. Under HOME it prints /Users/<account>/..., and outside HOME the full path, in the output a person pastes most often. The board should name the checkout by the display rule fsutil.DisplayPath states (home-relative under HOME, the directory's base name outside it), the sibling of iss-2609281329007423, which routed every other checkout and worktree display but not this one.
