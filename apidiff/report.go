package apidiff

import (
	"bytes"
	"fmt"
	"io"
)

// Report describes the changes detected by Changes.
type Report struct {
	Incompatible, Compatible []string
}

func (r Report) String() string {
	var buf bytes.Buffer
	if err := r.Text(&buf); err != nil {
		return fmt.Sprintf("!!%v", err)
	}
	return buf.String()
}

func (r Report) Text(w io.Writer) error {
	var err error

	write := func(s string) {
		if err == nil {
			_, err = io.WriteString(w, s)
		}
	}

	writeslice := func(ss []string) {
		for _, s := range ss {
			write("- ")
			write(s)
			write("\n")
		}
	}

	if len(r.Incompatible) > 0 {
		write("Incompatible changes:\n")
		writeslice(r.Incompatible)
	}
	if len(r.Compatible) > 0 {
		if len(r.Incompatible) > 0 {
			write("\n")
		}
		write("Compatible changes:\n")
		writeslice(r.Compatible)
	}
	return err
}
