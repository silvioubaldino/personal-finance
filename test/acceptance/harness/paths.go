//go:build acceptance

package harness

import (
	"path/filepath"
	"runtime"
)

func migrationsURL() string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "..")
	return "file://" + filepath.ToSlash(filepath.Join(root, "db", "migrations"))
}
