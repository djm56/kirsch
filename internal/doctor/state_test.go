//go:build unix

package doctor

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/session"
)

// stateRun runs doctor end to end (doctor.Run) with a global config file whose
// [session] storage_dir is storageDir, a --workspace folder outside any
// repository, and home as Options.HomeDir (a fresh temp dir when home is
// empty). It returns the outcome and the workspace's canonical root, which is
// what the session hash is taken from.
func stateRun(t *testing.T, storageDir, home string) (Outcome, string) {
	t.Helper()
	if home == "" {
		home = t.TempDir() // stateRootFor needs a home for every storage_dir
	}
	return stateRunRaw(t, storageDir, home)
}

// stateRunRaw is stateRun without the fixture home: home is passed through as
// given, empty included, for the tests about an unknown home directory.
func stateRunRaw(t *testing.T, storageDir, home string) (Outcome, string) {
	t.Helper()
	ws := workspaceDir(t, "")
	cfg := filepath.Join(t.TempDir(), "config.toml")
	body := ""
	if storageDir != "" {
		body = "[session]\nstorage_dir = \"" + storageDir + "\"\n"
	}
	writeFile(t, cfg, body)
	chdirOutsideRepo(t, ws)
	out := Run(Options{LookPath: lookWithout(), WorkspaceFlag: ws, GlobalConfigPath: cfg, HomeDir: home})
	if !out.ConfigLoaded || out.Workspace == nil {
		t.Fatalf("control: config loaded=%v workspace=%v", out.ConfigLoaded, out.Workspace)
	}
	return out, out.Workspace.CanonicalRoot
}

// TestStateAndLockLinesFollowContext: the state and lock lines come straight
// after the context line, in that order, and are the last two lines.
func TestStateAndLockLinesFollowContext(t *testing.T) {
	out, _ := stateRun(t, filepath.Join(t.TempDir(), "kr"), "")
	n := len(out.Results)
	if n < 3 || out.Results[n-3].Name != "context" || out.Results[n-2].Name != "state" || out.Results[n-1].Name != "lock" {
		t.Fatalf("last three lines = %+v, want context, state, lock", out.Results[max(0, n-3):])
	}
}

// TestStatePathPrinted drives doctor.Run and checks the path printed in the
// state line is exactly <root>/sessions/<hash>: a plain root, a storage_dir
// ending in "sessions" (stripped, then re-added by SessionDir), and an unclean
// path. The "~" cases are expanded by config.Load before stateRootFor sees them
// (against the fixture HomeDir); stateRootFor's own "~" branch is covered by
// TestStateRootFor.
//
// The unclean input is built by string concatenation, because filepath.Join
// cleans its result. config.Load does not clean it: it only expands a leading
// "~" (ExpandHome, config.go) and passes any other value through unchanged, so
// the ".." reaches stateRootFor, where filepath.Abs removes it (Abs cleans an
// absolute path; the explicit Clean after it changes nothing). The case asserts
// that its input really holds "..". It does not pin the cleaning itself: the
// printed path is built by session.SessionDir, whose Join cleans it anyway, so
// the "unclean" case here would pass without stateRootFor cleaning. That is
// pinned by TestStateRootFor's "unclean path without sessions cleaned" case.
func TestStatePathPrinted(t *testing.T) {
	home := t.TempDir()
	base := t.TempDir()
	sep := string(os.PathSeparator)
	unclean := base + sep + "a" + sep + ".." + sep + "b"
	if !strings.Contains(unclean, "..") {
		t.Fatalf("control: the unclean input %q holds no \"..\"", unclean)
	}
	for _, tc := range []struct {
		name, storage, root string
	}{
		{"plain root", filepath.Join(base, "plain"), filepath.Join(base, "plain")},
		{"trailing sessions", filepath.Join(base, "x", "sessions"), filepath.Join(base, "x")},
		{"tilde", "~/kr/sessions", filepath.Join(home, "kr")},
		{"tilde alone", "~", home},
		{"unclean", unclean, filepath.Join(base, "b")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, ws := stateRun(t, tc.storage, home)
			want := filepath.Join(tc.root, "sessions", session.WorkspaceHash(ws))
			got := one(t, out.Results, "state")
			if !strings.HasPrefix(got.Detail, want+" ") {
				t.Errorf("state detail = %q, want it to start with %q", got.Detail, want+" ")
			}
		})
	}
}

