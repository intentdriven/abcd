---
id: adr-2610241022330872
slug: setup-edits-the-shell-settings-to-put
status: accepted
date: 2026-10-24
supersedes: null
superseded_by: null
related_intents: [itd-2610240919030303, itd-2610240823055889]
related_rfcs: []
related_adrs: [adr-2610050711287503]
---

# ADR-2610241022330872: Setup edits the shell settings to put abcd's folder on the PATH, on a yes

## Context

`abcd ahoy install` puts its PATH entry in `~/.local/bin` by default (iss-171). On many machines that folder is not on the PATH, and then `abcd` cannot be run by name. Until now the rule was script-first: setup printed the one-line `export` fix and never edited a shell profile, written in the brief's setup page and in the `/abcd:ahoy` command page.

On 2026-10-24 the product thinker had to paste the README's long install line by hand before setup could finish. The research behind itd-2610240919030303 found that installers disagree: some edit the shell settings themselves, some only print the line. The product thinker chose that setup asks, and then confirmed that this reverses the written rule. The design review named the costs the change has to meet: which file each shell reads, settings files that are links into a shared dotfiles folder, undoing the edit when abcd is removed, and the new terminal the edit needs before it takes effect.

The status line does not need the PATH: it runs the recorded absolute path of the entry (adr-2610050711287503). The edit is for the person typing `abcd`.

## Decision

When the folder abcd installs into is not on the PATH, `abcd ahoy install` asks a person at a terminal, once, whether to add one line to the startup file their login shell reads. The question names the file and the line. abcd edits the file only on a yes. Declined, or when abcd will not edit the file, the line is listed for the person to add.

The edit meets this bar (`internal/core/shellpath`):

- Only zsh and bash, on macOS and Linux. zsh gets `${ZDOTDIR:-$HOME}/.zshrc`, which every interactive zsh reads. bash on macOS gets the login file bash reads (the first of `.bash_profile`, `.bash_login` and `.profile`, else a new `.bash_profile`). bash on Linux gets `~/.bashrc`. A `ZDOTDIR` outside the home folder, or any other shell, gets the checklist line.
- Only a file abcd can read whole: a regular file, not a link, owned by the person, writable by no one else, and at most 256 KiB. A file that is a link is kept elsewhere, so abcd never edits it.
- Never the same folder twice. Settings that already add the folder are left alone: abcd's own marked line, or a line of the person's that sets the PATH to include the folder, in any file that shell reads at startup.
- One line, added at the end and marked: `export PATH="$HOME/.local/bin:$PATH" # added by abcd ahoy install`. The folder comes first, ahead of the PATH it had, so the abcd setup installed answers before an older copy in a system folder.
- A copy of the file goes into `~/.abcd.noindex/backups/` first, which keeps the newest ten copies of each file. No copy, no write. The file is replaced atomically with its mode kept, read back, and put back from the copy when it does not hold exactly what was written.
- The file and the line are recorded in `~/.abcd.noindex/shell-path.json`. `abcd ahoy uninstall` removes that line and nothing else, puts back the newline abcd added before it, and deletes a file abcd created for the line alone. A line the person has since edited is no longer abcd's, and uninstall leaves it in place.
- `--yes` never gives the yes, and the question is never asked off a terminal.

This supersedes the rule that abcd never edits a shell profile. adr-2610050711287503 is unchanged: the harness's user settings are still touched only for the status line.

## Alternatives Considered

- Ask, then edit under the bar above (chosen): a person who says yes is done in one step, and the edit can be found and undone.
- Keep printing the line and never edit (the old rule): no risk to the person's files, but every person whose PATH lacks the folder has to find the right file and paste the line. That is the step the product thinker hit.
- Edit without asking, as some installers do: fewer questions, but it changes a file the person owns without a consent that names the change, which the setup flow's person-only questions rule out.
- Edit through a link into a dotfiles folder: reaches the person's real file, but it changes a file shared with other machines, and uninstall on one machine would take the line out on every machine.

## Consequences

- A person at a terminal can put abcd on the PATH by answering one question, and still has to open a new terminal before `abcd` runs by name there; setup says so.
- abcd now writes to a file outside its own folders and the repository. That write has its own record, copies and read-back, and uninstall removes it.
- The already-added check reads text. It can take a line for one that adds the folder when it does not, which costs the person the offer but never adds a second line. A line that adds the folder only through a file it sources is not seen.
- Shells other than zsh and bash, and settings kept as links, still need the person to add the line. The checklist names it.
- The brief's setup page and the `/abcd:ahoy` command page now describe the offer instead of the never-edit rule.
