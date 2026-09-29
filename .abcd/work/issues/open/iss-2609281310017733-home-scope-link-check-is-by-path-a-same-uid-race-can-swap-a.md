---
schema_version: 1
id: "iss-2609281310017733"
slug: "home-scope-link-check-is-by-path-a-same-uid-race-can-swap-a"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/home.go"
---

The symlinked ~/.abcd rule is enforced by path, not by descriptor, so a same-uid race can still slip a link in between the check and the use. fsutil.HomeScopeLink Lstats each directory below home (internal/fsutil/home.go), and the reader then opens the file by its full path (ReadHomeDeclaration, then ReadDeclaration's own Lstat, open and SameFile), while the writers check and then MkdirAll, lock or create by path (credential.SetMachine, oracle writeProviderBlock, ahoy writePathEntry, writeMachineRouting, wireStatusLine, and the history registry's EnsureRealDirAll after historyRoot). A process running as the same uid that swaps ~/.abcd for a symlink after the Lstat reads or writes through the link. It is the same residual the rules loader's Lstat carried, and it needs the caller's own uid, so it widens nothing an attacker at that uid could not do directly; it is recorded so the rule's guarantee is stated at the strength it actually has. os.Root would not close it: it follows a symlink that stays inside the root, and a dotfiles ~/.abcd usually points inside home. The airtight form uses only the standard library (syscall on darwin and linux, no new dependency): open home, then syscall.Openat(homefd, ".abcd", O_DIRECTORY|O_NOFOLLOW), then Openat(dirfd, leaf, O_NOFOLLOW) (O_CREAT with O_EXCL or O_NOFOLLOW for writers, Mkdirat for a missing level), and judge and read or write through those descriptors. Left open by the fix round that routed every reader and writer through HomeScopeLink (branch fix/drain-symlinked-home); not built there.
