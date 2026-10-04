# Upgrade to v0.13.0

v0.13.0 is a breaking release. abcd's own folder in your home directory is
`~/.abcd.noindex`, a name the Mac's search indexer passes over, where earlier
releases used `~/.abcd`. abcd never moves the old folder for you: while
`~/.abcd` stands, every abcd command and hook stops before it writes anything
and names the commands below. This guide moves a machine across once and then
brings each project abcd manages up to date.

A machine that never ran an earlier abcd has no `~/.abcd` and needs none of
this.

## Move the folder

Do this once on each machine.

1. Close every agent session that runs the abcd plugin, so no older hook writes
   `~/.abcd` again while you work.
2. Update abcd the way you installed it: `/plugin update abcd` for the plugin,
   `abcd update` for a binary in `~/.local/bin`, or the
   [install guide](install.md)'s one-liner.
3. In a terminal, rename the folder:

   ```sh
   mv ~/.abcd ~/.abcd.noindex
   ```

4. Reconnect the worktrees kept in it. git records each worktree's location in
   full, so until this runs, each repository lists its moved worktrees as
   prunable, and a prune would delete their links:

   ```sh
   for w in ~/.abcd.noindex/worktrees/*/*; do git -C "$w" worktree repair; done
   ```

5. Run abcd again, for example `abcd --version` in the terminal, or start a new
   agent session.

If both `~/.abcd` and `~/.abcd.noindex` exist, abcd moves neither and says so.
Keep the one you want under the name `~/.abcd.noindex`, take the other out of
your home folder, then run step 4.

## Refresh each managed project

The managed block in a project's conventions file names
`~/.abcd/rules.json` and `~/.abcd/trusted-roots` until abcd's setup runs in
that project again. In each project abcd manages:

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
