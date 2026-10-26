---
id: dcn-2610240845548776
slug: tool-installs-default-to-a-verified
date: 2026-10-24
---
2026-10-24 — Tool installs default to a verified download, on macOS and Linux alike: abcd fetches the tool's official release, checks it against the release's published checksums (and its signed build record where the project publishes one), and puts it in ~/.local/bin, needing no admin rights; this is the route abcd already uses to install itself. Homebrew is not the default, because a supported Homebrew for one non-admin account cannot be created by abcd on a Mac (Homebrew's installer refuses a prefix longer than /opt/homebrew, so no home-folder prefix fits) and a shared Homebrew is often not writable by the person. Decided by the product thinker on 2026-10-24 after the Homebrew-without-admin research (iss-2610240824571059).
