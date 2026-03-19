//go:build windows

package fileutil

func syncDir(_ string) error {
	// Directory fsync is not portable on Windows in this foundational layer.
	return nil
}
