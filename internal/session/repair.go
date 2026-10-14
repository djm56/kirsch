package session

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// repairSession truncates f at the end of the last valid JSONL line.
//
// It walks the file from the beginning, tracking the byte offset after each
// valid line. If the final line is corrupt or torn, the file is truncated to
// the last good offset and the caller is informed that recovery happened.
// Errors are only returned for underlying I/O problems; parse problems are
// repaired silently so loading a session never fails startup because of one
// bad file.
func repairSession(f *os.File) (bool, error) {
	info, err := f.Stat()
	if err != nil {
		return false, fmt.Errorf("stat session file: %w", err)
	}
	size := info.Size()
	if size == 0 {
		return false, nil
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return false, fmt.Errorf("seek session file: %w", err)
	}

	r := bufio.NewReader(f)
	var offset, lastGood int64
	for {
		line, err := r.ReadBytes('\n')
		n := int64(len(line))
		if n > 0 {
			trimmed := bytes.TrimSpace(line)
			if len(trimmed) == 0 || json.Valid(trimmed) {
				lastGood = offset + n
			}
		}
		offset += n
		if err == io.EOF {
			break
		}
		if err != nil {
			return false, fmt.Errorf("read session file: %w", err)
		}
	}

	if lastGood >= size {
		return false, nil
	}

	if err := f.Truncate(lastGood); err != nil {
		return false, fmt.Errorf("truncate session file: %w", err)
	}
	if err := f.Sync(); err != nil {
		return false, fmt.Errorf("sync session file: %w", err)
	}
	return true, nil
}
