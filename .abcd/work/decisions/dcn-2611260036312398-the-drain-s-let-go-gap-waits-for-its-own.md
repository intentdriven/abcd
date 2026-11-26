---
id: dcn-2611260036312398
slug: the-drain-s-let-go-gap-waits-for-its-own
date: 2026-11-26
---
2026-11-26 — The drain's let-go gap waits for its own pull request, made straight after the first drain-bundles pull request (itd-2611201450052326, spc-2611201455051897 step 4) lands. The gap: a move that dies after it writes the run and before it writes the drain state leaves that state recording holds the new run now carries, and a drive's let-go that runs before the next move opens the lane releases them plainly. It is in the single-lane judgement hold on the default branch and in a forming bundle, where it reaches up to bundle_max holds. It waits because it releases only this account's own holds, it needs the person's own process to die in that window, and the land fence refuses the run's landing afterwards, so nothing lands wrongly. The fix is one change in letGo that covers both paths, with finish writing the drain state before it releases any hold over the network, and a kill-point test of its own.
