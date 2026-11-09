package app

// Tests for m5-d17a: C51 (an empty session file left behind when the process
// quits during or after /new) and C52 (the provider session id surviving /new).
//
// Entry points: NewSession and Close are called directly, as the /new callback
// and the quit path in cmd/kirsch/main.go do. Turns are started through
// App.Submit.

import (
	"errors"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/agent"
	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/provider"
	"github.com/djm56/kirsch/internal/session"
	"github.com/djm56/kirsch/internal/telemetry"
)

// indexLast returns the workspace's last session id from index.json under the
// env's state root, or "" when there is no entry.
func (e *compactionEnv) indexLast(workspace string) string {
	e.t.Helper()
	idx, err := session.ReadIndex(e.dir)
	if err != nil {
		e.t.Fatalf("ReadIndex: %v", err)
	}
	return idx.Sessions[workspace].LastSessionID
}

// indexHas reports whether index.json holds any entry for the workspace. It
// reads the map directly: indexLast returns "" for a missing entry and for an
// entry with an empty id alike, so it cannot tell "deleted" from "blanked".
func (e *compactionEnv) indexHas(workspace string) bool {
	e.t.Helper()
	idx, err := session.ReadIndex(e.dir)
	if err != nil {
		e.t.Fatalf("ReadIndex: %v", err)
	}
	_, ok := idx.Sessions[workspace]
	return ok
}

// TestCloseAfterNewSessionLeavesNoEmptySession seeds a session with events,
// calls NewSession, writes nothing, then Close. The new session never held an
// event, so its file must be gone and index.json must point back at the
// seeded session, so the next start resumes a session that has content.
func TestCloseAfterNewSessionLeavesNoEmptySession(t *testing.T) {
	e := newCompactionEnv(t, 8192)
	e.seed(2, manualSeedTokens)
	oldID, oldPath := e.a.currentSessionID, e.logPath()
	workspace := e.a.layout.Workspace

	if err := e.a.NewSession(getenvKey); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if got := e.a.currentSessionID; got == oldID || got == "" {
		t.Fatalf("setup: id after /new = %q (old %q), want a fresh non-empty id", got, oldID)
	}
	e.a.Close()

	files := e.jsonlFiles()
	if len(files) != 1 || files[0] != oldPath {
		t.Errorf("session files after quit = %v, want only the seeded %s", files, oldPath)
	}
	if got := e.indexLast(workspace); got != oldID {
		t.Errorf("index.json points at %q after quit, want the seeded session %q", got, oldID)
	}
}

// TestCloseWhileNewSessionWaitsLeavesNoEmptySession starts a turn whose
// provider is slow to stop, calls NewSession (which blocks in inFlight.Wait
// with the new file already created), and Closes the app while it waits. When
// everything has settled there must be only the old session's file and
// index.json must point at it. This is a race test: Close itself waits on
// inFlight, so which of NewSession's branches runs first is not pinned. Either
// way NewSession must return nil or errAppClosing and leave no empty file.
// TestCloseBeforeInstallReturnsErrAppClosing pins the Close-first branches.
func TestCloseWhileNewSessionWaitsLeavesNoEmptySession(t *testing.T) {
	e := newCompactionEnv(t, 8192)
	lp := &lateProvider{started: make(chan struct{}), lag: 300 * time.Millisecond}
	e.a.providerBuilder = func(string, config.EndpointConfig, string, string, string, *telemetry.Logger) (provider.Provider, error) {
		return lp, nil
	}
	e.seed(2, manualSeedTokens)
	oldID, oldPath := e.a.currentSessionID, e.logPath()
	workspace := e.a.layout.Workspace

	e.a.Submit(compactionUserText, getenvKey)
	<-lp.started

	done := make(chan error, 1)
	go func() { done <- e.a.NewSession(getenvKey) }()

	// NewSession creates the new file before it waits: two files means it is
	// now in (or past) the wait.
	deadline := time.Now().Add(5 * time.Second)
	for len(e.jsonlFiles()) < 2 {
		if time.Now().After(deadline) {
			t.Fatal("NewSession did not create a second session file within 5s")
		}
		time.Sleep(2 * time.Millisecond)
	}
	e.a.Close()
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, errAppClosing) {
			t.Errorf("NewSession returned %v, want nil or errAppClosing", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("NewSession did not return within 5s of Close")
	}
	requireStoreMuFree(t, e.a, "NewSession racing Close")

	files := e.jsonlFiles()
	if len(files) != 1 || files[0] != oldPath {
		t.Errorf("session files after quit = %v, want only the old %s", files, oldPath)
	}
	if got := e.indexLast(workspace); got != oldID {
		t.Errorf("index.json points at %q after quit, want the old session %q", got, oldID)
	}
}

