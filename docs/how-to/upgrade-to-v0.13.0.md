# Upgrade to v0.13.0

v0.13.0 is a breaking release. abcd's own folder in your home directory is
`~/.abcd.noindex`, a name the Mac's search indexer passes over, where earlier
releases used `~/.abcd`. abcd never moves the old folder for you. This guide
moves a machine across once, says what to do if both folders end up standing,
and brings each project abcd manages up to date.

A machine that never ran an earlier abcd has no `~/.abcd` and needs none of
this.

## What stops until you rename

Once you update the plugin, and while `~/.abcd` still stands, abcd stops
everything before it writes anything: every command, every hook and the status
line. In an agent session no tool runs at all, because the safety check blocks
every command, the rename included. So the rename is done in a plain Terminal
window, outside any agent session.

## Move the folder

Do this once on each machine.

1. Update the plugin with `/plugin update abcd`, or a binary in
   `~/.local/bin` with `abcd update`.
2. Open a plain Terminal window and rename the folder:

   ```sh
   mv ~/.abcd ~/.abcd.noindex
   ```

3. In the same window, reconnect the worktrees kept in it. git records each
   worktree's location in full, so until this runs, each repository lists its
   moved worktrees as prunable, and a prune would delete their links:

   ```sh
   for w in ~/.abcd.noindex/worktrees/*/*; do git -C "$w" worktree repair; done
   ```

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
   for w in ~/.abcd.noindex/worktrees/*/*; do git -C "$w" worktree repair; done
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
