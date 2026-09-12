package tui

import (
	"os"
	"strings"
	"testing"
)

// Writes the current render of every scenario to a file, so stale reference
// grids can be reconciled from a renderer that is already byte-exact on the
// states it does match.
func TestGenerateGrids(t *testing.T) {
	if os.Getenv("KIRSCH_REGEN") == "" {
		t.Skip("set KIRSCH_REGEN=1 to regenerate")
	}
	grids := parseGrids(t)
	var b strings.Builder
	for _, sc := range scenarios() {
		g := grids[sc.Screen]
		rows := strings.Split(buildScenario(t, sc, g), "\n")
		for i := range rows {
			rows[i] = strings.TrimRight(rows[i], " ")
		}
		b.WriteString("@@SCREEN " + sc.Screen + "\n")
		b.WriteString(strings.Join(rows, "\n") + "\n")
	}
	if err := os.WriteFile("/tmp/kirsch-grids.txt", []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %d scenarios", len(scenarios()))
}
