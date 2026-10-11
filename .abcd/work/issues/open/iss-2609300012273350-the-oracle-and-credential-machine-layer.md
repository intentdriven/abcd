---
schema_version: 1
id: "iss-2609300012273350"
slug: "the-oracle-and-credential-machine-layer"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: integ23, from the review of keyRoutes (INFO)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/layered/layered.go"
remedy: "Waits on a facilitator ruling on whether abcd's machine layer may follow $HOME at all: the proposal is to refuse (read as absent, with a note) a machine layer whose home resolves inside the session's repository root, the tree a checkout controls, and to take the key-spending decision only from a home the account owns outside that root; grounds: Go's os.UserHomeDir documents that on Unix, including macOS, it returns the $HOME environment variable, so nothing below it distinguishes a home the session was given from the account's own, while every test and CI job sets a temporary HOME on purpose, which is why the choice between refusing a home inside the tree and keying on the account database's home is a ruling, not a fix"
---

The oracle and credential machine layer is whatever $HOME names: layered.RootsFor (internal/core/layered/layered.go:156) and credential.UserStore (internal/core/credential/store.go:92) both take os.UserHomeDir, which on Unix returns $HOME. An environment a repository can shape (a committed harness settings file whose env block sets HOME into the tree) would let a checkout supply the 'machine' ~/.abcd/config.json, and with it a keyed provider block and the machine routes that keyRoutes admits to spend a key, while the operating-system keychain still answers by the user's login session, so the repository's own route reaches the person's paid key through a machine layer it wrote. keyRoutes did not introduce this; it inherits the trust in $HOME.
