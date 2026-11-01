---
schema_version: 1
id: "iss-2610311818545897"
slug: "update-news-runs-the-plugin-binary-in-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "update news after the v0.16.0 update, in abcd's own checkout"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/update.md"
remedy: "In a source checkout of abcd, have the update page run the setup that waits (ahoy install --only-waiting), the one step that writes the repository, through scripts/abcd.sh, as AGENTS.md requires, which keeps the exit 2 its answers-file loop reads. Keep update news and its --seen and --answer calls on the plugin binary: they write only under ~/.abcd.noindex/ and refuse a dev build. A checkout build is versioned dev, so its --only-waiting asks no changed default and stamps no release; give it the release the offer named (for example the plugin binary's version, passed in) so it puts every changed default and stamps setup_version as the offer counted. Keep the dogfood rung, and make the session-start line in internal/surface/cli/update_news.go agree with the page."
---

The update news section of the update command page says to run the plugin's own binary, never another abcd, for update news and then for the setup that waits. In a source checkout of abcd, AGENTS.md says every abcd invocation is scripts/abcd.sh, because the plugin binary is the last release and can disagree with the tree. Following the page on 2026-10-31, the v0.16.0 binary ran the setup that waits in abcd's own checkout and rewrote AGENTS.md and .gitignore. Its text matched origin/main this time, the local main being the one behind, but nothing in the page keeps a release older than the tree from writing its own text there.
