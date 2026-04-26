// Package maildir provides filesystem helpers for Dovecot Maildir
// creation, soft-deletion and restoration.
package maildir

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"time"
)

// Create builds a Maildir hierarchy at path (cur, new, tmp), sets permissions
// to 700 and changes owner to vmail:vmail.
func Create(path string) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return fmt.Errorf("mkdir maildir: %w", err)
	}

	subdirs := []string{"cur", "new", "tmp"}
	for _, sub := range subdirs {
		if err := os.MkdirAll(filepath.Join(path, sub), 0700); err != nil {
			return fmt.Errorf("mkdir %s: %w", sub, err)
		}
	}

	// Set permissions
	if err := os.Chmod(path, 0700); err != nil {
		return fmt.Errorf("chmod maildir: %w", err)
	}
	for _, sub := range subdirs {
		if err := os.Chmod(filepath.Join(path, sub), 0700); err != nil {
			return fmt.Errorf("chmod %s: %w", sub, err)
		}
	}

	// Change owner to vmail:vmail
	u, err := user.Lookup("vmail")
	if err != nil {
		return fmt.Errorf("lookup vmail user: %w", err)
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return fmt.Errorf("parse vmail uid: %w", err)
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return fmt.Errorf("parse vmail gid: %w", err)
	}

	if err := os.Chown(path, uid, gid); err != nil {
		return fmt.Errorf("chown maildir: %w", err)
	}
	for _, sub := range subdirs {
		if err := os.Chown(filepath.Join(path, sub), uid, gid); err != nil {
			return fmt.Errorf("chown %s: %w", sub, err)
		}
	}

	return nil
}

// SoftDelete renames the Maildir to path.deleted-<timestamp>.
// It returns the new path or an error.
func SoftDelete(path string) (string, error) {
	newPath := path + ".deleted-" + time.Now().Format("20060102-150405")
	if err := os.Rename(path, newPath); err != nil {
		if os.IsNotExist(err) {
			// If the directory does not exist, we still consider the soft-delete
			// successful from a filesystem perspective.
			return newPath, nil
		}
		return "", fmt.Errorf("rename maildir: %w", err)
	}
	return newPath, nil
}

// Restore moves a soft-deleted Maildir back to its original path.
func Restore(deletedPath, originalPath string) error {
	if err := os.Rename(deletedPath, originalPath); err != nil {
		return fmt.Errorf("restore maildir: %w", err)
	}
	return nil
}

// FindDeleted looks for a soft-deleted Maildir matching the original path
// and returns the first match (e.g. /path/.deleted-20240102-150405).
func FindDeleted(originalPath string) (string, error) {
	matches, err := filepath.Glob(originalPath + ".deleted-*")
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no deleted maildir found for %s", originalPath)
	}
	return matches[0], nil
}
