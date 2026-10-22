---
id: dcn-2610222030477927
slug: abcd-looks-for-the-person-s-forge-login
date: 2026-10-22
---
2026-10-22 — abcd looks for the person's forge login before it starts an unattended run, and refuses to start where it finds one (facilitator, on state-of-the-art grounds, applying the product thinker's 2026-10-22 ruling C1 on itd-2610170634394993): before an away-mode run, build or drain starts unattended, abcd checks the places a forge login is kept on that machine for the forge host (the forge CLI's stored login, the token environment variables the forge CLI and git read, and git's credential helper asked for the host), reading only whether each answers and never the token itself, and refuses to start, naming each place it found one; it does not try to remove or hide a login, since the run belongs where none was ever saved (a cloud session, a container or a separate user account). A login the check cannot see is out of its reach, so the guarantee rests on where the run is started, which abcd states as the person's setup step (grounds: cli.github.com/manual/gh_help_environment; git-scm.com/docs/gitcredentials; adr-2610222026262980 rule 4)
