//go:build unix

package fileutil

import (
	"os"
)

func syncDir(dir string) error {
	df, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer df.Close()

	if err := df.Sync(); err != nil {
		return err
	}

	return nil
}
