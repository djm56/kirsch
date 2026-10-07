// Package prompt assembles the Kirsch system prompt from an embedded template
// and caller-supplied environment and project context.
package prompt

import (
	_ "embed"
	"strings"
	"text/template"
)

//go:embed system.md
var systemTemplate string

// Env carries environment data supplied by the caller. The agent tree does not
// gather this data itself; the app adapter wires it later.
type Env struct {
	// WorkspaceRoot is the absolute workspace root.
	WorkspaceRoot string
	// ProjectType is the detected project type, e.g. "go" or "unknown".
	ProjectType string
	// Branch is the current git branch, or "unknown" when not available.
	Branch string
	// Dirty is true when the working tree has uncommitted changes.
	Dirty bool
	// OS is the operating system name, e.g. "darwin", "linux", "windows".
	OS string
	// HasRG is true when ripgrep (rg) is available in PATH.
	HasRG bool
}

// assembleData is the template input.
type assembleData struct {
	Env
	HasProjectContext bool
	ProjectContext    string
}

// Assemble builds the complete system prompt for the given environment and
// optional project context. Passing an empty projectContext omits section 5.
// The same inputs always produce byte-identical output.
func Assemble(env Env, projectContext string) string {
	data := assembleData{
		Env:               env,
		HasProjectContext: projectContext != "",
		ProjectContext:    projectContext,
	}

	// system.md is embedded and static; parse errors are build-time bugs.
	tmpl := template.Must(template.New("system").Parse(systemTemplate))

	var b strings.Builder
	if err := tmpl.Execute(&b, data); err != nil {
		// Execution only fails for missing template values or I/O errors on the
		// builder; both are impossible here.
		panic("prompt: failed to execute embedded system.md: " + err.Error())
	}
	return b.String()
}
