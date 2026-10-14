package guard

import (
	"path/filepath"
	"strings"
	"testing"
)

// Only an unquoted leading `~` is tilde expansion. `'~/n.sh'`, `"~/n.sh"`
// and `\~/n.sh` name a directory literally called `~` under the working
// directory, so a quoted tilde is read as that directory wherever the
// reading resolves a word: a run's or a source's operand, a write's target
// (a redirection, cp, mv, tee, ln), a cd or pushd operand, a startup
// variable's value and the words of a command string. An unquoted `~` is
// still the home.
func TestQuotedTildeIsALiteralDirectory(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"w/~/ok.sh": hazardLine,
		"w/~/b/ok":  hazardLine,
		"h/ok.sh":   "echo hi",
		"h/s.sh":    hazardLine,
	})
	r := readingRegistry(dir, filepath.Join(dir, "h"))
	cases := []scriptCase{
		// A known directory: the write lands under its `~`, and so does the run.
		{`cd {d}/w && mkdir '~' && printf hi > '~/n.sh' && bash '~/n.sh'`, VerdictBlock, scriptWrittenEntryID},
		{`cd {d}/w && printf hi > "~/n.sh" && bash "~/n.sh"`, VerdictBlock, scriptWrittenEntryID},
		{`cd {d}/w && printf hi > \~/n.sh && bash \~/n.sh`, VerdictBlock, scriptWrittenEntryID},
		{`cd {d}/w && printf hi > '~/n.sh' && bash ./'~/n.sh'`, VerdictBlock, scriptWrittenEntryID},
		{`cd {d}/w && printf hi > '~/n.sh' && source '~/n.sh'`, VerdictBlock, scriptWrittenEntryID},
		{`cd {d}/w && printf hi > '~/n.sh' && . "~/n.sh"`, VerdictBlock, scriptWrittenEntryID},
		{`cd {d}/w && printf hi > '~/n.sh' && chmod +x '~/n.sh' && '~/n.sh'`, VerdictBlock, scriptWrittenEntryID},
		{`cd {d}/w && cp /tmp/x '~/n.sh' && bash '~/n.sh'`, VerdictBlock, scriptWrittenEntryID},
		{`cd {d}/w && mv /tmp/x "~/n.sh" && bash "~/n.sh"`, VerdictBlock, scriptWrittenEntryID},
		{`cd {d}/w && echo hi | tee \~/n.sh && bash \~/n.sh`, VerdictBlock, scriptWrittenEntryID},
		{`cd {d}/w && ln -s /tmp/x '~/n.sh' && bash '~/n.sh'`, VerdictBlock, scriptWrittenEntryID},
		{`cd {d}/w && bash -c "printf hi > '~/n.sh' && bash '~/n.sh'"`, VerdictBlock, scriptWrittenEntryID},
		// An unknown directory: a write under its `~` is the same file a run
		// of the same name there reads.
		{`cd "$X" && printf hi > '~/n.sh' && bash ./'~/n.sh'`, VerdictBlock, scriptWrittenEntryID},
		{`cd "$X" && printf hi > '~/n.sh' && bash '~/n.sh'`, VerdictBlock, scriptWrittenEntryID},
		{`cd "$X" && printf hi > "~/n.sh" && bash "./~/n.sh"`, VerdictBlock, scriptWrittenEntryID},
		{`cd "$X" && printf hi > \~/n.sh && source \~/n.sh`, VerdictBlock, scriptWrittenEntryID},
		{`cd "$X" && cp /tmp/x '~/n.sh' && bash ./'~/n.sh'`, VerdictBlock, scriptWrittenEntryID},
		// The file a quoted tilde names is the literal one, not the home's.
		{`cd {d}/w && bash '~/ok.sh'`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/w && bash "~/ok.sh"`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/w && bash \~/ok.sh`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/w && source '~/ok.sh'`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/w && bash -c "bash '~/ok.sh'"`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/w && cd '~' && bash ok.sh`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/w && cd "~" && bash ok.sh`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/w && pushd \~ && bash ok.sh`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/w && BASH_ENV='~/ok.sh' bash -c true`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/w && export BASH_ENV="~/ok.sh" && bash -c true`, VerdictBlock, scriptHazardEntryID},
		// An unquoted tilde is the home, as before.
		{`cd {d}/w && bash ~/ok.sh`, VerdictAllow, ""},
		{`cd {d}/w && cd ~ && bash ok.sh`, VerdictAllow, ""},
		{`cd {d}/w && BASH_ENV=~/ok.sh bash -c true`, VerdictAllow, ""},
		{`bash ~/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/w && source ~/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`BASH_ENV=~/s.sh bash -c true`, VerdictBlock, scriptHazardEntryID},
		{`printf hi > ~/x.sh && bash ~/x.sh`, VerdictBlock, scriptWrittenEntryID},
		{`cd "$X" && printf hi > ~/x.sh && bash ~/x.sh`, VerdictBlock, scriptWrittenEntryID},
		{`cd {d}/w && cp /tmp/x ~/x.sh && bash ~/x.sh`, VerdictBlock, scriptWrittenEntryID},
	}
	for _, c := range cases {
		cmd := strings.ReplaceAll(c.cmd, "{d}", dir)
		d, err := r.Check(cmd)
		if err != nil {
			t.Errorf("Check(%q): %v", c.cmd, err)
			continue
		}
		if d.Verdict != c.want || d.EntryID != c.id {
			t.Errorf("%s\n  = %s/%s, want %s/%s\n  %s", c.cmd, d.Verdict, d.EntryID, c.want, c.id, d.Message)
		}
	}
}

// A colon list's assignment expands an unquoted `~` at its start and after
// each `:`, as bash and zsh assign it, so the entry is the home's directory;
// a quoted one stays a directory called `~`.
func TestColonListTildeIsExpandedWhereUnquoted(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"h/b/ok":   hazardLine,
		"h/f/p":    hazardLine,
		"w/~/c/ok": hazardLine,
		"w/~/g/p":  hazardLine,
		"w/~/f/p":  "echo hi",
		"h/g/p":    "echo hi",
	})
	r := readingRegistry(dir, filepath.Join(dir, "h"))
	cases := []scriptCase{
		{`cd {d}/w && PATH=~/b bash ok`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/w && PATH=~/b:$PATH bash ok`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/w && PATH=/nonexistent:~/b bash ok`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/w && export PATH=~/b && bash ok`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/w && PATH=~/b && printf hi > ~/b/ok && bash ok`, VerdictBlock, scriptWrittenEntryID},
		{`cd {d}/w && PATH='~/c' bash ok`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/w && FPATH=~/f zsh -c 'autoload p; p'`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/w && fpath=(~/f) && autoload p && p`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/w && FPATH='~/g' zsh -c 'autoload p; p'`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/w && fpath=("~/g") && autoload p && p`, VerdictBlock, scriptHazardEntryID},
		{`cd {d}/w && FPATH=~/g zsh -c 'autoload p; p'`, VerdictAllow, ""},
	}
	for _, c := range cases {
		cmd := strings.ReplaceAll(c.cmd, "{d}", dir)
		d, err := r.Check(cmd)
		if err != nil {
			t.Errorf("Check(%q): %v", c.cmd, err)
			continue
		}
		if d.Verdict != c.want || d.EntryID != c.id {
			t.Errorf("%s\n  = %s/%s, want %s/%s\n  %s", c.cmd, d.Verdict, d.EntryID, c.want, c.id, d.Message)
		}
	}
}
