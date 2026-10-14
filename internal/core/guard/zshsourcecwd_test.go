package guard

import "testing"

// TestZshSourceInAnUnknownDirectory: zsh's `source` of a bare name looks in
// the working directory before PATH. Where the line leaves that directory
// unknown, the file there is unread, as a `./name` source there is, and is
// not refused for that alone, so the ordinary `cd "$(…)" && source name`
// lines allow in whichever shell the guard reads them; PATH's file, which
// zsh runs when the directory does not hold the name, is still read, and a
// file the line writes before the source is still refused (f07e31704), as a
// write it cannot place warns.
func TestZshSourceInAnUnknownDirectory(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"p/lib.sh": "echo benign",
		"h/lib.sh": hazardLine,
		"w/lib.sh": "echo benign",
	})
	runScriptCases(t, dir, []scriptCase{
		// Ordinary lines, read as bash and as zsh.
		{`cd "$(git rev-parse --show-toplevel)" && source .envrc`, VerdictAllow, ""},
		{`cd "$(mktemp -d)" && source setup.sh`, VerdictAllow, ""},
		{`pushd "$(dirname "$0")" && source env.sh && popd`, VerdictAllow, ""},
		{`zsh -c 'cd $X; PATH={d}/p; source lib.sh'`, VerdictAllow, ""},
		{`zsh -c 'cd $X && source lib.sh'`, VerdictAllow, ""},
		{`cd {d}/w && zsh -c 'PATH={d}/p; source lib.sh'`, VerdictAllow, ""},
		{`zsh -c 'cd $X; PATH={d}/p; . lib.sh'`, VerdictAllow, ""},
		{`zsh -c 'cd $X; source {d}/w/lib.sh'`, VerdictAllow, ""},
		{`bash -c 'cd $X; PATH={d}/p; source lib.sh'`, VerdictAllow, ""},

		// PATH's file is still read, and a write before the source still
		// counts.
		{`zsh -c 'cd $X; PATH={d}/h; source lib.sh'`, VerdictBlock, scriptHazardEntryID},
		{`zsh -c 'cd {d}/w && printf hi > n.sh && source n.sh'`, VerdictBlock, scriptWrittenEntryID},
		{`zsh -c 'cd $X; printf hi > n.sh; source n.sh'`, VerdictWarn, scriptWrittenEntryID},
		{`cd "$(mktemp -d)" && printf hi > n.sh && source n.sh`, VerdictWarn, scriptWrittenEntryID},

		// A write the guard places, then a source in a directory the line
		// leaves unknown: the file that runs may be the one written.
		{`cp {d}/h/lib.sh {d}/z.sh; cd "$(echo {d})" && source z.sh`, VerdictWarn, scriptWrittenEntryID},
		{`zsh -c 'printf x > {d}/w/n.sh; cd $X; source n.sh'`, VerdictWarn, scriptWrittenEntryID},
		{`zsh -c 'printf x > {d}/w/n.sh; cd $X; PATH={d}/p; source lib.sh'`, VerdictWarn, scriptWrittenEntryID},
		{`zsh -c 'printf x > {d}/w/n.sh; cd $X; PATH={d}/h; source lib.sh'`, VerdictBlock, scriptHazardEntryID},
	})
}

// TestARunInAnUnknownDirectoryAfterAPlacedWrite: every other form that runs
// a file by a relative name where the line leaves the directory unknown —
// `./name`, as a source or run directly, bash's search for a bare name, and a
// shell handed the file — warns when the line writes a file the guard places
// before it, since the file that runs may be the one written. With no write
// before it, each form still allows unread.
func TestARunInAnUnknownDirectoryAfterAPlacedWrite(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"w/keep": "",
	})
	runScriptCases(t, dir, []scriptCase{
		{`printf x > {d}/w/n.sh; cd "$(mktemp -d)" && source ./n.sh`, VerdictWarn, scriptWrittenEntryID},
		{`printf x > {d}/w/n.sh; cd "$(mktemp -d)" && . ./n.sh`, VerdictWarn, scriptWrittenEntryID},
		{`printf x > {d}/w/n.sh; cd "$(mktemp -d)" && ./n.sh`, VerdictWarn, scriptWrittenEntryID},
		{`printf x > {d}/w/n.sh; cd "$(mktemp -d)" && sub/n.sh`, VerdictWarn, scriptWrittenEntryID},
		{`bash -c 'printf x > {d}/w/n.sh; cd $X; source n.sh'`, VerdictWarn, scriptWrittenEntryID},
		{`bash -c 'printf x > {d}/w/n.sh; cd $X; . n.sh'`, VerdictWarn, scriptWrittenEntryID},
		{`printf x > {d}/w/n.sh; cd "$(mktemp -d)" && bash n.sh`, VerdictWarn, scriptWrittenEntryID},
		{`printf x > {d}/w/n.sh; cd "$(mktemp -d)" && sh ./n.sh`, VerdictWarn, scriptWrittenEntryID},
		{`printf x > {d}/w/n.sh; cd "$(mktemp -d)" && zsh n.sh`, VerdictWarn, scriptWrittenEntryID},

		// A write relative to the unknown directory itself is one the run
		// may reach, and so is a write the guard places beside a file PATH
		// supplies.
		{`cd "$(mktemp -d)" && printf hi > n.sh && source ./n.sh`, VerdictWarn, scriptWrittenEntryID},
		{`cd "$(mktemp -d)" && printf hi > n.sh && ./n.sh`, VerdictWarn, scriptWrittenEntryID},
		{`cd "$(mktemp -d)" && printf hi > n.sh && bash n.sh`, VerdictWarn, scriptWrittenEntryID},
		{`cd "$(mktemp -d)" && tar -xf {d}/w/keep && sh n.sh`, VerdictWarn, scriptWrittenEntryID},
		{`zsh -c 'cd $X; printf hi > $Y; PATH={d}/w; source keep'`, VerdictWarn, scriptWrittenEntryID},

		// No write before the run, or a run by an absolute path or in a
		// directory the line fixes: unchanged.
		{`cd "$(mktemp -d)" && source ./n.sh`, VerdictAllow, ""},
		{`cd "$(mktemp -d)" && ./configure`, VerdictAllow, ""},
		{`bash -c 'cd $X; source n.sh'`, VerdictAllow, ""},
		{`cd "$(mktemp -d)" && bash n.sh`, VerdictAllow, ""},
		{`cd "$(mktemp -d)" && sh ./n.sh`, VerdictAllow, ""},
		{`printf x > {d}/w/n.sh; cd "$(mktemp -d)" && bash {d}/w/keep`, VerdictAllow, ""},
		{`printf x > {d}/w/n.sh; cd {d}/w && bash keep`, VerdictAllow, ""},
		{`printf x 2>/dev/null >&2; cd "$(mktemp -d)" && bash n.sh`, VerdictAllow, ""},
		{`printf x > "$T/n.sh"; cd "$(mktemp -d)" && bash n.sh`, VerdictAllow, ""},
	})
}
