package command

import (
	"bytes"
	"fmt"
	"strings"
)

// CreateScript converts a slice of individual shell commands to
// a shell script.
func (m *manager) createScript(commands []string, secretData map[string]string) (string, error) {
	buf := new(bytes.Buffer)
	fmt.Fprintln(buf)
	fmt.Fprint(buf, optionScript)
	fmt.Fprintln(buf)
	var err error
	for _, command := range commands {
		escaped := fmt.Sprintf("%q", command)
		escaped = strings.Replace(escaped, "$", `\$`, -1)
		if len(secretData) > 0 {
			command, err = m.secretParser.SubstituteSecret(command, secretData)
			if err != nil {
				return "", err
			}
		}
		buf.WriteString(fmt.Sprintf(
			traceScript,
			escaped,
			command,
		))
	}
	return buf.String(), nil
}

// optionScript is a helper script this is added to the build
// to set shell options, in this case, to exit on error.
const optionScript = `
set -e
`

// traceScript is a helper script that is added to
// the build script to trace a command.
const traceScript = `
echo + %s
%s
`
