package services

import "golang.org/x/sys/windows"

// openWithDefaultApp opens path in the application Windows associates with its extension. It
// calls ShellExecute directly: going through cmd.exe would let characters such as "&" in a
// file name run as commands.
func openWithDefaultApp(path string) error {
	file, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	return windows.ShellExecute(0, nil, file, nil, nil, windows.SW_SHOWNORMAL)
}
