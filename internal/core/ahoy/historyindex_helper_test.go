package ahoy

// writeHistoryIndex seeds ~/.abcd.noindex/history/index.json for a test, through a
// fresh walk of the directory and without the history lock, so a test can call
// it from inside afterHistoryReloadHook while the lock is held. Production
// writes go through writeHistoryIndexIn, with the directory whose lock is held.
func writeHistoryIndex(idx *historyIndex) error {
	dir, err := historyDir(false)
	if err != nil {
		return err
	}
	defer dir.Close()
	return writeHistoryIndexIn(dir, idx)
}