// TestStateRootFor covers stateRootFor directly, including the cases Run cannot
// reach through a loaded config (empty storage_dir, "~" paths, unknown home).
func TestStateRootFor(t *testing.T) {
	for _, tc := range []struct {
		name, storage, home, want string
		wantErr                   bool
	}{
		{"empty uses default", "", "/h", "/h/.local/share/kirsch", false},
		{"empty without home", "", "", "", true},
		{"tilde without home", "~/x", "", "", true},
		{"tilde alone", "~", "/h", "/h", false},
		{"tilde sessions", "~/.local/share/kirsch/sessions", "/h", "/h/.local/share/kirsch", false},
		{"sessions stripped", "/data/sessions", "/h", "/data", false},
		{"sessions not last is kept", "/data/sessions/more", "/h", "/data/sessions/more", false},
		{"sessions lookalike kept", "/data/mysessions", "/h", "/data/mysessions", false},
		{"unclean path cleaned", "/data/./k//sessions/", "/h", "/data/k", false},
		{"unclean path without sessions cleaned", "/data/./k//x/../y", "/h", "/data/k/y", false},
		{"absolute with home", "/data/k", "/h", "/data/k", false},
		{"absolute without home is an error, as App.stateRoot", "/data/k", "", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := stateRootFor(tc.storage, tc.home)
			if (err != nil) != tc.wantErr || got != tc.want {
				t.Errorf("stateRootFor(%q, %q) = %q, %v; want %q, err=%v", tc.storage, tc.home, got, err, tc.want, tc.wantErr)
			}
		})
	}
}

// TestStateLineStatuses covers the session-directory branches of the state
// line through doctor.Run: existing and writable (ok), missing with a writable
// ancestor (ok), and three that stop kirsch starting (fail): a non-writable
// existing directory (mode 0o500, restored in Cleanup), missing with a
// non-writable ancestor, and a file where the directory should be. The
// non-writable cases skip as root, for whom access(2) says yes.
func TestStateLineStatuses(t *testing.T) {
	restore := func(t *testing.T, path string) {
		t.Cleanup(func() { _ = os.Chmod(path, 0o700) })
	}
	mkSessionDir := func(t *testing.T, root string, ws string) string {
		dir := session.SessionDir(root, ws)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		return dir
	}

	t.Run("existing writable is ok", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "kr")
		_, ws := stateRun(t, root, "")
		dir := mkSessionDir(t, root, ws)
		out, _ := stateRunSame(t, root, ws)
		got := one(t, out.Results, "state")
		if got.Status != StatusOK || got.Detail != dir+" exists and is writable" {
			t.Errorf("state = %+v", got)
		}
	})

	t.Run("missing with writable ancestor is ok", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "kr")
		out, ws := stateRun(t, root, "")
		dir := session.SessionDir(root, ws)
		anc := filepath.Dir(root) // root and sessions do not exist yet
		want := dir + " does not exist yet; kirsch will create it (" + anc + " is writable)"
		if got := one(t, out.Results, "state"); got.Status != StatusOK || got.Detail != want {
			t.Errorf("state = %+v, want ok %q", got, want)
		}
	})

	t.Run("non-writable directory is fail", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("running as root: access(2) grants write, cannot test a non-writable directory")
		}
		root := filepath.Join(t.TempDir(), "kr")
		_, ws := stateRun(t, root, "")
		dir := mkSessionDir(t, root, ws)
		restore(t, dir)
		if err := os.Chmod(dir, 0o500); err != nil {
			t.Fatal(err)
		}
		out, _ := stateRunSame(t, root, ws)
		got := one(t, out.Results, "state")
		want := dir + " is not writable; " + cannotStartNew
		if got.Status != StatusFail || got.Detail != want {
			t.Errorf("state = %+v, want fail %q", got, want)
		}
		// Control: the same directory with its mode restored is ok.
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		out, _ = stateRunSame(t, root, ws)
		if got := one(t, out.Results, "state"); got.Status != StatusOK {
			t.Errorf("control after chmod 0o700: state = %+v, want ok", got)
		}
	})

	t.Run("missing with non-writable ancestor is fail", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("running as root")
		}
		parent := t.TempDir()
		restore(t, parent)
		root := filepath.Join(parent, "kr")
		if err := os.Chmod(parent, 0o500); err != nil {
			t.Fatal(err)
		}
		out, _ := stateRunSame(t, root, "")
		got := one(t, out.Results, "state")
		want := " does not exist and " + parent + " is not writable; " + cannotStart
		if got.Status != StatusFail || !strings.HasSuffix(got.Detail, want) {
			t.Errorf("state = %+v, want fail ending %q", got, want)
		}
	})

	t.Run("file where the directory should be is fail", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "kr")
		_, ws := stateRun(t, root, "")
		dir := session.SessionDir(root, ws)
		writeFile(t, dir, "x")
		out, _ := stateRunSame(t, root, ws)
		got := one(t, out.Results, "state")
		want := dir + " is not a folder; " + cannotStart
		if got.Status != StatusFail || got.Detail != want {
			t.Errorf("state = %+v, want fail %q", got, want)
		}
	})
}

// stateRunSame runs doctor with the same storage root and, when ws is non-empty,
// against that same workspace folder, so the session hash matches a directory
// the test prepared. stateRun picks a fresh workspace folder each call, so this
// helper takes the prepared one from the first run's canonical root.
func stateRunSame(t *testing.T, storageDir, ws string) (Outcome, string) {
	t.Helper()
	if ws == "" {
		return stateRun(t, storageDir, "")
	}
	cfg := filepath.Join(t.TempDir(), "config.toml")
	writeFile(t, cfg, "[session]\nstorage_dir = \""+storageDir+"\"\n")
	chdirOutsideRepo(t, ws)
	out := Run(Options{LookPath: lookWithout(), WorkspaceFlag: ws, GlobalConfigPath: cfg, HomeDir: t.TempDir()})
	if !out.ConfigLoaded || out.Workspace == nil || out.Workspace.CanonicalRoot != ws {
		t.Fatalf("control: loaded=%v workspace=%v want root %q", out.ConfigLoaded, out.Workspace, ws)
	}
	return out, ws
}

