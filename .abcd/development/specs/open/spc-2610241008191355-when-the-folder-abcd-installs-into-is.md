---
id: spc-2610241008191355
slug: when-the-folder-abcd-installs-into-is
intent: itd-2610240919030303
origin: researcher-authored
production_mode: dictated-and-formatted
---
# when-the-folder-abcd-installs-into-is

## Summary

spc-2610241008191355 delivers itd-2610240919030303: when the folder abcd installs into is not on the PATH, `abcd ahoy install` asks a person at a terminal, once, whether to add one marked line to the startup file their login shell reads. It edits the file only on a yes, never adds the line twice, and `abcd ahoy uninstall` removes it again. Declined, or when abcd will not edit the file, the line is on the checklist. adr-2610241022330872 records the reversal of the rule that abcd never edits a shell profile.

### Scope

zsh and bash, on macOS and Linux (the intent's two scope conditions). Any other shell, and a `ZDOTDIR` outside the home folder, get the checklist line. The end-of-run checklist itself belongs to the setup flow (itd-2610240823055889). This spec gives that flow the core API and, until the flow lands, carries the line on the install's notes.

### Approach

- `internal/core/shellpath` holds every decision and the edit, and prints nothing. `PlanFor(env, dir)` reads only, and returns the state (`on_path`, `already_added`, `offer`, `checklist`), the file, the marked line, the unmarked export line and the reason. `Apply(env, plan)` makes the edit on a yes, `Remove(env, dir)` undoes it, and `ChecklistLine(home, dir)` gives the line for the checklist.
- File per shell: zsh `${ZDOTDIR:-$HOME}/.zshrc` (every interactive zsh reads it; `.zprofile` is read by login shells only). bash on macOS: the first of `.bash_profile`, `.bash_login` and `.profile`, else a new `.bash_profile`, because a macOS terminal opens a login shell and a new `.bash_profile` would hide an existing `.profile`. bash on Linux: `~/.bashrc`.
- The line is `export PATH="$HOME/.local/bin:$PATH" # added by abcd ahoy install`. The folder comes first so the installed abcd answers before an older copy in a system folder.
- Safety bar: a regular file that is not a link, owned by the person, writable by no one else, at most 256 KiB (`fsutil.ReadGuarded`); re-judged and re-read at Apply, and refused if it changed since the question. A copy goes into `~/.abcd.noindex/backups/shell-<file>.<stamp>` first (newest ten kept; no copy, no write). Then an atomic replace that keeps the mode, a read-back, and the file put back from the copy on a mismatch. The record goes in `~/.abcd.noindex/shell-path.json`.
- ahoy: `detectBinDirOnPath` keeps `path.bin_dir_not_on_path` as the required, non-resolvable diagnosis and adds the optional offer `path.shell_settings_offered` (category `shell-path`, never asked as a category approval). `stepShellPath` runs right after `stepPathEntry`, so a first install is offered too. It asks through `Prompter.Confirm` only at a terminal and never under `--yes`. `noteReachability` then says to open a new terminal, that the settings already add the folder, or gives the line with why abcd did not offer it. `Uninstall` carries `shell_path` on its receipt.

### Acceptance criteria

1. Not on the PATH, setup at a terminal: the offer is asked once, naming the file and the line. On a yes a copy is kept (when the file existed), one line is added, and the notes say to open a new terminal. Tests: `TestShellPathOfferAddsTheLineOnYes`, `TestOfferAddsOneLineAndRemoveTakesOnlyIt`.
2. Settings already add the folder: nothing is asked or edited (`TestShellPathAlreadyAddedDoesNothing`, `TestAlreadyAddedByThePerson`). The settings file is a link kept elsewhere: nothing is asked or edited, and the line is on the checklist with the reason (`TestShellPathLinkedFileIsNotEdited`, `TestLinkedFileGoesOnTheChecklist`).
3. A no: the line is on the checklist (`TestShellPathDeclinedPutsTheLineOnTheChecklist`; `--yes` and off a terminal in `TestShellPathNeverAskedWithoutAPerson`). Removal takes out abcd's line and nothing else, byte for byte, including the newline it added and a file it created (`TestRemoveRestoresAFileWithNoFinalNewline`, `TestCreatedFileIsRemovedWithTheLine`, `TestRemoveWithoutRecordFindsTheMarkedLine`, and the uninstall half of `TestShellPathOfferAddsTheLineOnYes`).

## Footprint

- packages: internal/core/shellpath, internal/core/ahoy, internal/surface/cli
- tests: internal/core/shellpath/shellpath_test.go, internal/core/ahoy/shell_path_test.go, internal/core/ahoy/bin_install_test.go

## Steps

_No steps listed: the spec is built as one step. To split it, list the steps in order as `1. <title>`, each with `- packages:` and `- tests:` indented beneath it; `- landed: <pull request or commit>` marks a step that has landed._
