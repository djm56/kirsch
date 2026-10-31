package doctor

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/djm56/kirsch/internal/session"
)

// Names of the two lines this file reports.
const (
	checkState = "state"
	checkLock  = "lock"
)

// The two endings of a state-line fail. Start-up runs startupPostAttach →
// initSession (chooseSessionInit, cmd/kirsch/main.go), which ends in
// App.StartNewSession → ensureStore → createSessionStore (internal/app/app.go)
// whenever it starts a new session: on --new, with auto_resume off, when there
// is nothing to resume, when resuming fails, and when another kirsch holds the
// lock. createSessionStore needs the state root (App.stateRoot), then
// Layout.CreateSession (MkdirAll of the session directory, ReadIndex, then
// WriteIndex, which writes a temporary file in the root and renames it onto
// index.json), then session.New, which creates the session file. Any error
// there is returned out of startupPostAttach before the TUI runs, and main
// prints it and exits 1.
//
// cannotStart ends a fail that also breaks the resume path, so every start
// fails. cannotStartNew ends a fail where a folder fails the write-and-search
// check: every start that needs a new session fails, which is all that
// cannotStartNew claims. Where the folder is still searchable,
// App.ResumeLastSession can reopen an existing session file in it and only logs
// a failed index update, so a start that resumes may succeed.
const (
	cannotStart    = "kirsch cannot start until this is fixed"
	cannotStartNew = "kirsch cannot create a new session, so any start that needs one fails until this is fixed"
)

// stateRootFor works out the session layout root from the configured
// storage_dir, with the behaviour of the unexported App.stateRoot in
// internal/app/app.go, which doctor cannot call. The two must agree:
//   - an empty storage_dir gives <home>/.local/share/kirsch;
//   - "~" or "~/..." is expanded with home;
//   - the result is made absolute and cleaned;
//   - a trailing "sessions" element is stripped, because the layout adds it.
//
// home comes from Options.HomeDir rather than os.UserHomeDir. Like
// App.stateRoot, which asks os.UserHomeDir for every non-empty storage_dir
// whether or not the path uses it, stateRootFor returns the "home directory
// unknown" error whenever home is empty, whatever the storage_dir is.
func stateRootFor(storageDir, home string) (string, error) {
	if home == "" {
		return "", errors.New("home directory unknown")
	}

	dir := storageDir
	if dir == "" {
		return filepath.Join(home, ".local", "share", "kirsch"), nil
	}

	if dir == "~" || strings.HasPrefix(dir, "~"+string(os.PathSeparator)) {
		dir = filepath.Join(home, strings.TrimPrefix(dir, "~"))
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve session storage dir: %w", err)
	}
	// Redundant (filepath.Abs already cleans), but kept so this stays a
	// line-for-line copy of App.stateRoot; do not drop it on one side only.
	abs = filepath.Clean(abs)

	if filepath.Base(abs) == "sessions" {
		return filepath.Dir(abs), nil
	}
	return abs, nil
}

// stateResults returns the state and lock lines, in that order.
//
// The state line fails when the state folder would stop kirsch starting (see
// cannotStart and cannotStartNew for the start-up path it is read from). The
// lock line is only ok or warn: App.AcquireLock, which startupPostAttach calls
// before initSession, logs and carries on for a state-root error, a lock error
// or a held lock, so no lock problem stops a start. With no loaded config, or
// no workspace, both lines say so, check nothing and warn: the config line has
// already failed the run in the first case, and the repository line decides
// the second. Nothing here creates a file or folder.
func stateResults(out Outcome, home string) []Result {
	switch {
	case !out.ConfigLoaded:
		return []Result{{StatusWarn, checkState, skippedNoConfig}, {StatusWarn, checkLock, skippedNoConfig}}
	case out.Workspace == nil:
		return []Result{{StatusWarn, checkState, skippedNoWorkspace}, {StatusWarn, checkLock, skippedNoWorkspace}}
	}

	root, err := stateRootFor(out.Config.Session.StorageDir, home)
	if err != nil {
		// App.stateRoot fails the same way, so both ResumeLastSession and
		// createSessionStore fail and no start succeeds.
		msg := "cannot work out the state folder: " + errText(err, "")
		return []Result{
			{StatusFail, checkState, msg + "; " + cannotStart},
			{StatusWarn, checkLock, msg},
		}
	}
	dir := session.SessionDir(root, out.Workspace.CanonicalRoot)
	return []Result{stateResult(root, dir), lockResult(root, out.Workspace.CanonicalRoot, dir)}
}