// TestLockLineHeldThenFree: while the test holds the lock with session.Acquire
// doctor reports held (warn); the lock is still held after the run (a second
// Acquire gets ErrLockHeld); after Release doctor reports free (ok). One process.
func TestLockLineHeldThenFree(t *testing.T) {
	root := filepath.Join(t.TempDir(), "kr")
	_, ws := stateRun(t, root, "")

	l, err := session.Acquire(root, ws)
	if err != nil {
		t.Fatal(err)
	}
	released := false
	defer func() {
		if !released {
			_ = l.Release()
		}
	}()

	out, _ := stateRunSame(t, root, ws)
	got := one(t, out.Results, "lock")
	wantHeld := "held by another kirsch in this workspace; a new start would skip resume and session grants"
	if got.Status != StatusWarn || got.Detail != wantHeld {
		t.Errorf("held: lock = %+v, want warn %q", got, wantHeld)
	}
	if _, err := session.Acquire(root, ws); !errors.Is(err, session.ErrLockHeld) {
		t.Errorf("Acquire after doctor = %v, want ErrLockHeld (doctor must not release the holder)", err)
	}

	if err := l.Release(); err != nil {
		t.Fatal(err)
	}
	released = true
	out, _ = stateRunSame(t, root, ws)
	if got := one(t, out.Results, "lock"); got.Status != StatusOK || got.Detail != "free" {
		t.Errorf("free: lock = %+v, want ok \"free\"", got)
	}
}

// TestDoctorCreatesNothing: with a root that does not exist, it still does not
// exist after the run; with an existing session directory and no .lock, there is
// still no .lock afterwards. The lock line says "no lock file yet" in the second
// case, and the first case's lock line is ok too.
func TestDoctorCreatesNothing(t *testing.T) {
	t.Run("missing root stays missing", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "kr")
		out, _ := stateRun(t, root, "")
		if _, err := os.Stat(root); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("root after doctor: %v, want not exist", err)
		}
		if got := one(t, out.Results, "lock"); got.Status != StatusOK || got.Detail != "free (no lock file yet)" {
			t.Errorf("lock = %+v", got)
		}
	})

	t.Run("existing session dir gets no lock file", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "kr")
		_, ws := stateRun(t, root, "")
		dir := session.SessionDir(root, ws)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		out, _ := stateRunSame(t, root, ws)
		if _, err := os.Stat(filepath.Join(dir, ".lock")); !errors.Is(err, os.ErrNotExist) {
			t.Errorf(".lock after doctor: %v, want not exist", err)
		}
		entries, rerr := os.ReadDir(dir)
		if rerr != nil {
			t.Fatal(rerr)
		}
		if len(entries) != 0 {
			t.Errorf("session dir holds %d entries after doctor, want 0", len(entries))
		}
		if got := one(t, out.Results, "lock"); got.Detail != "free (no lock file yet)" {
			t.Errorf("lock = %+v", got)
		}
	})

	t.Run("control: Acquire does create the lock file", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "kr")
		_, ws := stateRun(t, root, "")
		l, err := session.Acquire(root, ws)
		if err != nil {
			t.Fatal(err)
		}
		defer l.Release()
		if _, err := os.Stat(filepath.Join(session.SessionDir(root, ws), ".lock")); err != nil {
			t.Errorf("control: .lock after Acquire: %v", err)
		}
	})
}

// TestStateAndLockSkipped: with the config not loaded both lines read
// "skipped: config did not load"; with the config loaded but no workspace both
// read "skipped: no workspace". Both are warn.
func TestStateAndLockSkipped(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "config.toml")
	writeFile(t, bad, "this is = not [valid toml\n")
	chdirOutsideRepo(t)
	out := Run(Options{LookPath: lookWithout(), WorkspaceFlag: t.TempDir(), GlobalConfigPath: bad})
	if out.ConfigLoaded {
		t.Fatal("control: the broken config loaded")
	}
	for _, name := range []string{"state", "lock"} {
		if got := one(t, out.Results, name); got.Status != StatusWarn || got.Detail != "skipped: config did not load" {
			t.Errorf("%s = %+v, want warn \"skipped: config did not load\"", name, got)
		}
	}

	out = Run(Options{LookPath: lookWithout(), WorkspaceFlag: filepath.Join(t.TempDir(), "missing")})
	if !out.ConfigLoaded || out.Workspace != nil {
		t.Fatalf("control: loaded=%v workspace=%v", out.ConfigLoaded, out.Workspace)
	}
	for _, name := range []string{"state", "lock"} {
		if got := one(t, out.Results, name); got.Status != StatusWarn || got.Detail != "skipped: no workspace" {
			t.Errorf("%s = %+v, want warn \"skipped: no workspace\"", name, got)
		}
	}
}