// TestRepeatedNewSessionThenCloseResumesTheRealSession calls NewSession twice
// with nothing written, then Close. The first empty session is replaced by the
// second, so index.json must still end up at the seeded session and no empty
// file may remain.
func TestRepeatedNewSessionThenCloseResumesTheRealSession(t *testing.T) {
	e := newCompactionEnv(t, 8192)
	e.seed(2, manualSeedTokens)
	oldID, oldPath := e.a.currentSessionID, e.logPath()
	workspace := e.a.layout.Workspace

	for i := 0; i < 2; i++ {
		if err := e.a.NewSession(getenvKey); err != nil {
			t.Fatalf("NewSession %d: %v", i+1, err)
		}
	}
	e.a.Close()

	files := e.jsonlFiles()
	if len(files) != 1 || files[0] != oldPath {
		t.Errorf("session files after quit = %v, want only the seeded %s", files, oldPath)
	}
	if got := e.indexLast(workspace); got != oldID {
		t.Errorf("index.json points at %q after quit, want the seeded session %q", got, oldID)
	}
}

// TestNewSessionRotatesTheProviderSessionID runs one turn, calls NewSession,
// runs a second turn, and compares the session id the provider builder
// received each time. /new must give the second turn a different id in the
// same format (32 lowercase hex characters).
func TestNewSessionRotatesTheProviderSessionID(t *testing.T) {
	e := newCompactionEnv(t, 8192, provider.Turn{Text: "one"}, provider.Turn{Text: "two"})
	fixed := strings.Repeat("ab", 16)
	e.a.SetSessionID(fixed)

	var mu sync.Mutex
	var ids []string
	e.a.providerBuilder = func(_ string, _ config.EndpointConfig, _, _, sessionID string, _ *telemetry.Logger) (provider.Provider, error) {
		mu.Lock()
		ids = append(ids, sessionID)
		mu.Unlock()
		return e.fake, nil
	}

	e.a.Submit("first", getenvKey)
	e.waitRequests(1)
	if err := e.a.NewSession(getenvKey); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	e.a.Submit("second", getenvKey)
	e.waitRequests(2)

	mu.Lock()
	defer mu.Unlock()
	if len(ids) != 2 {
		t.Fatalf("provider builder called %d times, want 2: %v", len(ids), ids)
	}
	if ids[0] != fixed {
		t.Errorf("first turn id = %q, want the id set before /new (%q)", ids[0], fixed)
	}
	if ids[1] == ids[0] {
		t.Errorf("provider session id %q unchanged by /new", ids[1])
	}
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(ids[1]) {
		t.Errorf("rotated id %q is not 32 lowercase hex characters", ids[1])
	}
}

// fileSize returns the size of path, failing the test if it cannot be stat'd.
func fileSize(t *testing.T, path string) int64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return info.Size()
}

// recordOne records one user message on the live store and flushes it, so the
// session file is non-empty on disk.
func recordOne(t *testing.T, a *App) {
	t.Helper()
	a.storeMu.Lock()
	store := a.store
	a.storeMu.Unlock()
	store.Record(agent.RecordAppend, agent.Message{
		Role:    agent.RoleUser,
		Content: []agent.Block{{Kind: agent.BlockText, Text: "kept"}},
	})
	if err := store.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
}

// TestNewSessionThatReceivedAnEventSurvivesClose is the positive control for
// the delete guard in abandonEmpty. It calls NewSession, records and flushes
// one event on the new store, then Close. The new session holds content, so
// its file must still exist and be non-empty, and index.json must point at it.
// The seeded session also survives.
func TestNewSessionThatReceivedAnEventSurvivesClose(t *testing.T) {
	e := newCompactionEnv(t, 8192)
	e.seed(2, manualSeedTokens)
	oldPath := e.logPath()
	workspace := e.a.layout.Workspace

	if err := e.a.NewSession(getenvKey); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	newID := e.a.currentSessionID
	newPath := session.SessionPath(e.a.layout.Root, workspace, newID)
	recordOne(t, e.a)
	e.a.Close()

	if got := fileSize(t, newPath); got == 0 {
		t.Errorf("new session file %s is empty after Close, want its event kept", newPath)
	}
	if got := fileSize(t, oldPath); got == 0 {
		t.Errorf("seeded session file %s is empty after Close", oldPath)
	}
	if got := e.indexLast(workspace); got != newID {
		t.Errorf("index.json points at %q, want the session that holds an event %q", got, newID)
	}
}

// TestResumedSessionIsNeverDeleted resumes a session whose file is empty (the
// size guard in abandonEmpty alone would not protect it) and checks that
// neither Close nor NewSession removes it. Entry points: ResumeLastSession,
// then Close in one App, then ResumeLastSession, NewSession and Close in
// another. The empty session NewSession creates and Close then abandons is
// the control: it shows the abandon path ran in the same test while the
// resumed file stayed.
func TestResumedSessionIsNeverDeleted(t *testing.T) {
	tmp := t.TempDir()
	seed := clearGrantsApp(t, tmp)
	// CreateSession points index.json at a new id and session.New creates the
	// file; closing it with no event leaves an empty file, as a session that
	// quit before its first event would.
	layout, err := session.NewLayout(tmp, seed.ws.CanonicalRoot)
	if err != nil {
		t.Fatalf("NewLayout: %v", err)
	}
	path, id, err := layout.CreateSession(time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	s, err := session.New(path)
	if err != nil {
		t.Fatalf("session.New: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close seeded store: %v", err)
	}
	if got := fileSize(t, path); got != 0 {
		t.Fatalf("setup: seeded file has %d bytes, want 0", got)
	}

	a := clearGrantsApp(t, tmp)
	if err := a.ResumeLastSession(); err != nil {
		t.Fatalf("ResumeLastSession: %v", err)
	}
	a.Close()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("resumed empty session removed by Close: %v", err)
	}

	b := clearGrantsApp(t, tmp)
	if err := b.ResumeLastSession(); err != nil {
		t.Fatalf("ResumeLastSession (second): %v", err)
	}
	if err := b.NewSession(getenvKey); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	newPath := session.SessionPath(b.layout.Root, b.layout.Workspace, b.currentSessionID)
	b.Close()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("resumed empty session removed by NewSession or Close: %v", err)
	}
	if _, err := os.Stat(newPath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("control: the empty /new session %s should be gone, stat err = %v", newPath, err)
	}
	idx, err := session.ReadIndex(tmp)
	if err != nil {
		t.Fatalf("ReadIndex: %v", err)
	}
	if got := idx.Sessions[b.layout.Workspace].LastSessionID; got != id {
		t.Errorf("index.json points at %q, want the resumed session %q", got, id)
	}
}

