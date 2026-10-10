---
schema_version: 1
id: "iss-2610090821543020"
slug: "core-worktree-ancestor-becomes-store-root"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "private security report, filed 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/repo.go"
remedy: "Keep the containment check and also require that `rev-parse --git-dir` for the asked directory is that toplevel's `.git` or a gitfile pointing at its common dir; prove it with a gitutil test (watched fail first) that `core.worktree=../..` makes `CheckoutRoot` return ErrToplevelShape and `abcd decide` writes nothing in the parent, a subdirectory of a normal checkout still resolves, a sibling worktree is still refused, and the site and launch-precheck refusals of a parent output directory hold; sweep siblings (every root resolver that trusts `--show-toplevel`)."
resolution: "Toplevel now asks git for the git directory it discovered alongside the toplevel and accepts the toplevel only when its .git is that directory or a gitfile naming it, so an ancestor selected by core.worktree is refused and CheckoutRoot writes no store there"
impact: fix
---

A copied checkout whose `.git/config` sets `core.worktree=../..` makes abcd take an ancestor directory as the repository root, so `abcd decide`, intent, spec, capture and memory writes land under that ancestor instead of the checkout.

A private security report, fixed in this release; its advisory, with the full text and reproduction, is published with the release.

Evidence (lines at main 7549ca2d5): `ToplevelContext` runs `rev-parse --show-toplevel` (internal/gitutil/repo.go:565) and `ToplevelShaped` accepts any absolute answer that `PathWithin` says contains the asked directory (internal/gitutil/repo.go:599, internal/fsutil/paths.go:125). `CheckoutRoot` (internal/gitutil/repo.go:369) feeds `decideStoreRoot` (internal/surface/cli/decide.go:91) and the intent, spec, memory and capture-ledger roots. internal/core/site/outdir.go:173 names `core.worktree` as a decoy; `site.Build` and `launch.PrecheckPayload` refuse the parent. The written bodies are abcd's own records, and no program runs. `git clone` drops the key; the config has to arrive in a copied `.git`.