# Release 0.14.2 (2026-10-15)

abcd's home keeps itself out of a dotfiles repository. abcd keeps session transcripts, private idea verdicts and run state under `~/.abcd.noindex`. For anyone who keeps their whole home folder in a git repository they push to a public remote, that put those files one `git add -A` away from the remote. The first time abcd runs, it writes a `.gitignore` holding `*` in `~/.abcd.noindex`, so the next `git add -A` skips everything abcd keeps there and nothing private leaves the machine by accident. abcd writes the file once: if you delete it because you want your dotfiles repository to track abcd's home, abcd leaves it deleted. Credentials stay out of every git working tree whatever the file says. (itd-2610140858048004)

Also in this release:

- RepoPrompt over MCP is one opt-in reviewer route, beside the host's own agents (itd-6)

The line-by-line record of this release is its section in CHANGELOG.md.
