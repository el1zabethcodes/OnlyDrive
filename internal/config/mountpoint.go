package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// DefaultMountRoot is the directory where per-account mountpoints live on
// macOS and Linux (the mount backends take a path, not a drive letter).
// GD_HOME overrides the root so tests and portable installs keep mounts
// inside their sandbox; the default is ~/.gd/mnt.
func DefaultMountRoot() string {
	if base := os.Getenv("GD_HOME"); base != "" {
		return filepath.Join(base, "mnt")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), AppName, "mnt")
	}
	return filepath.Join(home, "."+AppName, "mnt")
}

// MountTarget returns the target for the next mount: a free drive letter on
// Windows, a per-account directory under DefaultMountRoot elsewhere (e.g.
// ~/.gd/mnt/acc1). The directory is created for non-Windows targets.
func MountTarget(name string) (string, error) {
	if runtime.GOOS == "windows" {
		return NextFreeLetter()
	}
	mp := filepath.Join(DefaultMountRoot(), name)
	if err := os.MkdirAll(mp, 0o755); err != nil {
		return "", fmt.Errorf("mount point '%s' is already in use or inaccessible: %w", mp, err)
	}
	if err := ValidateMountTarget(mp); err != nil {
		return "", err
	}
	return mp, nil
}

// ValidateMountTarget reports whether a mount target is usable before rclone
// touches it. A failed rclone mount on a busy or malformed target is a
// generic error that sends users debugging the mount backend; these errors
// name the target as the problem and say what to do about it.
func ValidateMountTarget(target string) error {
	if target == "" {
		return fmt.Errorf("mount point is empty (run: gd mount to pick one)")
	}
	if IsDriveLetter(target) {
		if _, err := os.Stat(target + `\`); err == nil {
			return fmt.Errorf("mount point '%s' is already in use or inaccessible (free it with: gd unmount, or remove the drive using the letter)", target)
		}
		return nil
	}
	if !filepath.IsAbs(target) {
		return fmt.Errorf("mount point '%s' is not an absolute path (use a full path like %s)", target, filepath.Join(DefaultMountRoot(), "acc1"))
	}
	if fi, err := os.Stat(target); err == nil && !fi.IsDir() {
		return fmt.Errorf("mount point '%s' is already in use or inaccessible (a file occupies it)", target)
	}
	return nil
}

// IsDriveLetter reports whether a stored mount target is a Windows drive
// letter ("X:") as opposed to a filesystem path (macOS/Linux mountpoint).
// Purely lexical so stored configs stay readable on any platform.
func IsDriveLetter(target string) bool {
	return len(target) == 2 && target[0] >= 'A' && target[0] <= 'Z' && target[1] == ':'
}