// TestFreshLaunchSessionIsAbandonedLikeANewOne covers sessions made by
// ensureStore (the startup path, reached through StartNewSession), not by
// NewSession. Entry points: StartNewSession, then NewSession and Close.
func TestFreshLaunchSessionIsAbandonedLikeANewOne(t *testing.T) {
	t.Run("empty startup then quit", func(t *testing.T) {
		e := newCompactionEnv(t, 8192)
		if err := e.a.StartNewSession(); err != nil {
			t.Fatalf("StartNewSession: %v", err)
		}
		workspace := e.a.layout.Workspace
		if n := len(e.jsonlFiles()); n != 1 {
			t.Fatalf("setup: %d session files after startup, want 1", n)
		}
		e.a.Close()
		if files := e.jsonlFiles(); len(files) != 0 {
			t.Errorf("session files after quit = %v, want none", files)
		}
		if e.indexHas(workspace) {
			t.Errorf("index.json still has an entry for the workspace (id %q), want it deleted", e.indexLast(workspace))
		}
	})
	t.Run("empty startup then /new then quit", func(t *testing.T) {
		e := newCompactionEnv(t, 8192)
		if err := e.a.StartNewSession(); err != nil {
			t.Fatalf("StartNewSession: %v", err)
		}
		workspace := e.a.layout.Workspace
		if err := e.a.NewSession(getenvKey); err != nil {
			t.Fatalf("NewSession: %v", err)
		}
		e.a.Close()
		if files := e.jsonlFiles(); len(files) != 0 {
			t.Errorf("session files after quit = %v, want none", files)
		}
		if e.indexHas(workspace) {
			t.Errorf("index.json still has an entry for the workspace (id %q), want it deleted", e.indexLast(workspace))
		}
	})
	t.Run("startup with an event then /new then quit (control)", func(t *testing.T) {
		e := newCompactionEnv(t, 8192)
		if err := e.a.StartNewSession(); err != nil {
			t.Fatalf("StartNewSession: %v", err)
		}
		workspace := e.a.layout.Workspace
		firstID := e.a.currentSessionID
		firstPath := session.SessionPath(e.a.layout.Root, workspace, firstID)
		recordOne(t, e.a)
		if err := e.a.NewSession(getenvKey); err != nil {
			t.Fatalf("NewSession: %v", err)
		}
		e.a.Close()
		files := e.jsonlFiles()
		if len(files) != 1 || files[0] != firstPath {
			t.Errorf("session files after quit = %v, want only the first %s", files, firstPath)
		}
		if got := e.indexLast(workspace); got != firstID {
			t.Errorf("index.json points at %q, want the first session %q", got, firstID)
		}
	})
}

