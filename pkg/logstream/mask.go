package logstream

import (
	"io"
	"strings"
)

const (
	maskedStr = "****************"
)

// masker wraps a stream writer with a masker
type masker struct {
	w io.Writer
	r *strings.Replacer
}

// NewMasker returns a masker that wraps io.Writer w.
func NewMasker(w io.Writer, secretData map[string]string) io.Writer {
	var oldnew []string
	for _, secret := range secretData {
		if secret == "" {
			continue
		}
		for _, part := range strings.Split(secret, "\n") {
			part = strings.TrimSpace(part)
			// avoid masking empty or single character strings.
			if len(part) < 2 {
				continue
			}
			oldnew = append(oldnew, part, maskedStr)
		}
	}
	if len(oldnew) == 0 {
		return w
	}
	return &masker{
		w: w,
		r: strings.NewReplacer(oldnew...),
	}
}

// Write writes p to the base writer. The method scans for any
// sensitive data in p and masks before writing.
func (m *masker) Write(p []byte) (n int, err error) {
	_, err = m.w.Write([]byte(m.r.Replace(string(p))))
	return len(p), err
}
