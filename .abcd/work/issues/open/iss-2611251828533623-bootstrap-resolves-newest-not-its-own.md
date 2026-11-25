---
schema_version: 1
id: "iss-2611251828533623"
slug: "bootstrap-resolves-newest-not-its-own"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "manual build of 25 November, logged as a test of what abcd relies on"
origin: researcher-authored
production_mode: hand-written
found_at: "hooks/bootstrap.sh"
remedy: "Provision the release the version in the plugin.json of the root names, with the download cache keyed by tag; adr-2611251808065076 asks for this request to be named or removed. Grounds: the manifest is in the root, and the marketplace already pins the archive by version."
---

The bootstrap that gives a fresh plugin root its binary resolves the newest release, not the one its own plugin version names: hooks/bootstrap.sh reads the redirect of the releases page and pins its download to that tag. adr-38 says provisioning fetches exactly the artifact a plugin update names and never discovers one, so a plugin at one version can be given the binary of another, by a request a hook makes unasked. Read in the script on 2026-11-25; the update chapter of the brief describes the same resolve.
