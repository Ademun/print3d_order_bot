package file

import (
	"io"
	"log/slog"
	"os"
)

func closeFileStream() func(fs io.ReadCloser) {
	return func(fs io.ReadCloser) {
		if err := fs.Close(); err != nil {
			slog.Error("failed to close file stream", "err", err)
		}
	}
}

func closeFile() func(out *os.File) {
	return func(out *os.File) {
		if err := out.Close(); err != nil {
			slog.Error("failed to close file", "err", err)
		}
	}
}