// statFn is the stat call the state line uses for the session directory, its
// ancestors, the state root and index.json. It is os.Stat; tests replace it to
// reach branches a real filesystem shows earlier: an ancestor that is not a
// folder or cannot be read, a root that is not a folder or cannot be read, and
// an index.json that cannot be stat'ed. On a real filesystem each of those
// makes os.Stat(dir) fail first (ENOTDIR or EACCES), which takes the "cannot
// read" branch, so those branches are defensive.
var statFn = os.Stat

// stateResult reports whether kirsch could create a new session under root:
// first the session directory dir (sessionDirResult), then the root and its
// index.json (rootProblem), in the order Layout.CreateSession touches them. The
// first problem found is the line; with none, the line is sessionDirResult's
// ok. A "not checked on this platform" warn from sessionDirResult is returned
// as it is, without the root checks, which could not be made either.
func stateResult(root, dir string) Result {
	res := sessionDirResult(dir)
	if res.Status != StatusOK {
		return res
	}
	if prob, found := rootProblem(root); found {
		return prob
	}
	return res
}

// sessionDirResult reports whether the session directory can take a new
// session file. An existing directory must pass access(2) for write and
// search, because session.New creates the file in it. A missing one is fine if
// its nearest existing ancestor is a writable directory, because
// Layout.CreateSession (and session.Acquire) create the missing folders with
// os.MkdirAll; otherwise MkdirAll fails and so does every start.
func sessionDirResult(dir string) Result {
	info, err := statFn(dir)
	switch {
	case err == nil:
		if !info.IsDir() {
			return Result{StatusFail, checkState, dir + " is not a folder; " + cannotStart}
		}
		writable, checked := writableDir(dir)
		switch {
		case !checked:
			return Result{StatusWarn, checkState, dir + " exists; writability not checked on this platform"}
		case !writable:
			return Result{StatusFail, checkState, dir + " is not writable; " + cannotStartNew}
		}
		return Result{StatusOK, checkState, dir + " exists and is writable"}
	case !errors.Is(err, os.ErrNotExist):
		return Result{StatusFail, checkState, "cannot read " + dir + ": " + errText(err, dir) + "; " + cannotStart}
	}

	// The directory does not exist: look for the nearest ancestor that does.
	anc := filepath.Dir(dir)
	for {
		ai, aerr := statFn(anc)
		if aerr == nil {
			writable, checked := writableDir(anc)
			switch {
			case !ai.IsDir():
				return Result{StatusFail, checkState, dir + " does not exist and " + anc + " is not a folder; " + cannotStart}
			case !checked:
				return Result{StatusWarn, checkState, dir + " does not exist yet; writability of " + anc + " not checked on this platform"}
			case !writable:
				return Result{StatusFail, checkState, dir + " does not exist and " + anc + " is not writable; " + cannotStart}
			}
			return Result{StatusOK, checkState, dir + " does not exist yet; kirsch will create it (" + anc + " is writable)"}
		}
		next := filepath.Dir(anc)
		if !errors.Is(aerr, os.ErrNotExist) || next == anc {
			return Result{StatusFail, checkState, dir + " does not exist and " + anc + " cannot be read: " + errText(aerr, anc) + "; " + cannotStart}
		}
		anc = next
	}
}

// rootProblem reports a state-root problem that would stop
// Layout.CreateSession, and whether there is one. It is called only after
// sessionDirResult was ok.
//
// A missing root is no problem: sessionDirResult has already found the nearest
// existing ancestor of the session directory, which is above the root, to be
// writable, so MkdirAll creates the root, and index.json cannot exist yet. An
// existing root must be a folder that passes access(2) for write and search,
// because WriteIndex creates a temporary file in it and renames it onto
// index.json. Then index.json is checked (indexProblem).
func rootProblem(root string) (Result, bool) {
	info, err := statFn(root)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return Result{}, false
	case err != nil:
		return Result{StatusFail, checkState, "cannot read " + root + ": " + errText(err, root) + "; " + cannotStart}, true
	case !info.IsDir():
		return Result{StatusFail, checkState, root + " is not a folder; " + cannotStart}, true
	}
	writable, checked := writableDir(root)
	switch {
	case !checked:
		return Result{StatusWarn, checkState, root + " exists; writability not checked on this platform"}, true
	case !writable:
		return Result{StatusFail, checkState, root + " is not writable, so index.json cannot be updated; " + cannotStartNew}, true
	}
	return indexProblem(session.IndexPath(root))
}

