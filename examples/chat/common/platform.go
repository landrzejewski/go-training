package common

import (
	"os"
	"runtime"
)

// FileMode returns the appropriate file mode for the platform
func FileMode() os.FileMode {
	if runtime.GOOS == "windows" {
		return 0666 // Windows doesn't use Unix permissions
	}
	return 0644 // Unix-like systems
}

// DirMode returns the appropriate directory mode for the platform
func DirMode() os.FileMode {
	if runtime.GOOS == "windows" {
		return 0777 // Windows doesn't use Unix permissions
	}
	return 0755 // Unix-like systems
}
