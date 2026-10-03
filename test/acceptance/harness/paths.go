//go:build acceptance

package harness

import (
	"path/filepath"
	"runtime"
)

// migrationsURL points golang-migrate at <repo root>/db/migrations. go test runs with the
// package directory as cwd, so the relative path used by the API itself does not work.
func migrationsURL() string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "..")
	return "file://" + filepath.ToSlash(filepath.Join(root, "db", "migrations"))
}