// TestStateNeedsHome: with an empty HomeDir both lines name the cause, for a
// storage_dir that uses the home directory and for an absolute one that does
// not (App.stateRoot asks for the home directory in both cases). The state line
// fails, because createSessionStore and ResumeLastSession both need the state
// root, so no start succeeds; the lock line warns, because App.AcquireLock
// carries on without one.
func TestStateNeedsHome(t *testing.T) {
	for _, storage := range []string{"~/kr", filepath.Join(t.TempDir(), "kr")} {
		out, _ := stateRunRaw(t, storage, "")
		for _, tc := range []struct {
			name string
			want Status
		}{{"state", StatusFail}, {"lock", StatusWarn}} {
			got := one(t, out.Results, tc.name)
			if got.Status != tc.want || !strings.Contains(got.Detail, "home directory unknown") {
				t.Errorf("%s for %q = %+v, want %v naming the unknown home directory", tc.name, storage, got, tc.want)
			}
		}
		if got := one(t, out.Results, "state"); !strings.HasSuffix(got.Detail, "; "+cannotStart) {
			t.Errorf("state for %q = %+v, want it to end %q", storage, got, "; "+cannotStart)
		}
	}
}

// TestLockResultProbeError: lockResult turns a probe error into a warn starting
// "could not check the lock: ". The error comes from a real condition:
// root/sessions is a file. That the detail is one line is TestLockLineOneLine's
// job, which has the positive control (an error that does contain a newline);
// this error has none, so a one-line check here could not fail.
func TestLockResultProbeError(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "sessions"), "x")
	got := lockResult(root, t.TempDir(), session.SessionDir(root, "w"))
	if got.Status != StatusWarn || !strings.HasPrefix(got.Detail, "could not check the lock: ") {
		t.Errorf("lockResult = %+v, want a warn starting \"could not check the lock: \"", got)
	}
}

// statusCase is one fixture for TestStateAndLockStatusPerCase. prep builds the
// state folder and returns the storage_dir and the workspace folder to run
// against (empty ws means a fresh one), and noHome runs with an empty HomeDir.
// asRoot marks a fixture that relies on access(2) refusing the user, which it
// does not do for root.
type statusCase struct {
	name        string
	asRoot      bool
	noHome      bool
	prep        func(t *testing.T) (storage, ws string)
	state, lock Status
}

// chmodRestore sets path to mode and restores 0o700 in Cleanup, so t.TempDir
// can remove the tree.
func chmodRestore(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	t.Cleanup(func() { _ = os.Chmod(path, 0o700) })
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

// preparedRoot returns a storage root under a temp dir and a workspace whose
// session directory exists in it.
func preparedRoot(t *testing.T) (root, ws, dir string) {
	t.Helper()
	root = filepath.Join(t.TempDir(), "kr")
	_, ws = stateRun(t, root, "")
	dir = session.SessionDir(root, ws)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	return root, ws, dir
}

// TestStateAndLockStatusPerCase drives doctor.Run once per fixture and pins
// the status of both lines. The lock line is never a fail in any fixture,
// because App.AcquireLock returns nil on every path. The state line is a fail
// in every fixture that stops a start (see TestStateFailMatchesSessionStart for
// the start-up side) and ok in the writable ones, which are the positive
// controls for the fails.
func TestStateAndLockStatusPerCase(t *testing.T) {
	afile := func(t *testing.T) string {
		f := filepath.Join(t.TempDir(), "afile")
		writeFile(t, f, "x")
		return f
	}
	cases := []statusCase{
		{name: "tilde with home", prep: func(*testing.T) (string, string) { return "~/kr", "" }, state: StatusOK, lock: StatusOK},
		{name: "absolute, nothing yet", prep: func(t *testing.T) (string, string) { return filepath.Join(t.TempDir(), "kr"), "" }, state: StatusOK, lock: StatusOK},
		{name: "existing writable session dir", prep: func(t *testing.T) (string, string) {
			root, ws, _ := preparedRoot(t)
			return root, ws
		}, state: StatusOK, lock: StatusOK},
		{name: "readable index.json", prep: func(t *testing.T) (string, string) {
			root, ws, _ := preparedRoot(t)
			writeFile(t, session.IndexPath(root), "{}")
			return root, ws
		}, state: StatusOK, lock: StatusOK},
		{name: "lock held", prep: func(t *testing.T) (string, string) {
			root, ws, _ := preparedRoot(t)
			l, err := session.Acquire(root, ws)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = l.Release() })
			return root, ws
		}, state: StatusOK, lock: StatusWarn},
		{name: "lock file not writable", asRoot: true, prep: func(t *testing.T) (string, string) {
			root, ws, lockPath := lockedThenFree(t)
			chmodRestore(t, lockPath, 0o400)
			return root, ws
		}, state: StatusOK, lock: StatusWarn},
		{name: "session dir not writable", asRoot: true, prep: func(t *testing.T) (string, string) {
			root, ws, dir := preparedRoot(t)
			chmodRestore(t, dir, 0o500)
			return root, ws
		}, state: StatusFail, lock: StatusOK},
		{name: "lock file and session dir not writable", asRoot: true, prep: func(t *testing.T) (string, string) {
			root, ws, lockPath := lockedThenFree(t)
			chmodRestore(t, lockPath, 0o400)
			chmodRestore(t, filepath.Dir(lockPath), 0o500)
			return root, ws
		}, state: StatusFail, lock: StatusWarn},
		{name: "missing, ancestor not writable", asRoot: true, prep: func(t *testing.T) (string, string) {
			parent := t.TempDir()
			chmodRestore(t, parent, 0o500)
			return filepath.Join(parent, "kr"), ""
		}, state: StatusFail, lock: StatusOK},
		{name: "root not writable", asRoot: true, prep: func(t *testing.T) (string, string) {
			root, ws, _ := preparedRoot(t)
			chmodRestore(t, root, 0o500)
			return root, ws
		}, state: StatusFail, lock: StatusOK},
		{name: "index.json not readable", asRoot: true, prep: func(t *testing.T) (string, string) {
			root, ws, _ := preparedRoot(t)
			writeFile(t, session.IndexPath(root), "{}")
			chmodRestore(t, session.IndexPath(root), 0o200)
			return root, ws
		}, state: StatusFail, lock: StatusOK},
		{name: "file where the session dir should be", prep: func(t *testing.T) (string, string) {
			root := filepath.Join(t.TempDir(), "kr")
			_, ws := stateRun(t, root, "")
			writeFile(t, session.SessionDir(root, ws), "x")
			return root, ws
		}, state: StatusFail, lock: StatusWarn},
		{name: "root under a file", prep: func(t *testing.T) (string, string) { return filepath.Join(afile(t), "kr"), "" }, state: StatusFail, lock: StatusWarn},
		{name: "tilde without home", noHome: true, prep: func(*testing.T) (string, string) { return "~/kr", "" }, state: StatusFail, lock: StatusWarn},
		{name: "absolute without home", noHome: true, prep: func(t *testing.T) (string, string) { return filepath.Join(t.TempDir(), "kr"), "" }, state: StatusFail, lock: StatusWarn},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.asRoot && os.Geteuid() == 0 {
				t.Skip("running as root: access(2) grants every permission")
			}
			storage, ws := tc.prep(t)
			var out Outcome
			switch {
			case tc.noHome:
				out, _ = stateRunRaw(t, storage, "")
			default:
				out, _ = stateRunSame(t, storage, ws)
			}
			if got := one(t, out.Results, "state"); got.Status != tc.state {
				t.Errorf("state = %+v, want status %v", got, tc.state)
			}
			got := one(t, out.Results, "lock")
			if got.Status != tc.lock {
				t.Errorf("lock = %+v, want status %v", got, tc.lock)
			}
			if got.Status == StatusFail {
				t.Errorf("lock = %+v: the lock line must never fail", got)
			}
		})
	}
}

