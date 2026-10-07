# Upgrade to v0.13.0

v0.13.0 is a breaking release. abcd's own folder in your home directory is
`~/.abcd.noindex`, a name the Mac's search indexer passes over, where earlier
releases used `~/.abcd`. abcd never moves the old folder for you. This guide
moves a machine across once, says what to do if both folders end up standing,
and brings each project abcd manages up to date. Two more changes follow the
rename: only your own settings send a role to a command-line runner, never a
project's committed ones, and AGENTS.md is the one conventions file abcd
writes.

A machine that never ran an earlier abcd has no `~/.abcd` and needs none of
this.

## What stops until you rename

Once you update the plugin, and while `~/.abcd` still stands, abcd stops
everything before it writes anything: every command, every hook and the status
line. In an agent session the safety check refuses every shell command, the
rename included: right after a plugin update the new plugin folder holds no
abcd binary yet, and the check refuses everything until the folder is renamed.
So the rename is done in a plain Terminal window, outside any agent session.

## Move the folder

Do this once on each machine.

1. Update the plugin with `/plugin update abcd`, or a binary in
   `~/.local/bin` with `abcd update`.
2. Open a plain Terminal window and rename the folder:

   ```sh
   mv ~/.abcd ~/.abcd.noindex
   ```

3. In the same window, reconnect the worktrees kept in it, however deep each
   one sits. git records each worktree's location in full, so until this runs,
   each repository lists its moved worktrees as prunable, and a prune would
   delete their links:

   ```sh
   find ~/.abcd.noindex/worktrees -type d -exec test -e {}/.git \; -prune -exec test -f {}/.git \; -exec git -C {} worktree repair \;
   ```

   Each `repair: gitdir incorrect` line it prints is a link it fixed, not a
   failure. `abcd ahoy` names any worktree still unlinked afterwards, with the
   command that repairs that one.

4. Start the agent session again. A session that worked inside a worktree under
   the old folder opens from that worktree's new path, under
   `~/.abcd.noindex/worktrees/`.

## Recover when both folders exist

If something older ran after the rename, such as a session that was still open
with the earlier plugin, it creates a new, small `~/.abcd` beside
`~/.abcd.noindex`. abcd then stops and names both folders, and moves neither.

1. Look inside the new `~/.abcd` to see what it holds, typically a run log or a
   transcript from that one older run.
2. Move it out of your home folder, keeping it wherever you keep things you may
   want later.
3. Run the repair line again:

   ```sh
   find ~/.abcd.noindex/worktrees -type d -exec test -e {}/.git \; -prune -exec test -f {}/.git \; -exec git -C {} worktree repair \;
   ```

## Refresh each managed project

A project abcd manages keeps naming the old folder, `~/.abcd/rules.json` and
`~/.abcd/trusted-roots`, in the managed block of its conventions file until
abcd's setup runs in that project again. In each project abcd manages:

1. Run `abcd ahoy`. It reports the block as `marker block outdated`.
2. Run `abcd ahoy install`. It rewrites the block alone and leaves every line
   around it as it was.
3. Commit the changed conventions file.

## Update your own scripts

Search your scripts, shell profile and CI workflows for `~/.abcd/` and
`$HOME/.abcd`, and change each to `~/.abcd.noindex`. A link from `~/.abcd` to
the new folder does not keep them working: abcd takes any entry at the old
name, a link included, for the old folder and stops.

A project's own `.abcd/` folder, beside its sources, keeps its name. Change
only the paths that point into your home directory.

## Route a role to a runner from your own settings

A role, such as the implementer or a reviewer, can run through a command-line
runner (`claude` or `opencode`) instead of in your agent session. A runner
spends your own key, so only you may send a role to one: from your machine's
settings, `~/.abcd.noindex/config.json`, or for one run from the command you
type. A project's committed `.abcd/config.json` may still keep a role on the
host, and nothing more.

When a project's `.abcd/config.json` sets `roles.<role>.runner` to `claude` or
`opencode`, abcd skips that route, prints a line naming the file and the key,
and runs the role as if the project had not routed it: through your own route
if you set one, otherwise in your agent session. Setting `runner.<name>` in a
project's file stops abcd with a message naming your machine's file instead.

If you want the role to keep running through the runner:

1. Copy the `roles.<role>.runner` entry from the project's `.abcd/config.json`
   into `~/.abcd.noindex/config.json`.
2. Remove it from the project's file, and commit that change.
3. Run the command again and check that the skip line is gone.

The `opencode` runner reads none of the project's own instructions, settings,
agents or skills when it runs a role, so a run sees only what abcd hands it.

## Keep your project's conventions in AGENTS.md

abcd writes its managed block into `AGENTS.md` and into no other conventions
file: it writes no tool's own copy, such as `CLAUDE.md`, and setup creates no
link to `AGENTS.md`. Most agent tools read `AGENTS.md` directly. Some read a
file of their own in its place whenever one exists, in the project or in a
folder above it, so `AGENTS.md` stays hidden from that tool; setup warns about
each such file it finds and changes none of them.

A project set up earlier with the saved choice `docs.target` of `claude_md` or
`both` in `.abcd/config.json` still reads as set up, but `abcd ahoy install`
stops before changing anything and names that one setting. In each such
project:

1. Change the setting:

   ```sh
   abcd ahoy install --docs-target agents_md
   ```

   This moves the block out of `CLAUDE.md` and into `AGENTS.md`. Use
   `--docs-target skip` instead to take it out of both.
2. If a tool's own file only repeats `AGENTS.md` (a link to it, or a copy),
   setup offers to remove it, and removes it only when you answer `retire`.
3. If `CLAUDE.md` holds your own words, setup leaves it exactly as it is and
   warns that `AGENTS.md` stays hidden until you move those words into
   `AGENTS.md` and remove the file. Do that, then start a new agent session.
4. Commit the changed files.
