package file

import (
	"fmt"
	"strings"
)

type FailedFile struct {
	Filename string
	Err      error
}

type ErrProcessingFiles struct {
	TotalFiles  int
	FailedFiles []FailedFile
}

func (e ErrProcessingFiles) Error() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d files processed. Got %d errors", e.TotalFiles, len(e.FailedFiles)))
	for _, ff := range e.FailedFiles {
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("%s: %s", ff.Filename, ff.Err.Error()))
	}
	return sb.String()
}