// TestStateRootNotWritable: the session directory exists and is writable, but
// the state root that holds index.json is not (mode 0o500), so WriteIndex
// cannot create its temporary file there and every new session fails. The
// state line is a fail naming the root. The control, the same root made
// writable again, is ok with the session directory's own detail. Skipped as
// root.
func TestStateRootNotWritable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: access(2) grants write")
	}
	root, ws, dir := preparedRoot(t)
	chmodRestore(t, root, 0o500)
	out, _ := stateRunSame(t, root, ws)
	want := root + " is not writable, so index.json cannot be updated; " + cannotStartNew
	if got := one(t, out.Results, "state"); got.Status != StatusFail || got.Detail != want {
		t.Errorf("state = %+v, want fail %q", got, want)
	}

	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	out, _ = stateRunSame(t, root, ws)
	if got := one(t, out.Results, "state"); got.Status != StatusOK || got.Detail != dir+" exists and is writable" {
		t.Errorf("control, writable root: state = %+v, want ok", got)
	}
}

// TestStateIndexNotReadable: index.json exists in a writable root but cannot
// be read (mode 0o200), or a folder sits where it should be. ReadIndex fails
// on both, so the state line is a fail naming the file. The control is the
// same file made readable again: ok. Doctor never parses the file, so its
// content ("{}") does not matter here. The mode case skips as root.
func TestStateIndexNotReadable(t *testing.T) {
	t.Run("not readable", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("running as root: access(2) grants read")
		}
		root, ws, dir := preparedRoot(t)
		idx := session.IndexPath(root)
		writeFile(t, idx, "{}")
		chmodRestore(t, idx, 0o200)
		out, _ := stateRunSame(t, root, ws)
		want := idx + " is not readable; " + cannotStart
		if got := one(t, out.Results, "state"); got.Status != StatusFail || got.Detail != want {
			t.Errorf("state = %+v, want fail %q", got, want)
		}

		if err := os.Chmod(idx, 0o600); err != nil {
			t.Fatal(err)
		}
		out, _ = stateRunSame(t, root, ws)
		if got := one(t, out.Results, "state"); got.Status != StatusOK || got.Detail != dir+" exists and is writable" {
			t.Errorf("control, readable index: state = %+v, want ok", got)
		}
	})

	t.Run("a folder", func(t *testing.T) {
		root, ws, _ := preparedRoot(t)
		idx := session.IndexPath(root)
		if err := os.Mkdir(idx, 0o700); err != nil {
			t.Fatal(err)
		}
		out, _ := stateRunSame(t, root, ws)
		want := idx + " is a folder, not a file; " + cannotStart
		if got := one(t, out.Results, "state"); got.Status != StatusFail || got.Detail != want {
			t.Errorf("state = %+v, want fail %q", got, want)
		}
	})
}

