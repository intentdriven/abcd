---
id: dcn-2608200000000006
slug: homebrew-distribution-recorded-and
date: 2026-08-20
---
2026-08-20 — Homebrew distribution recorded and parked. Viable now that the
repo is public, and it is what the reference class (gh, lazygit, k9s) leans
on; a personal tap is the solo-maintainer shape and homebrew-core needs
notability. Deferred until after a binary-update verb exists, because the
two interact: an updater must gate on install-channel detection and print
`brew upgrade abcd` instead of self-replacing when the binary's RESOLVED
path sits under a brew prefix (/opt/homebrew, /usr/local,
/home/linuxbrew/.linuxbrew — a prefix test, never a "Cellar" substring
match). The named incident is flyctl's autoupdate postmortem, "Flyctl
Versions, Autoupdating, and the CLI Apocalypse"
(https://community.fly.io/t/flyctl-versions-autoupdating-and-the-cli-apocalypse/13794):
brew-path update loops, Nix read-only breakage, Windows UAC. Note
for the revisit: goreleaser's `homebrew_casks` pipe is the current SOTA
route, but abcd's release pipeline is bespoke (semantic-gate attestations,
no goreleaser), so the tap push would be a hand-rolled release-workflow step.
