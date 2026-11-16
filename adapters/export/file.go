package export

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

func writeFile(path string, write func(w io.Writer) error) (err error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("создание %s: %w", path, err)
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	return write(f)
}

func formatNumber(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