// TestStateRootDefensiveBranches reaches the root and index.json stat branches
// that a real filesystem shows earlier (os.Stat of the session directory would
// fail first), through the statFn seam. The session directory is real and
// writable, so sessionDirResult is ok and its own access(2) call runs on a real
// path; only the root's or index.json's stat answer is faked. The control, with
// no fake for the root, is ok.
func TestStateRootDefensiveBranches(t *testing.T) {
	root, _, dir := preparedRoot(t)
	f := filepath.Join(t.TempDir(), "afile")
	writeFile(t, f, "x")
	infos := map[string]os.FileInfo{}
	for _, p := range []string{dir, root, f} {
		fi, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		infos[p] = fi
	}
	dirInfo, rootInfo, fileInfo := infos[dir], infos[root], infos[f]
	idx := session.IndexPath(root)
	denied := errors.New("permission denied")
	for _, tc := range []struct {
		name  string
		errs  map[string]error
		infos map[string]os.FileInfo
		want  Result
	}{
		{
			"control", nil,
			map[string]os.FileInfo{dir: dirInfo, root: rootInfo},
			Result{StatusOK, checkState, dir + " exists and is writable"},
		},
		{
			"root cannot be read",
			map[string]error{root: &fs.PathError{Op: "stat", Path: root, Err: denied}},
			map[string]os.FileInfo{dir: dirInfo},
			Result{StatusFail, checkState, "cannot read " + root + ": permission denied; " + cannotStart},
		},
		{
			"root is not a folder", nil,
			map[string]os.FileInfo{dir: dirInfo, root: fileInfo},
			Result{StatusFail, checkState, root + " is not a folder; " + cannotStart},
		},
		{
			"index cannot be read",
			map[string]error{idx: &fs.PathError{Op: "stat", Path: idx, Err: denied}},
			map[string]os.FileInfo{dir: dirInfo, root: rootInfo},
			Result{StatusFail, checkState, "cannot read " + idx + ": permission denied; " + cannotStart},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fakeStat(t, tc.errs, tc.infos)
			if got := stateResult(root, dir); got != tc.want {
				t.Errorf("stateResult = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// startNewSession repeats, with the exported session API only, the three
// calls App.createSessionStore makes to start a new session:
// session.NewLayout, Layout.CreateSession and session.New. It is not
// App.StartNewSession itself (that needs an App, and stateRoot is unexported);
// TestStateFailMatchesSessionStart uses it as the start-up side of the
// comparison. The store is closed before returning.
func startNewSession(root, ws string) error {
	layout, err := session.NewLayout(root, ws)
	if err != nil {
		return err
	}
	path, _, err := layout.CreateSession(time.Now().UTC())
	if err != nil {
		return err
	}
	st, err := session.New(path)
	if err != nil {
		return err
	}
	return st.Close()
}

// TestStateFailMatchesSessionStart ties the state line to what start-up does:
// for each fixture, doctor runs first (it changes nothing), then
// startNewSession runs the new-session calls against the same folder. A fail
// must come with a start-up error, and the writable control's ok with a
// success. For the session-directory fixture the error comes from session.New,
// not CreateSession: CreateSession only creates folders and index.json, both
// of which still work there. Skipped as root.
func TestStateFailMatchesSessionStart(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: access(2) grants every permission")
	}
	for _, tc := range []struct {
		name string
		prep func(t *testing.T) (root, ws string)
		want Status
	}{
		{"writable control", func(t *testing.T) (string, string) {
			root, ws, _ := preparedRoot(t)
			return root, ws
		}, StatusOK},
		{"root not writable", func(t *testing.T) (string, string) {
			root, ws, _ := preparedRoot(t)
			chmodRestore(t, root, 0o500)
			return root, ws
		}, StatusFail},
		{"session dir not writable", func(t *testing.T) (string, string) {
			root, ws, dir := preparedRoot(t)
			chmodRestore(t, dir, 0o500)
			return root, ws
		}, StatusFail},
		{"index.json not readable", func(t *testing.T) (string, string) {
			root, ws, _ := preparedRoot(t)
			writeFile(t, session.IndexPath(root), "{}")
			chmodRestore(t, session.IndexPath(root), 0o200)
			return root, ws
		}, StatusFail},
		{"missing, ancestor not writable", func(t *testing.T) (string, string) {
			parent := t.TempDir()
			root := filepath.Join(parent, "kr")
			_, ws := stateRun(t, root, "")
			chmodRestore(t, parent, 0o500)
			return root, ws
		}, StatusFail},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, ws := tc.prep(t)
			out, _ := stateRunSame(t, root, ws)
			got := one(t, out.Results, "state")
			if got.Status != tc.want {
				t.Fatalf("state = %+v, want status %v", got, tc.want)
			}
			err := startNewSession(root, ws)
			switch tc.want {
			case StatusOK:
				if err != nil {
					t.Errorf("doctor said ok but the new-session calls failed: %v", err)
				}
			default:
				if err == nil {
					t.Errorf("doctor said fail (%q) but the new-session calls succeeded", got.Detail)
				}
			}
		})
	}
}

// TestRunFailsOnStateFolder: doctor.Run with a global config whose
// [session] storage_dir sits in a folder that is not writable (mode 0o500)
// gives a fail state line, and Failed reports the run as failed. The control,
// the same folder made writable again, gives an ok state line and a run that
// does not fail (the other lines of this fixture only warn). Skipped as root.
func TestRunFailsOnStateFolder(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: access(2) grants write")
	}
	parent := t.TempDir()
	root := filepath.Join(parent, "kr")
	chmodRestore(t, parent, 0o500)
	out, _ := stateRun(t, root, "")
	if got := one(t, out.Results, "state"); got.Status != StatusFail {
		t.Errorf("state = %+v, want fail", got)
	}
	if !Failed(out.Results) {
		t.Errorf("Failed = false with a fail state line: %+v", out.Results)
	}

	if err := os.Chmod(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	out, _ = stateRun(t, root, "")
	if got := one(t, out.Results, "state"); got.Status != StatusOK {
		t.Errorf("control: state = %+v, want ok", got)
	}
	if Failed(out.Results) {
		t.Errorf("control: Failed = true: %+v", out.Results)
	}
}

// TestStateResultCannotRead reaches the "cannot read" branch with a real
// condition: the storage root sits under a regular file, so os.Stat of the
// session directory fails with ENOTDIR, which is not "does not exist". The
// detail names the directory once: errText drops the path from the PathError.
func TestStateResultCannotRead(t *testing.T) {
	file := filepath.Join(t.TempDir(), "afile")
	writeFile(t, file, "x")
	root := filepath.Join(file, "kr")
	out, ws := stateRun(t, root, "")
	dir := session.SessionDir(root, ws)
	got := one(t, out.Results, "state")
	want := "cannot read " + dir + ": not a directory; " + cannotStart
	if got.Status != StatusFail || got.Detail != want {
		t.Errorf("state = %+v, want fail %q", got, want)
	}
	if n := strings.Count(got.Detail, dir); n != 1 {
		t.Errorf("detail names the directory %d times, want 1: %q", n, got.Detail)
	}
}

// fakeStat replaces statFn for the test (restored in Cleanup). A path in errs
// fails with that error; a path in infos succeeds with that FileInfo; any other
// path fails with a PathError wrapping ErrNotExist. statFn is a package
// variable, so a test that calls fakeStat must not call t.Parallel.
func fakeStat(t *testing.T, errs map[string]error, infos map[string]os.FileInfo) {
	t.Helper()
	old := statFn
	t.Cleanup(func() { statFn = old })
	statFn = func(name string) (os.FileInfo, error) {
		if err, ok := errs[name]; ok {
			return nil, err
		}
		if fi, ok := infos[name]; ok {
			return fi, nil
		}
		return nil, &fs.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
	}
}

// TestStateResultAncestorNotAFolder reaches the ancestor "is not a folder"
// branch through the statFn seam: the session directory does not exist and its
// nearest existing ancestor is a regular file. A real filesystem cannot produce
// this (os.Stat of the directory would say ENOTDIR), so the stat answers are
// faked; the file's FileInfo is a real one.
func TestStateResultAncestorNotAFolder(t *testing.T) {
	file := filepath.Join(t.TempDir(), "afile")
	writeFile(t, file, "x")
	fi, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(file, "sessions", "h")
	anc := filepath.Join(file, "sessions")
	fakeStat(t, nil, map[string]os.FileInfo{anc: fi})
	got := stateResult(file, dir)
	want := dir + " does not exist and " + anc + " is not a folder; " + cannotStart
	if got.Status != StatusFail || got.Detail != want {
		t.Errorf("state = %+v, want fail %q", got, want)
	}
}

// TestStateResultAncestorCannotBeRead reaches the ancestor "cannot be read"
// branch through the statFn seam: the session directory does not exist and
// stat of its parent fails with a permission error. As above, a real
// filesystem would fail the directory's own stat first. The detail names the
// ancestor once outside the directory path (dir starts with anc, so dir is
// removed before counting). The count fails on its own if the full error text
// is printed, which repeats anc as "stat /r/sessions: permission denied"; the
// exact comparison above catches that too.
func TestStateResultAncestorCannotBeRead(t *testing.T) {
	dir := filepath.Join(string(os.PathSeparator), "r", "sessions", "h")
	anc := filepath.Dir(dir)
	fakeStat(t, map[string]error{
		anc: &fs.PathError{Op: "stat", Path: anc, Err: errors.New("permission denied")},
	}, nil)
	got := stateResult(filepath.Dir(anc), dir)
	want := dir + " does not exist and " + anc + " cannot be read: permission denied; " + cannotStart
	if got.Status != StatusFail || got.Detail != want {
		t.Errorf("state = %+v, want fail %q", got, want)
	}
	if n := strings.Count(strings.ReplaceAll(got.Detail, dir, ""), anc); n != 1 {
		t.Errorf("detail names the ancestor %d times, want 1: %q", n, got.Detail)
	}
}

// TestLockLineNotCheckedHere: lockLine with LockUnknown and no error (the state
// the non-Unix ProbeLock returns) is a warn saying so, not "free".
func TestLockLineNotCheckedHere(t *testing.T) {
	got := lockLine(session.LockUnknown, nil)
	if got.Status != StatusWarn || got.Detail != "not checked on this platform" {
		t.Errorf("lockLine = %+v, want warn \"not checked on this platform\"", got)
	}
}

// TestLockLineOneLine: lockLine and errText put an error on one line. The
// positive control is the error with a newline in it: it must come out joined
// with "; ", so a missing oneLine shows as a newline in the detail.
func TestLockLineOneLine(t *testing.T) {
	got := lockLine(session.LockUnknown, errors.New("first\nsecond\r\nthird"))
	want := "could not check the lock: first; second; third"
	if got.Status != StatusWarn || got.Detail != want {
		t.Errorf("lockLine = %+v, want warn %q", got, want)
	}
	if strings.ContainsAny(got.Detail, "\r\n") {
		t.Errorf("lockLine detail still has a line break: %q", got.Detail)
	}
}

// TestErrText: a *fs.PathError for the printed path loses its path (it would be
// shown twice); a PathError for another path, and any other error, are kept
// whole; every result is one line.
func TestErrText(t *testing.T) {
	for _, tc := range []struct {
		name    string
		err     error
		printed string
		want    string
	}{
		{"path already printed", &fs.PathError{Op: "stat", Path: "/p", Err: errors.New("permission denied")}, "/p", "permission denied"},
		{"other path kept", &fs.PathError{Op: "stat", Path: "/q", Err: errors.New("permission denied")}, "/p", "stat /q: permission denied"},
		{"plain error kept", errors.New("boom"), "/p", "boom"},
		{"wrapped path error", fmt.Errorf("open lock file: %w", &fs.PathError{Op: "open", Path: "/p", Err: errors.New("denied")}), "/p", "denied"},
		{"newline in the inner error", &fs.PathError{Op: "stat", Path: "/p", Err: errors.New("a\nb")}, "/p", "a; b"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := errText(tc.err, tc.printed); got != tc.want {
				t.Errorf("errText = %q, want %q", got, tc.want)
			}
		})
	}
}

// lockedThenFree creates the session directory's .lock file by taking and
// releasing the lock, then returns the storage root, workspace root and the
// lock file path.
func lockedThenFree(t *testing.T) (root, ws, lockPath string) {
	t.Helper()
	root = filepath.Join(t.TempDir(), "kr")
	_, ws = stateRun(t, root, "")
	l, err := session.Acquire(root, ws)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Release(); err != nil {
		t.Fatal(err)
	}
	return root, ws, filepath.Join(session.SessionDir(root, ws), ".lock")
}

// TestLockLineFreeLeavesNoLock: the session directory and an unlocked .lock
// exist. doctor reports free, and afterwards session.Acquire succeeds at once,
// so the probe did not leave its shared lock behind.
func TestLockLineFreeLeavesNoLock(t *testing.T) {
	root, ws, _ := lockedThenFree(t)
	out, _ := stateRunSame(t, root, ws)
	if got := one(t, out.Results, "lock"); got.Status != StatusOK || got.Detail != "free" {
		t.Fatalf("lock = %+v, want ok \"free\"", got)
	}
	l, err := session.Acquire(root, ws)
	if err != nil {
		t.Fatalf("Acquire after doctor = %v, want success (doctor left its lock behind)", err)
	}
	_ = l.Release()
}

// TestLockLineNotWritable: a free lock whose .lock file the user cannot write
// is a warn, because session.Acquire opens it write-only and App.AcquireLock
// carries on with the single-instance guard off. The control, the same
// file made writable again, is ok "free". Skipped as root, for whom access(2)
// says yes.
func TestLockLineNotWritable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: access(2) grants write")
	}
	root, ws, lockPath := lockedThenFree(t)
	t.Cleanup(func() { _ = os.Chmod(lockPath, 0o600) })
	if err := os.Chmod(lockPath, 0o400); err != nil {
		t.Fatal(err)
	}
	out, _ := stateRunSame(t, root, ws)
	want := "free, but " + lockPath + " is not writable; the single-instance guard will be off"
	if got := one(t, out.Results, "lock"); got.Status != StatusWarn || got.Detail != want {
		t.Errorf("read-only: lock = %+v, want warn %q", got, want)
	}

	if err := os.Chmod(lockPath, 0o600); err != nil {
		t.Fatal(err)
	}
	out, _ = stateRunSame(t, root, ws)
	if got := one(t, out.Results, "lock"); got.Status != StatusOK || got.Detail != "free" {
		t.Errorf("control, writable: lock = %+v, want ok \"free\"", got)
	}
}