// TestCloseBeforeInstallReturnsErrAppClosing holds NewSession after its old
// turn has stopped (beforeInstallHook, called directly by NewSession) and runs
// Close there, so Close provably runs while the new session is pending. It
// then lets NewSession go on to installSession, which must see closed. The
// outcome must be errAppClosing, only the seeded session's file left, and
// index.json pointing at it.
func TestCloseBeforeInstallReturnsErrAppClosing(t *testing.T) {
	e := newCompactionEnv(t, 8192)
	e.seed(2, manualSeedTokens)
	oldID, oldPath := e.a.currentSessionID, e.logPath()
	workspace := e.a.layout.Workspace

	reached := make(chan struct{})
	release := make(chan struct{})
	e.a.beforeInstallHook = func() {
		close(reached)
		<-release
	}
	done := make(chan error, 1)
	go func() { done <- e.a.NewSession(getenvKey) }()

	select {
	case <-reached:
	case <-time.After(5 * time.Second):
		t.Fatal("NewSession did not reach the pre-install hook within 5s")
	}
	if n := len(e.jsonlFiles()); n != 2 {
		t.Fatalf("setup: %d session files while NewSession is pending, want 2", n)
	}
	e.a.Close()
	close(release)

	select {
	case err := <-done:
		if !errors.Is(err, errAppClosing) {
			t.Errorf("NewSession returned %v, want errAppClosing", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("NewSession did not return within 5s")
	}
	requireStoreMuFree(t, e.a, "NewSession refused at install")
	files := e.jsonlFiles()
	if len(files) != 1 || files[0] != oldPath {
		t.Errorf("session files after quit = %v, want only the seeded %s", files, oldPath)
	}
	if got := e.indexLast(workspace); got != oldID {
		t.Errorf("index.json points at %q, want the seeded session %q", got, oldID)
	}
	if e.a.currentSessionID != oldID {
		t.Errorf("currentSessionID = %q after refused install, want %q", e.a.currentSessionID, oldID)
	}
}

// TestNewSessionAndStartAfterCloseAreRefused calls NewSession and
// StartNewSession after Close. Both must return errAppClosing and leave no
// session file and no index entry behind.
func TestNewSessionAndStartAfterCloseAreRefused(t *testing.T) {
	e := newCompactionEnv(t, 8192)
	workspace := e.a.ws.CanonicalRoot
	e.a.Close()

	if err := e.a.StartNewSession(); !errors.Is(err, errAppClosing) {
		t.Errorf("StartNewSession after Close = %v, want errAppClosing", err)
	}
	if err := e.a.NewSession(getenvKey); !errors.Is(err, errAppClosing) {
		t.Errorf("NewSession after Close = %v, want errAppClosing", err)
	}
	requireStoreMuFree(t, e.a, "NewSession after Close")
	if files := e.jsonlFiles(); len(files) != 0 {
		t.Errorf("session files = %v, want none", files)
	}
	idx, err := session.ReadIndex(e.dir)
	if err != nil {
		t.Fatalf("ReadIndex: %v", err)
	}
	if len(idx.Sessions) != 0 {
		t.Errorf("index.json = %v, want no entries (workspace %s)", idx.Sessions, workspace)
	}
}

// TestReplacedEmptySessionDoesNotMoveIndexOffTheNewOne is the control for the
// stale-index guard in abandonEmpty (entry.LastSessionID != origin.id). The
// startup session is empty and /new replaces it, so installSession abandons
// it while index.json already points at the new session. An event is then
// recorded on the new session and the app quits. If the guard were missing the
// abandon would restore the startup session's "no entry" over the new one, and
// index.json would no longer point at a session that holds content.
func TestReplacedEmptySessionDoesNotMoveIndexOffTheNewOne(t *testing.T) {
	e := newCompactionEnv(t, 8192)
	if err := e.a.StartNewSession(); err != nil {
		t.Fatalf("StartNewSession: %v", err)
	}
	workspace := e.a.layout.Workspace
	firstPath := session.SessionPath(e.a.layout.Root, workspace, e.a.currentSessionID)
	if err := e.a.NewSession(getenvKey); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	newID := e.a.currentSessionID
	newPath := session.SessionPath(e.a.layout.Root, workspace, newID)
	recordOne(t, e.a)
	e.a.Close()

	if _, err := os.Stat(firstPath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("control: the replaced empty startup session %s should be gone, stat err = %v", firstPath, err)
	}
	if got := fileSize(t, newPath); got == 0 {
		t.Errorf("new session file %s is empty after Close, want its event kept", newPath)
	}
	if got := e.indexLast(workspace); got != newID {
		t.Errorf("index.json points at %q, want the new session that holds an event %q", got, newID)
	}
}

// TestCloseAbandonsAPendingNewSessionBeforeTheLiveOne pins the order in Close.
// A seeded session S0 has content; /new makes empty session SA live; a second
// /new then creates SB and is held before install (beforeInstallHook), so SB
// is pending while SA is live. Close must abandon SB first (index.json goes
// back to SA), then SA (back to S0). In the other order SA is removed while
// the index still names SB, and SB's restore target SA is then gone, so the
// index would lose the workspace's entry instead of returning to S0.
func TestCloseAbandonsAPendingNewSessionBeforeTheLiveOne(t *testing.T) {
	e := newCompactionEnv(t, 8192)
	e.seed(2, manualSeedTokens)
	seedID, seedPath := e.a.currentSessionID, e.logPath()
	workspace := e.a.layout.Workspace
	if err := e.a.NewSession(getenvKey); err != nil {
		t.Fatalf("NewSession (first): %v", err)
	}

	reached := make(chan struct{})
	release := make(chan struct{})
	e.a.beforeInstallHook = func() {
		close(reached)
		<-release
	}
	done := make(chan error, 1)
	go func() { done <- e.a.NewSession(getenvKey) }()
	select {
	case <-reached:
	case <-time.After(5 * time.Second):
		t.Fatal("NewSession did not reach the pre-install hook within 5s")
	}
	if n := len(e.jsonlFiles()); n != 3 {
		t.Fatalf("setup: %d session files (seeded, live, pending), want 3", n)
	}
	e.a.Close()
	close(release)
	select {
	case err := <-done:
		if !errors.Is(err, errAppClosing) {
			t.Errorf("NewSession returned %v, want errAppClosing", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("NewSession did not return within 5s")
	}
	requireStoreMuFree(t, e.a, "NewSession refused at install")

	files := e.jsonlFiles()
	if len(files) != 1 || files[0] != seedPath {
		t.Errorf("session files after quit = %v, want only the seeded %s", files, seedPath)
	}
	if got := e.indexLast(workspace); got != seedID {
		t.Errorf("index.json points at %q after quit, want the seeded session %q", got, seedID)
	}
}

// TestCloseDeletesTheIndexEntryWhenTheRestoreTargetIsGone runs two Apps over
// one state root, as two Kirsch processes in concurrent mode would. A starts
// session SA; B starts SB, whose previous entry is SA. A quits and removes SA;
// B quits, finds the index naming SB and would restore SA, which no longer
// exists. The entry must be deleted rather than left pointing at SA, or the
// next launch fails with "session resume failed".
func TestCloseDeletesTheIndexEntryWhenTheRestoreTargetIsGone(t *testing.T) {
	tmp := t.TempDir()
	a := clearGrantsApp(t, tmp)
	b := clearGrantsApp(t, tmp)
	if err := a.StartNewSession(); err != nil {
		t.Fatalf("StartNewSession A: %v", err)
	}
	if err := b.StartNewSession(); err != nil {
		t.Fatalf("StartNewSession B: %v", err)
	}
	workspace := a.layout.Workspace
	a.Close()
	b.Close()

	idx, err := session.ReadIndex(tmp)
	if err != nil {
		t.Fatalf("ReadIndex: %v", err)
	}
	if entry, ok := idx.Sessions[workspace]; ok {
		t.Errorf("index.json still has an entry (id %q) after both quit, want it deleted", entry.LastSessionID)
	}
}

// TestCloseReleasesTheLockOnlyAfterTheAbandonedFileIsGone observes the lock
// from inside abandonEmpty (afterAbandonRemoveHook, which runs right after the
// empty file is removed): a second Acquire on the workspace must still report
// ErrLockHeld there. Once Close has returned the lock must be free, which is
// the positive control that the probe can succeed at all. Releasing the lock
// before the delete would let another Kirsch resume the file being removed.
func TestCloseReleasesTheLockOnlyAfterTheAbandonedFileIsGone(t *testing.T) {
	e := newCompactionEnv(t, 8192)
	if err := e.a.AcquireLock(); err != nil {
		t.Fatalf("AcquireLock: %v", err)
	}
	if e.a.lock == nil {
		t.Fatal("setup: AcquireLock left no lock held")
	}
	if err := e.a.StartNewSession(); err != nil {
		t.Fatalf("StartNewSession: %v", err)
	}
	workspace := e.a.ws.CanonicalRoot

	var calls int
	var probeErr error
	e.a.afterAbandonRemoveHook = func() {
		calls++
		l, err := session.Acquire(e.dir, workspace)
		if err == nil {
			_ = l.Release()
		}
		probeErr = err
	}
	e.a.Close()

	if calls != 1 {
		t.Fatalf("afterAbandonRemoveHook ran %d times, want 1 (the empty startup session)", calls)
	}
	if !errors.Is(probeErr, session.ErrLockHeld) {
		t.Errorf("probe during abandon returned %v, want ErrLockHeld (lock must still be held)", probeErr)
	}
	l, err := session.Acquire(e.dir, workspace)
	if err != nil {
		t.Fatalf("control: Acquire after Close = %v, want the lock released", err)
	}
	_ = l.Release()
}

// TestResumeLastSessionAfterCloseIsRefused seeds a session with content, quits,
// and calls ResumeLastSession on the closed App. Without the closed guard the
// resume would succeed (index.json names the seeded file) and open a store
// nobody closes; it must return errAppClosing and leave no live store.
func TestResumeLastSessionAfterCloseIsRefused(t *testing.T) {
	e := newCompactionEnv(t, 8192)
	e.seed(2, manualSeedTokens)
	e.a.Close()

	if err := e.a.ResumeLastSession(); !errors.Is(err, errAppClosing) {
		t.Errorf("ResumeLastSession after Close = %v, want errAppClosing", err)
	}
	if e.a.store != nil {
		t.Error("ResumeLastSession after Close left a live store")
	}
}

// TestCloseDuringNewSessionCreateCannotFreeTheLockFirst (review W1) starts
// Close at the moment NewSession has created its session file and index.json
// names it (afterCreateSessionHook, called by createSessionStore), and takes a
// snapshot of the disk as soon as Close returns, when the workspace lock is
// free. Entry points: NewSession, then Close on a second goroutine, as the /new
// callback and the quit path would race. At that snapshot only the seeded
// session may exist and index.json must name it: a NewSession file still on
// disk with the lock free is one another Kirsch could resume just before it is
// deleted.
//
// The hook decides whether to wait for Close. When storeMu is free at the hook
// (creation is not ordered against Close), it waits for Close to return, so
// the snapshot is taken while the new file exists. When storeMu is held (the
// current ordering), Close cannot pass storeMu until NewSession lets go, and
// waiting would deadlock, so it returns at once. The snapshot check is the same
// either way. Lock state is probed with session.Acquire; the probe succeeding
// is the control that the snapshot was taken with the lock free.
func TestCloseDuringNewSessionCreateCannotFreeTheLockFirst(t *testing.T) {
	e := newCompactionEnv(t, 8192)
	if err := e.a.AcquireLock(); err != nil {
		t.Fatalf("AcquireLock: %v", err)
	}
	if e.a.lock == nil {
		t.Fatal("setup: AcquireLock left no lock held")
	}
	e.seed(2, manualSeedTokens)
	seedID, seedPath := e.a.currentSessionID, e.logPath()
	workspace := e.a.layout.Workspace
	if got := fileSize(t, seedPath); got == 0 {
		t.Fatalf("setup: seeded session %s is empty", seedPath)
	}

	// Written by the Close goroutine before closeDone is closed; read after.
	var snapFiles []string
	var snapIndexID string
	var snapIndexErr, snapLockErr error
	closeDone := make(chan struct{})

	var calls int
	var waitTimedOut bool
	e.a.afterCreateSessionHook = func() {
		calls++
		if calls != 1 {
			return
		}
		guarded := !e.a.storeMu.TryLock()
		if !guarded {
			e.a.storeMu.Unlock()
		}
		go func() {
			e.a.Close()
			snapFiles = e.jsonlFiles()
			idx, err := session.ReadIndex(e.dir)
			snapIndexErr = err
			if err == nil {
				snapIndexID = idx.Sessions[workspace].LastSessionID
			}
			l, err := session.Acquire(e.dir, e.a.ws.CanonicalRoot)
			if err == nil {
				_ = l.Release()
			}
			snapLockErr = err
			close(closeDone)
		}()
		if !guarded {
			select {
			case <-closeDone:
			case <-time.After(5 * time.Second):
				waitTimedOut = true
			}
		}
	}

	err := e.a.NewSession(getenvKey)
	select {
	case <-closeDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not return within 5s of NewSession returning")
	}

	if calls != 1 {
		t.Fatalf("afterCreateSessionHook ran %d times, want 1 (the /new session)", calls)
	}
	if waitTimedOut {
		t.Fatal("storeMu was free at the hook but Close did not return within 5s")
	}
	if err != nil && !errors.Is(err, errAppClosing) {
		t.Errorf("NewSession returned %v, want nil or errAppClosing", err)
	}
	if snapLockErr != nil {
		t.Fatalf("control: Acquire right after Close returned = %v, want the lock free", snapLockErr)
	}
	if snapIndexErr != nil {
		t.Fatalf("ReadIndex at the snapshot: %v", snapIndexErr)
	}
	if len(snapFiles) != 1 || snapFiles[0] != seedPath {
		t.Errorf("with the lock already free, session files = %v, want only the seeded %s", snapFiles, seedPath)
	}
	if snapIndexID != seedID {
		t.Errorf("with the lock already free, index.json names %q, want the seeded session %q", snapIndexID, seedID)
	}
	if got := fileSize(t, seedPath); got == 0 {
		t.Errorf("seeded session %s is empty after Close", seedPath)
	}
}

// requireStoreMuFree fails the test if storeMu cannot be taken within 2s. It
// is called after an App method has returned, when nothing else should hold
// the lock, so a lock still held means an early return in that method kept
// it. On failure it releases the leaked lock (a sync.Mutex is not owned by a
// goroutine) so that the env's cleanup, which calls Close and takes storeMu,
// does not hang the test binary; then it stops the test.
func requireStoreMuFree(t *testing.T, a *App, after string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !a.storeMu.TryLock() {
		if time.Now().After(deadline) {
			a.storeMu.Unlock()
			t.Fatalf("storeMu still held 2s after %s returned: an early return kept the lock", after)
		}
		time.Sleep(time.Millisecond)
	}
	a.storeMu.Unlock()
}

// TestCloseDuringInstallWaitsForTheReplacedSessionToGo (review 2f W1) starts
// Close at the moment installSession has released storeMu and has not yet
// closed and abandoned the session it replaced (afterInstallUnlockHook, called
// by installSession). Entry points: NewSession, then Close on a second
// goroutine, as the /new callback and the quit path would race.
//
// Setup: a seeded session S0 with content, then /new making the empty session
// SA live. A second /new installs SB and replaces SA. In the hook:
//   - abandonMu must already be held, so Close's abandon steps cannot run
//     before SA is removed;
//   - Close is started and the hook waits until Close holds storeMu (the
//     positive control that Close really is running inside the window), then
//     up to 500ms more for Close to return. If Close returns while the hook
//     still holds installSession up, the workspace lock is free while SA is on
//     disk and, with SB abandoned, index.json names SA: another Kirsch could
//     resume SA just before installSession deletes it. That snapshot fails the
//     test. With the ordering in place Close is blocked in abandonEmpty and the
//     500ms passes with nothing to snapshot; Close returns only once the hook
//     lets installSession go on.
//
// When everything has settled only S0 may exist and index.json must name it.
func TestCloseDuringInstallWaitsForTheReplacedSessionToGo(t *testing.T) {
	e := newCompactionEnv(t, 8192)
	e.seed(2, manualSeedTokens)
	seedID, seedPath := e.a.currentSessionID, e.logPath()
	workspace := e.a.layout.Workspace
	if err := e.a.NewSession(getenvKey); err != nil {
		t.Fatalf("NewSession (first): %v", err)
	}
	replacedID := e.a.currentSessionID
	replacedPath := session.SessionPath(e.a.layout.Root, workspace, replacedID)
	if got := fileSize(t, replacedPath); got != 0 {
		t.Fatalf("setup: session %s to be replaced has %d bytes, want 0", replacedPath, got)
	}

	closeDone := make(chan struct{})
	var calls int
	var abandonMuFree, closeNeverHeldStoreMu, closeReturnedInWindow, snapOnDisk bool
	var snapIndexID string
	e.a.afterInstallUnlockHook = func() {
		calls++
		if calls != 1 {
			return
		}
		if e.a.abandonMu.TryLock() {
			e.a.abandonMu.Unlock()
			abandonMuFree = true
		}
		go func() {
			e.a.Close()
			close(closeDone)
		}()
		snapshot := func() {
			closeReturnedInWindow = true
			_, err := os.Stat(replacedPath)
			snapOnDisk = err == nil
			if idx, err := session.ReadIndex(e.dir); err == nil {
				snapIndexID = idx.Sessions[workspace].LastSessionID
			}
		}
		// Wait for Close to hold storeMu. Nothing else can: this goroutine is
		// NewSession's and holds only opMu and abandonMu here.
		deadline := time.After(5 * time.Second)
		for held := false; !held; {
			select {
			case <-closeDone:
				snapshot()
				return
			case <-deadline:
				closeNeverHeldStoreMu = true
				return
			case <-time.After(time.Millisecond):
				if e.a.storeMu.TryLock() {
					e.a.storeMu.Unlock()
				} else {
					held = true
				}
			}
		}
		select {
		case <-closeDone:
			snapshot()
		case <-time.After(500 * time.Millisecond):
		}
	}

	err := e.a.NewSession(getenvKey)
	select {
	case <-closeDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not return within 5s of NewSession returning")
	}

	if calls != 1 {
		t.Fatalf("afterInstallUnlockHook ran %d times, want 1 (the second /new)", calls)
	}
	if closeNeverHeldStoreMu {
		t.Fatal("control: Close neither took storeMu nor returned within 5s, so it never ran inside the window")
	}
	if abandonMuFree {
		t.Errorf("abandonMu was free when installSession released storeMu: Close can finish its abandon steps before the replaced session %s is removed", replacedPath)
	}
	if closeReturnedInWindow && snapOnDisk {
		t.Errorf("Close returned (workspace lock free) while the replaced empty session %s was still on disk; index.json named %q", replacedPath, snapIndexID)
	}
	if err != nil {
		t.Errorf("NewSession returned %v, want nil (it installed before Close)", err)
	}
	files := e.jsonlFiles()
	if len(files) != 1 || files[0] != seedPath {
		t.Errorf("session files after quit = %v, want only the seeded %s", files, seedPath)
	}
	if got := e.indexLast(workspace); got != seedID {
		t.Errorf("index.json points at %q after quit, want the seeded session %q", got, seedID)
	}
}

// TestCloseCannotSlipBetweenTheClosedCheckAndTheCreate (review 2f suggestion
// 1) starts Close inside createSessionStore, after its closed check and before
// it creates anything (beforeCreateSessionHook). Entry points: NewSession, and
// StartNewSession (ensureStore), each racing Close on a second goroutine.
//
// The hook probes storeMu with TryLock. If the lock is free there, the check
// and the creation are not in one hold: the hook then waits for Close to
// return, so Close runs wholly inside the gap, sets closed, finds nothing to
// abandon and frees the workspace lock; the file created afterwards is then
// never abandoned. If storeMu is held (the current code), Close cannot pass it
// until the caller lets go, and waiting would deadlock, so the hook returns at
// once. Either way, once everything has settled no session file made by the
// racing call may remain and index.json must be as it was before that call.
func TestCloseCannotSlipBetweenTheClosedCheckAndTheCreate(t *testing.T) {
	// race arms the hook to start Close, runs call, waits for Close, and
	// reports whether the hook saw storeMu free.
	race := func(t *testing.T, e *compactionEnv, call func() error) (gap bool, err error) {
		t.Helper()
		closeDone := make(chan struct{})
		var calls int
		e.a.beforeCreateSessionHook = func() {
			calls++
			if calls != 1 {
				return
			}
			gap = e.a.storeMu.TryLock()
			if gap {
				e.a.storeMu.Unlock()
			}
			go func() {
				e.a.Close()
				close(closeDone)
			}()
			if gap {
				<-closeDone
			}
		}
		err = call()
		select {
		case <-closeDone:
		case <-time.After(5 * time.Second):
			t.Fatal("Close did not return within 5s")
		}
		if calls != 1 {
			t.Fatalf("beforeCreateSessionHook ran %d times, want 1", calls)
		}
		return gap, err
	}

	t.Run("NewSession", func(t *testing.T) {
		e := newCompactionEnv(t, 8192)
		e.seed(2, manualSeedTokens)
		seedID, seedPath := e.a.currentSessionID, e.logPath()
		workspace := e.a.layout.Workspace

		gap, err := race(t, e, func() error { return e.a.NewSession(getenvKey) })
		if gap {
			t.Errorf("storeMu was free between the closed check and the create")
		}
		// Once NewSession has parked its session and let go of storeMu, it and
		// the waiting Close race for storeMu again: NewSession may install
		// first (nil) or Close may abandon the parked session first
		// (errAppClosing). Either way nothing may be left.
		if err != nil && !errors.Is(err, errAppClosing) {
			t.Errorf("NewSession returned %v, want nil or errAppClosing", err)
		}
		files := e.jsonlFiles()
		if len(files) != 1 || files[0] != seedPath {
			t.Errorf("session files after quit = %v, want only the seeded %s", files, seedPath)
		}
		if got := e.indexLast(workspace); got != seedID {
			t.Errorf("index.json points at %q, want the seeded session %q", got, seedID)
		}
	})
	t.Run("StartNewSession", func(t *testing.T) {
		e := newCompactionEnv(t, 8192)
		workspace := e.a.ws.CanonicalRoot

		gap, _ := race(t, e, e.a.StartNewSession)
		if gap {
			t.Errorf("storeMu was free between the closed check and the create")
		}
		// StartNewSession may win (Close then abandons its empty session) or
		// lose (errAppClosing); either way nothing may be left.
		if files := e.jsonlFiles(); len(files) != 0 {
			t.Errorf("session files after quit = %v, want none", files)
		}
		if e.indexHas(workspace) {
			t.Errorf("index.json still has an entry for the workspace (id %q), want none", e.indexLast(workspace))
		}
	})
}

// TestFailedStoreOpenDiscardsTheHalfMadeSession (review 2f suggestion 4)
// makes the store open fail after CreateSession has pointed index.json at the
// new id (openSessionStore, called by createSessionStore in place of
// session.New). Entry points: NewSession and StartNewSession. The half-made
// session must not be what the next start resumes: an empty file is removed,
// index.json is restored even when no file was created, and a non-empty file
// is left on disk (it is not this code's to delete) while index.json is still
// restored. The seeded session's file and index entry are the positive
// control that the restore went back to something that exists.
func TestFailedStoreOpenDiscardsTheHalfMadeSession(t *testing.T) {
	errInjected := errors.New("injected open failure")
	// failing returns an openSessionStore that writes content (nil for no
	// file) to path, records path, and fails.
	failing := func(content []byte, opened *string) func(string, ...session.Option) (*session.Store, error) {
		return func(path string, _ ...session.Option) (*session.Store, error) {
			*opened = path
			if content != nil {
				if err := os.WriteFile(path, content, 0o600); err != nil {
					return nil, err
				}
			}
			return nil, errInjected
		}
	}

	for _, tc := range []struct {
		name     string
		content  []byte
		wantFile bool
	}{
		{"empty file is removed", []byte{}, false},
		{"no file created", nil, false},
		{"non-empty file is kept", []byte("x"), true},
	} {
		t.Run("NewSession/"+tc.name, func(t *testing.T) {
			e := newCompactionEnv(t, 8192)
			e.seed(2, manualSeedTokens)
			seedID, seedPath := e.a.currentSessionID, e.logPath()
			workspace := e.a.layout.Workspace

			var opened string
			e.a.openSessionStore = failing(tc.content, &opened)
			err := e.a.NewSession(getenvKey)
			if !errors.Is(err, errInjected) {
				t.Fatalf("NewSession returned %v, want the injected open failure", err)
			}
			if opened == "" {
				t.Fatal("control: openSessionStore was never called")
			}
			requireStoreMuFree(t, e.a, "the failed NewSession")
			_, statErr := os.Stat(opened)
			if onDisk := statErr == nil; onDisk != tc.wantFile {
				t.Errorf("half-made session file %s on disk = %v, want %v", opened, onDisk, tc.wantFile)
			}
			if fileSize(t, seedPath) == 0 {
				t.Errorf("seeded session %s is empty", seedPath)
			}
			if got := e.indexLast(workspace); got != seedID {
				t.Errorf("index.json points at %q after the failed open, want the seeded session %q", got, seedID)
			}
			if e.a.currentSessionID != seedID {
				t.Errorf("currentSessionID = %q after the failed open, want %q", e.a.currentSessionID, seedID)
			}
		})
	}
	t.Run("StartNewSession/empty file is removed", func(t *testing.T) {
		e := newCompactionEnv(t, 8192)
		workspace := e.a.ws.CanonicalRoot
		var opened string
		e.a.openSessionStore = failing([]byte{}, &opened)
		if err := e.a.StartNewSession(); !errors.Is(err, errInjected) {
			t.Fatalf("StartNewSession returned %v, want the injected open failure", err)
		}
		if opened == "" {
			t.Fatal("control: openSessionStore was never called")
		}
		if files := e.jsonlFiles(); len(files) != 0 {
			t.Errorf("session files after the failed open = %v, want none", files)
		}
		if e.indexHas(workspace) {
			t.Errorf("index.json has an entry (id %q) after the failed open, want none", e.indexLast(workspace))
		}
		// Control: the same App can still create a session once the open works.
		e.a.openSessionStore = nil
		if err := e.a.StartNewSession(); err != nil {
			t.Fatalf("control: StartNewSession after the failure: %v", err)
		}
		if n := len(e.jsonlFiles()); n != 1 {
			t.Errorf("control: %d session files after a working StartNewSession, want 1", n)
		}
	})
}
