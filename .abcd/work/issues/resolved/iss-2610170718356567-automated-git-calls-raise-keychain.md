---
schema_version: 1
id: "iss-2610170718356567"
slug: "automated-git-calls-raise-keychain"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "conductor-intent session, keychain prompt at 08:12:45 on 2026-10-17"
origin: researcher-authored
production_mode: hand-written
found_at: "git calls run with a temporary HOME or isolated global config (tests, internal/gitutil)"
remedy: "Make every git call abcd and its tests make that could reach a remote unable to prompt the person: tests never reach a real forge (local fixture remotes only, with a guard that fails a test naming a network URL), and any git call run without the person's own config also skips the system config's credential helpers (GIT_CONFIG_NOSYSTEM=1 or credential.helper reset to empty, with GIT_TERMINAL_PROMPT=0 and GCM_INTERACTIVE=never), so a missing credential is a named failure, never an OS dialog."
resolution: "Under a test binary abcd's networked git refuses every non-local transport, clears every credential helper and drops an inherited GIT_ALLOW_PROTOCOL, so a test can no longer reach a real forge or raise the keychain dialog; the cloud script tests, which run a built binary, are held to local transports by their environment. Production behaviour is unchanged."
impact: internal
resolved_by:
  commit: "c1abf9014"
---

An automated git call in an abcd-managed checkout raised a macOS keychain prompt asking the person for their github.com credentials, which confused them: nothing they had started asked for credentials. At 08:12:45 on 2026-10-17 the system log shows git-credential-osxkeychain (Homebrew git 2.52.0) asking for the github.com keychain item under the account all abcd sessions run as; the person denied it. Their own global git config sends github.com credentials to the gh helper and clears the keychain helper, so the call ran without that config (a temporary HOME, as tests set, or an isolated global config) while still reading the system gitconfig, which names osxkeychain. The likeliest source is a test run in a peer's worktree that reached https://github.com; the exact test is not yet confirmed. A person who learns to approve such prompts would hand a GitHub credential to a test or a background process.

## Mechanism

Traced in code by a peer session on 2026-10-17, the same morning:

- abcd's networked git runs under `ScrubbedEnv`. That includes `gitutil.Net`, `NetEach` and `NetInput`, the claim remote and status code, and `gitutil.RemoteHeadBranch`.
- `ScrubbedEnv` strips every `GIT_CONFIG_*` variable. A test's `GIT_CONFIG_NOSYSTEM` or `GIT_CONFIG_GLOBAL` therefore never reaches the call, and the system gitconfig still applies, with its `credential.helper=osxkeychain`.
- With `HOME` pointed at a temporary directory, the person's own global config, with its gh helper and keychain reset, is not read.
- Against a `https://github.com/<owner>/<repo>` that does not exist, GitHub answers 401. git then asks the keychain helper, which raises the dialog. A public repository answers anonymously and would not prompt.

At 08:12:45 several test runs were live across sessions: two in one peer's worktrees (the claim, loop and cloudscripts packages among them), and others' claim, cli and loop runs. Candidate tests that drive production code against a github.com origin:

- `cloud_handback_test.go`: origin `https://github.com/acme/widgets.git`, relying on a repository-local `insteadOf` that a ScrubbedEnv path without that config would miss;
- `cloud_brief_test.go`;
- loop `configwatch_test.go`: a `url.*.insteadOf` carrying a token.

The prompt can recur on any machine whose system gitconfig names a credential helper, whenever these packages run, preflight included.

## Grounds

- pursued: git refuses a transport its protocol policy disallows before connecting, and an empty credential.helper in command scope clears every helper a config file names; shown wrong if a test under the guard raises a credential prompt or reaches a non-local remote
