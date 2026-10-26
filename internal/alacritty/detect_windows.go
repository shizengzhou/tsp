//go:build windows

package alacritty

import (
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// alacrittyAncestor reports whether alacritty.exe is one of our parent
// processes. Alacritty on Windows hosts the shell through ConPTY, which resets
// TERM to xterm-256color and leaves TERM_PROGRAM and the ALACRITTY_* variables
// unset, so the environment alone cannot identify the terminal.
func alacrittyAncestor() bool {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(snapshot)

	parents := make(map[uint32]uint32)
	names := make(map[uint32]string)
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		parents[entry.ProcessID] = entry.ParentProcessID
		names[entry.ProcessID] = windows.UTF16ToString(entry.ExeFile[:])
	}

	pid := uint32(os.Getpid())
	for range 8 {
		parent, ok := parents[pid]
		if !ok || parent == 0 || parent == pid {
			return false
		}
		pid = parent
		name := strings.ToLower(strings.TrimSuffix(names[pid], ".exe"))
		if name == "alacritty" {
			return true
		}
	}
	return false
}
