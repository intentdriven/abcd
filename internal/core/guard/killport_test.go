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