// indexProblem reports an index.json that ReadIndex could not read, and whether
// there is one. ReadIndex treats a missing file as an empty index, so missing
// is no problem. A folder in its place, a stat error, or a file that fails
// access(2) for read makes ReadIndex return an error, which fails
// Layout.CreateSession and, through Layout.LastSession, the resume path too.
//
// Doctor checks access only and never parses the file. A corrupt index.json
// also makes ReadIndex fail, and so would stop a start, but finding it would
// mean reading the user's session index, which is out of scope for this line.
func indexProblem(path string) (Result, bool) {
	info, err := statFn(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return Result{}, false
	case err != nil:
		return Result{StatusFail, checkState, "cannot read " + path + ": " + errText(err, path) + "; " + cannotStart}, true
	case info.IsDir():
		return Result{StatusFail, checkState, path + " is a folder, not a file; " + cannotStart}, true
	}
	readable, checked := readableFile(path)
	switch {
	case !checked:
		return Result{StatusWarn, checkState, path + " exists; readability not checked on this platform"}, true
	case !readable:
		return Result{StatusFail, checkState, path + " is not readable; " + cannotStart}, true
	}
	return Result{}, false
}

// errText renders err for a one-line detail. When err is an *fs.PathError for
// the path the detail already prints, only the underlying error is kept, so the
// path is not shown twice. The result always passes through oneLine, so a
// multi-line error cannot split the report line.
//
// Only the first *fs.PathError that errors.As finds is considered, and when it
// matches, the text of every wrapper around it goes too: an error such as
// "open lock file: open /p: denied", printed for /p, becomes "denied".
func errText(err error, printed string) string {
	var pe *fs.PathError
	if printed != "" && errors.As(err, &pe) && pe.Path == printed {
		return oneLine(pe.Err.Error())
	}
	return oneLine(err.Error())
}

// lockLine turns a ProbeLock outcome into a report line. It holds no I/O, so
// the wording and the one-line rule can be tested with injected values. A
// LockFree outcome is the plain "free"; lockResult refines it from the lock
// file on disk.
func lockLine(state session.LockState, err error) Result {
	switch {
	case err != nil:
		return Result{StatusWarn, checkLock, "could not check the lock: " + errText(err, "")}
	case state == session.LockHeld:
		return Result{StatusWarn, checkLock, "held by another kirsch in this workspace; a new start would skip resume and session grants"}
	case state == session.LockFree:
		return Result{StatusOK, checkLock, "free"}
	}
	return Result{StatusWarn, checkLock, "not checked on this platform"}
}

// lockResult reports whether another kirsch holds this workspace's lock, using
// session.ProbeLock, which creates nothing and never waits. dir is the session
// directory, used only to inspect the lock file once the lock reads free: a
// missing file is still ok, and a file the user cannot write is a warn, because
// session.Acquire opens it for writing and App.AcquireLock then logs "lock
// unavailable" and carries on with the single-instance guard off. The detail
// says the guard will be off rather than that kirsch starts, because whether
// it starts is the state line's question (a .lock file the user cannot write
// can sit in a session directory the user cannot write either).
func lockResult(root, workspace, dir string) Result {
	st, err := session.ProbeLock(root, workspace)
	res := lockLine(st, err)
	if err != nil || st != session.LockFree {
		return res
	}
	lockPath := filepath.Join(dir, ".lock")
	if _, serr := os.Stat(lockPath); errors.Is(serr, os.ErrNotExist) {
		return Result{StatusOK, checkLock, "free (no lock file yet)"}
	}
	if writable, checked := writableFile(lockPath); checked && !writable {
		return Result{StatusWarn, checkLock, "free, but " + lockPath + " is not writable; the single-instance guard will be off"}
	}
	return res
}
