---
schema_version: 1
id: "iss-2610310331279313"
slug: "the-install-guide-says-setup-adds-the"
severity: "nitpick"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/how-to/install.md"
remedy: "State in docs/how-to/install.md the files setup chooses, as internal/core/shellpath picks them: ZDOTDIR for zsh, and the first existing bash login file on macOS."
---

The install guide says setup adds the PATH line to .zshrc for zsh and .bash_profile for bash on macOS; setup honours ZDOTDIR and edits the first of .bash_profile, .bash_login and .profile that exists.
