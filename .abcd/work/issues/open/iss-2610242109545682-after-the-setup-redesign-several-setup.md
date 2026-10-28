---
schema_version: 1
id: "iss-2610242109545682"
slug: "after-the-setup-redesign-several-setup"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md"
remedy: "In one docs change: add the four entries to the tree in 01-ahoy.md and widen the backups line; add the shell line and tool copies to the uninstall summary; add deps.gitleaks_upgrade to the tool paragraph; extend the --yes flag help and regenerate docs/reference/cli; and state the .bash_profile, .bash_login, .profile order and ZDOTDIR in docs/how-to/install.md."
---

After the setup redesign, several setup descriptions are incomplete rather than false: the brief's abcd home tree omits identity/, tools.json and shell-path.json and says backups/ holds only the assistant's settings; the uninstall summary omits that uninstall removes the shell PATH line and abcd's own tool copies; the tool paragraph omits the owned-copy upgrade offer; the --yes help text does not list the shell settings, the commit checks or tools among what it skips; and the public install how-to simplifies which bash startup file is edited.
