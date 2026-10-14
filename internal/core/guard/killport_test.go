package guard

import "testing"

// TestKillPortIsStoppingByHolder — iss-2610112232319827, ruling AS (product
// thinker, 2026-10-11): `kill-port` stops every process that holds the port it
// names, as `fuser -k` does, so it blocks with the same successor. npx, `npm
// exec`, `pnpm dlx` and `yarn dlx` run the program their first operand names
// (a package spec's version left aside), so the guard reads through them as a
// wrapper.
func TestKillPortIsStoppingByHolder(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`npx kill-port 3000`, VerdictBlock, "kill-port"},
		{`npx kill-port 3000 8080`, VerdictBlock, "kill-port"},
		{`npx -y kill-port 3000`, VerdictBlock, "kill-port"},
		{`npx --yes kill-port --port 3000`, VerdictBlock, "kill-port"},
		{`npx -p kill-port kill-port 3000`, VerdictBlock, "kill-port"},
		{`npx kill-port@1.6.1 3000`, VerdictBlock, "kill-port"},
		{`npx kill-port@latest 3000`, VerdictBlock, "kill-port"},
		{`npx -- kill-port 3000`, VerdictBlock, "kill-port"},
		{`npm exec kill-port 3000`, VerdictBlock, "kill-port"},
		{`npm exec -- kill-port 3000`, VerdictBlock, "kill-port"},
		{`npm x kill-port 3000`, VerdictBlock, "kill-port"},
		{`npm exec --yes -- kill-port@1 3000`, VerdictBlock, "kill-port"},
		{`pnpm dlx kill-port 3000`, VerdictBlock, "kill-port"},
		{`yarn dlx kill-port 3000`, VerdictBlock, "kill-port"},
		{`yarn dlx -p kill-port kill-port 3000`, VerdictBlock, "kill-port"},
		{`kill-port 3000`, VerdictBlock, "kill-port"},
		{`./node_modules/.bin/kill-port 3000`, VerdictBlock, "kill-port"},
		// The runners read through as a wrapper, so a blocker behind one is
		// read too.
		{`npx pkill make`, VerdictBlock, "pkill-by-pattern"},
		{`npm exec -- ` + hazardLine, VerdictBlock, "git-push-force"},

		// Running another package, installing kill-port, and looking.
		{`npx prettier --check .`, VerdictAllow, ""},
		{`npx --yes create-vite@latest app`, VerdictAllow, ""},
		{`npm exec eslint .`, VerdictAllow, ""},
		{`pnpm dlx create-vite app`, VerdictAllow, ""},
		{`yarn dlx cowsay hi`, VerdictAllow, ""},
		{`npm run build`, VerdictAllow, ""},
		{`npx --version`, VerdictAllow, ""},
		{`npm install --save-dev kill-port`, VerdictAllow, ""},
		{`npm execute-thing kill-port`, VerdictAllow, ""},
		{`pnpm install kill-port`, VerdictAllow, ""},
		{`yarn add kill-port`, VerdictAllow, ""},
		{`lsof -i :3000`, VerdictAllow, ""},
		{`abcd capture "npx kill-port 3000 stopped a peer's server"`, VerdictAllow, ""},
	})
}

// TestARepoEntryNamingARunnerStillFires — reading npx, `npm exec`, `pnpm dlx`
// and `yarn dlx` as wrappers must not take the runner word itself away from a
// repository's own entries. Before the runners were read as wrappers a
// committed .abcd/guard.json entry whose command is npx (or npm, pnpm or yarn)
// blocked `npx some-pkg`; it must still, or the read changes how that
// repository's guard works.
func TestARepoEntryNamingARunnerStillFires(t *testing.T) {
	root := writeOverride(t, `{"schema_version":1,"entries":{
		"no-npx":{"tier":"blocker","pattern":{"command":"npx"},"why":"this repository pins its tools","successor":"run the pinned script"},
		"no-npm-exec":{"tier":"blocker","pattern":{"command":"npm","subcommand":"exec"},"why":"this repository pins its tools","successor":"run the pinned script"},
		"no-pnpm":{"tier":"blocker","pattern":{"command":"pnpm"},"why":"this repository uses npm","successor":"use npm"},
		"no-yarn-dlx":{"tier":"blocker","pattern":{"command":"yarn","subcommand":"dlx"},"why":"this repository pins its tools","successor":"run the pinned script"}}}`)
	r, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ cmd, entry string }{
		{`npx some-pkg`, "no-npx"},
		{`npx --yes some-pkg@1 --flag`, "no-npx"},
		{`npm exec some-pkg`, "no-npm-exec"},
		{`pnpm dlx some-pkg`, "no-pnpm"},
		{`yarn dlx some-pkg`, "no-yarn-dlx"},
	} {
		t.Run(tc.cmd, func(t *testing.T) {
			d, err := r.Check(tc.cmd)
			if err != nil {
				t.Fatal(err)
			}
			if d.Verdict != VerdictBlock || d.EntryID != tc.entry {
				t.Errorf("Check(%q) = %q via %q, want %q via %q", tc.cmd, d.Verdict, d.EntryID, VerdictBlock, tc.entry)
			}
		})
	}
	// The bundled reading behind the runner still holds beside it.
	d, err := r.Check(`npx kill-port 3000`)
	if err != nil {
		t.Fatal(err)
	}
	if d.Verdict != VerdictBlock {
		t.Errorf("Check(npx kill-port 3000) = %q, want %q", d.Verdict, VerdictBlock)
	}
}
