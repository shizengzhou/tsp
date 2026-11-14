//go:build windows

// Package winproc reads the parent process chain. ConPTY resets the terminal
// environment on Windows, so the process tree is often the only way to tell
// which terminal hosts tsp.
package winproc

import (
	"os"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Process is one ancestor process.
type Process struct {
	// Name is the lower-case executable name without the .exe suffix.
	Name string
	// Path is the full executable path, empty when it cannot be read.
	Path string
}

var (
	once      sync.Once
	ancestors []Process
)

// Ancestors returns the parent processes of the current process, nearest
// first, up to 8 levels. The result is computed once per process.
func Ancestors() []Process {
	once.Do(func() { ancestors = findAncestors() })
	return ancestors
}

func findAncestors() []Process {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer windows.CloseHandle(snapshot)

	parents := make(map[uint32]uint32)
	names := make(map[uint32]string)
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		parents[entry.ProcessID] = entry.ParentProcessID
		names[entry.ProcessID] = windows.UTF16ToString(entry.ExeFile[:])
	}

	var out []Process
	pid := uint32(os.Getpid())
	for range 8 {
		parent, ok := parents[pid]
		if !ok || parent == 0 || parent == pid {
			break
		}
		pid = parent
		name := strings.ToLower(strings.TrimSuffix(names[pid], ".exe"))
		if name == "" {
			break
		}
		out = append(out, Process{Name: name, Path: imagePath(pid)})
	}
	return out
}

// imagePath returns the full path of pid, or "" when it cannot be opened.
func imagePath(pid uint32) string {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(handle)

	buf := make([]uint16, windows.MAX_LONG_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(handle, 0, &buf[0], &size); err != nil {
		return ""
	}
	return windows.UTF16ToString(buf[:size])
}
