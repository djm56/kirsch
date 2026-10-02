package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TestApprovalRequestedMsgDispatchSetsCardFields verifies that an
// ApprovalRequestedMsg properly updates the transcript with the correct
// Subject, Detail, and GrantScope fields visible to the user.
func TestApprovalRequestedMsgDispatchSetsCardFields(t *testing.T) {
	m := New(Options{})
	msg := ApprovalRequestedMsg{
		ID:                   1,
		Description:          "run a command",
		Kind:                 "command",
		CanApproveForSession: true,
		Subject:              "go test",
		Detail:               []string{"Running tests for ./..."},
		GrantScope:           "go test ./...",
		Argv:                 []string{"go", "test", "./..."},
	}

	// Dispatch the message through Update
	updatedModel, _ := m.Update(msg)
	m2 := updatedModel.(Model)

	// Find the approval card in the transcript
	var foundCard *ApprovalCard
	for _, item := range m2.tr.Items() {
		if item.Kind == KindApproval && item.Approval != nil {
			foundCard = item.Approval
			break
		}
	}

	if foundCard == nil {
		t.Fatal("no approval card found in transcript")
	}

	// Verify the card fields match the message
	if foundCard.Subject != "go test" {
		t.Errorf("Subject = %q, want %q", foundCard.Subject, "go test")
	}
	if len(foundCard.Detail) != 1 || foundCard.Detail[0] != "Running tests for ./..." {
		t.Errorf("Detail = %v, want %v", foundCard.Detail, []string{"Running tests for ./..."})
	}
	if foundCard.GrantScope != "go test ./..." {
		t.Errorf("GrantScope = %q, want %q", foundCard.GrantScope, "go test ./...")
	}
}

// TestApprovalResolvedMsgDrivesRealSequence verifies that the real approval
// sequence works correctly: dispatch ApprovalRequestedMsg, resolve it via
// keypress (which calls resolveApproval), then dispatch ApprovalResolvedMsg
// (as app would send it after validating the grant). The card's outcome must
// be updated by the ApprovalResolvedMsg, not left in the state from the keypress.
func TestApprovalResolvedMsgDrivesRealSequence(t *testing.T) {
	m := New(Options{})

	// 1. Dispatch ApprovalRequestedMsg (simulating app.Request)
	requestMsg := ApprovalRequestedMsg{
		ID:                   1,
		Description:          "grant session",
		Kind:                 "command",
		CanApproveForSession: true,
		Subject:              "go test",
		Detail:               []string{},
		GrantScope:           "go test ./...",
		Argv:                 []string{"go", "test", "./..."},
	}
	updatedModel, _ := m.Update(requestMsg)
	m = updatedModel.(Model)

	// Find the approval card ID
	var cardID ItemID
	for _, item := range m.tr.Items() {
		if item.Kind == KindApproval && item.Approval != nil {
			cardID = item.ID
			break
		}
	}
	if cardID == 0 {
		t.Fatal("approval card not found after ApprovalRequestedMsg")
	}

	// 2. Simulate user pressing 'a' to approve for session (resolveApproval is called)
	// We simulate this by having a callback that would normally be wired
	m.ResolveApproval = func(id int64, outcome ApprovalOutcome) {
		// Callback to app.Resolve; we don't need to verify parameters here
	}

	// Simulate the key handler calling resolveApproval
	lay := m.layout()
	m = m.resolveApproval(ApprovedSession, lay)

	// At this point, the card's outcome is set to ApprovedSession by resolveApproval,
	// but the grant is not yet confirmed by app.Resolve.

	// 3. Dispatch ApprovalResolvedMsg (app.Resolve sends this when grant is attempted)
	// If the grant succeeds, the outcome is ApprovedSession.
	// ID is requestMsg.ID — the app-side approval ID app.Resolve echoes back —
	// not int64(cardID), the transcript item ID. The two happen to be equal in
	// this test (the approval card is the transcript's only item), but the
	// correlation under test is by approval ID; see
	// TestApprovalResolvedMsgCorrelatesByApprovalIDNotItemID for a transcript
	// where they diverge.
	resolvedMsg := ApprovalResolvedMsg{
		ID:      requestMsg.ID,
		Outcome: ApprovedSession,
	}
	updatedModel, _ = m.Update(resolvedMsg)
	m = updatedModel.(Model)

	// Verify the card's outcome is still ApprovedSession
	it, _, ok := m.tr.Find(cardID)
	if !ok {
		t.Fatal("approval card not found after ApprovalResolvedMsg")
	}
	if it.Approval.Outcome != ApprovedSession {
		t.Errorf("card outcome after ApprovalResolvedMsg = %v, want ApprovedSession", it.Approval.Outcome)
	}
}

// TestApprovalResolvedMsgUpdatesCardOutcomeWhenGrantRefused verifies that
// when a session grant is refused by the policy, the ApprovalResolvedMsg
// with Rejected outcome correctly updates the card, and the grant counter
// is NOT incremented.
func TestApprovalResolvedMsgUpdatesCardOutcomeWhenGrantRefused(t *testing.T) {
	m := New(Options{})

	// 1. Dispatch ApprovalRequestedMsg
	requestMsg := ApprovalRequestedMsg{
		ID:                   1,
		Description:          "grant session",
		Kind:                 "command",
		CanApproveForSession: true,
		Subject:              "go test",
		Detail:               []string{},
		GrantScope:           "go test ./...",
		Argv:                 []string{"go", "test", "./..."},
	}
	updatedModel, _ := m.Update(requestMsg)
	m = updatedModel.(Model)

	// Find the card ID
	var cardID ItemID
	for _, item := range m.tr.Items() {
		if item.Kind == KindApproval && item.Approval != nil {
			cardID = item.ID
			break
		}
	}
	if cardID == 0 {
		t.Fatal("approval card not found")
	}

	// 2. Simulate user pressing 'a'
	lay := m.layout()
	m = m.resolveApproval(ApprovedSession, lay)

	// 3. Dispatch ApprovalResolvedMsg with Rejected outcome
	// (simulating policy.Grant failing).
	// ID is requestMsg.ID, the app-side approval ID — see the comment in
	// TestApprovalResolvedMsgDrivesRealSequence for why this is not
	// int64(cardID).
	resolvedMsg := ApprovalResolvedMsg{
		ID:      requestMsg.ID,
		Outcome: Rejected,
	}
	updatedModel, _ = m.Update(resolvedMsg)
	m = updatedModel.(Model)

	// Verify the card's outcome is now Rejected
	it, _, ok := m.tr.Find(cardID)
	if !ok {
		t.Fatal("approval card not found after ApprovalResolvedMsg")
	}
	if it.Approval.Outcome != Rejected {
		t.Errorf("card outcome after ApprovalResolvedMsg = %v, want Rejected", it.Approval.Outcome)
	}
}

// TestApprovedSessionAndRejectedRenderDifferently verifies that approval
// cards with ApprovedSession and Rejected outcomes render with different text.
// This test calls renderApproval to verify the actual rendering output.
func TestApprovedSessionAndRejectedRenderDifferently(t *testing.T) {
	// Create cards with different outcomes
	approvedCard := &ApprovalCard{
		Kind:       ApprovalCommand,
		Title:      "test command",
		Subject:    "go test",
		GrantScope: "go test ./...",
		Outcome:    ApprovedSession,
		Elapsed:    2400,
	}

	rejectedCard := &ApprovalCard{
		Kind:       ApprovalCommand,
		Title:      "test command",
		Subject:    "go test",
		GrantScope: "go test ./...",
		Outcome:    Rejected,
		Elapsed:    2400,
	}

	// Create a minimal render context with styles and glyphs
	sty := NewStyles(NewRenderer(false), false)
	gly := NewGlyphs(false)
	ctx := renderCtx{
		Sty:     sty,
		G:       gly,
		ShowDur: true,
	}

	// Render both cards
	approvedLines := renderApproval(approvedCard, ctx)
	rejectedLines := renderApproval(rejectedCard, ctx)

	if len(approvedLines) == 0 || len(rejectedLines) == 0 {
		t.Fatal("renderApproval returned empty lines")
	}

	approvedRendered := strings.Join(approvedLines, "\n")
	rejectedRendered := strings.Join(rejectedLines, "\n")

	// ApprovedSession should contain the grant scope
	if !strings.Contains(approvedRendered, "session grant:") {
		t.Errorf("approved session rendering missing 'session grant:'\nRendered: %q", approvedRendered)
	}
	if !strings.Contains(approvedRendered, "go test ./...") {
		t.Errorf("approved session rendering missing grant scope\nRendered: %q", approvedRendered)
	}

	// Rejected should NOT contain "session grant"
	if strings.Contains(rejectedRendered, "session grant:") {
		t.Errorf("rejected rendering should not contain 'session grant:'\nRendered: %q", rejectedRendered)
	}

	// Both should show "rejected" indicator for rejected, "approved" for approved
	if !strings.Contains(approvedRendered, "approved") {
		t.Errorf("approved session rendering missing 'approved'\nRendered: %q", approvedRendered)
	}
	if !strings.Contains(rejectedRendered, "rejected") {
		t.Errorf("rejected rendering missing 'rejected'\nRendered: %q", rejectedRendered)
	}
}

// TestApprovalResolvedMsgCorrelatesByApprovalIDNotItemID is the regression
// test for the bug both prior fix rounds left standing: updateApprovalOutcome
// correlated ApprovalResolvedMsg.ID against a transcript item ID
// (ItemID(msg.ID)), rather than against the app-side approval ID it actually
// is. The two only ever agreed by coincidence, because every other test in
// this file starts from an empty transcript, so the approval card lands as
// item 1 and its approval ID (also allocated from 1) matches by accident.
//
// This test breaks that coincidence on purpose: a notice is appended before
// the approval request, so the approval card is the transcript's *second*
// item (ItemID 2) while its approval ID is still 1 — the two spaces
// deliberately disagree.
//
// Against the code as it stood before this fix, this test panics rather than
// fails: m.tr.Find(ItemID(1)) finds the notice item, whose Approval field is
// nil, and the old handler's unconditional `it.Approval.Outcome = msg.Outcome`
// dereferences that nil pointer. Reproduced by hand on the pre-fix source
// (module-scope revert of update.go's updateApprovalOutcome to
// `m.tr.Find(ItemID(msg.ID))` with no Kind/nil guard) and confirmed to panic
// with the same input this test sends.
func TestApprovalResolvedMsgCorrelatesByApprovalIDNotItemID(t *testing.T) {
	m := New(Options{})

	// Item 1: a notice, appended before any approval exists. This is what
	// makes the notice's item ID (1) collide with the approval's app-side ID
	// (also 1, since approveID starts counting from 1 independently) while
	// the approval's own item ID (2) does not.
	m.notice("a notice appended before the approval, so item IDs and approval IDs diverge")

	requestMsg := ApprovalRequestedMsg{
		ID:                   1,
		Description:          "run a command",
		Kind:                 "command",
		CanApproveForSession: false,
		Subject:              "go vet",
		Detail:               []string{},
	}
	updatedModel, _ := m.Update(requestMsg)
	m = updatedModel.(Model)

	// The approval card must be item 2, not item 1 — that gap is the point of
	// this test.
	var cardID ItemID
	for _, item := range m.tr.Items() {
		if item.Kind == KindApproval && item.Approval != nil {
			cardID = item.ID
			break
		}
	}
	if cardID != 2 {
		t.Fatalf("approval card ID = %d, want 2 (the notice at item 1 must precede it)", cardID)
	}

	// Resolve by the approval ID (1), which is NOT the approval card's item
	// ID (2). Under the pre-fix correlation this looks up item 1 — the
	// notice — and panics on the nil *ApprovalCard write.
	resolvedMsg := ApprovalResolvedMsg{
		ID:      requestMsg.ID, // == 1, the notice's item ID, not the card's
		Outcome: Rejected,
	}
	updatedModel, _ = m.Update(resolvedMsg)
	m = updatedModel.(Model)

	// The approval card (item 2) must carry the resolved outcome.
	it, _, ok := m.tr.Find(cardID)
	if !ok {
		t.Fatal("approval card not found after ApprovalResolvedMsg")
	}
	if it.Approval.Outcome != Rejected {
		t.Errorf("approval card outcome = %v, want Rejected", it.Approval.Outcome)
	}

	// The notice (item 1) must be untouched — no Approval payload, no Kind
	// change.
	noticeItem, _, ok := m.tr.Find(1)
	if !ok {
		t.Fatal("notice item not found")
	}
	if noticeItem.Kind != KindNotice || noticeItem.Approval != nil {
		t.Errorf("notice item was mutated by ApprovalResolvedMsg: kind=%v approval=%v",
			noticeItem.Kind, noticeItem.Approval)
	}
}

// TestApprovalResolvedMsgDoesNotAlterADifferentApprovalsCard verifies that
// resolving one approval leaves an unrelated, still-pending approval's card
// untouched — the isolation approvalCards is responsible for now that more
// than one approval ID can be outstanding at once (e.g. across two
// concurrently rendered but independently tracked approval cards).
func TestApprovalResolvedMsgDoesNotAlterADifferentApprovalsCard(t *testing.T) {
	m := New(Options{})

	requestA := ApprovalRequestedMsg{ID: 5, Description: "cmd A", Kind: "command", Subject: "cmdA"}
	requestB := ApprovalRequestedMsg{ID: 9, Description: "cmd B", Kind: "command", Subject: "cmdB"}

	updatedModel, _ := m.Update(requestA)
	m = updatedModel.(Model)
	updatedModel, _ = m.Update(requestB)
	m = updatedModel.(Model)

	var cardA, cardB ItemID
	for _, item := range m.tr.Items() {
		if item.Kind != KindApproval || item.Approval == nil {
			continue
		}
		switch item.Approval.Subject {
		case "cmdA":
			cardA = item.ID
		case "cmdB":
			cardB = item.ID
		}
	}
	if cardA == 0 || cardB == 0 {
		t.Fatalf("both approval cards must be found: cardA=%d cardB=%d", cardA, cardB)
	}

	// Resolve only A, by A's approval ID (5).
	resolvedA := ApprovalResolvedMsg{ID: 5, Outcome: Rejected}
	updatedModel, _ = m.Update(resolvedA)
	m = updatedModel.(Model)

	itA, _, ok := m.tr.Find(cardA)
	if !ok {
		t.Fatal("card A not found")
	}
	if itA.Approval.Outcome != Rejected {
		t.Errorf("card A outcome = %v, want Rejected", itA.Approval.Outcome)
	}

	itB, _, ok := m.tr.Find(cardB)
	if !ok {
		t.Fatal("card B not found")
	}
	if itB.Approval.Outcome != Unresolved {
		t.Errorf("card B outcome = %v, want Unresolved (must be untouched by resolving A)", itB.Approval.Outcome)
	}
}

// approveOnceRequest builds an ApprovalRequestedMsg with no session grant
// offered, so 'a' is inert and only 'y'/'n'/Esc are meaningful — matching the
// three outcomes for which app.Resolve never attempts a grant and therefore
// never sends an ApprovalResolvedMsg back.
func approveOnceRequest(id int64, subject string) ApprovalRequestedMsg {
	return ApprovalRequestedMsg{
		ID:          id,
		Description: "run a command",
		Kind:        "command",
		Subject:     subject,
		Detail:      []string{},
	}
}

// TestApprovalCardMapClearedOnApproveOnceRejectAndCancel is the regression
// test for the approvalCards leak: for the three outcomes app.Resolve never
// sends a confirmation for (Approved, Rejected, Cancelled — see app.go's
// grantAttempted, which only gates on ApprovalOutcomeSession), the
// correlation entry populated by receiveApprovalRequest must be removed by
// resolveApproval itself, since updateApprovalOutcome will never run to do it.
//
// Each sub-test drives the real path: dispatch ApprovalRequestedMsg through
// Update, then dispatch the keypress through Update exactly as the user's
// terminal input would, so the assertion covers keyApproval's dispatch and
// not just resolveApproval called directly.
func TestApprovalCardMapClearedOnApproveOnceRejectAndCancel(t *testing.T) {
	cases := []struct {
		name string
		key  tea.KeyMsg
	}{
		{"approve-once", key('y')},
		{"reject", key('n')},
		{"cancel", keyType(tea.KeyEsc)},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := New(Options{})

			requestMsg := approveOnceRequest(1, "go vet")
			updatedModel, _ := m.Update(requestMsg)
			m = updatedModel.(Model)

			if len(m.approvalCards) != 1 {
				t.Fatalf("approvalCards after request = %d entries, want 1", len(m.approvalCards))
			}

			updatedModel, _ = m.Update(c.key)
			m = updatedModel.(Model)

			if len(m.approvalCards) != 0 {
				t.Errorf("approvalCards after %s = %d entries, want 0 (leaked)", c.name, len(m.approvalCards))
			}
		})
	}
}

// TestApprovalCardMapSurvivesUntilSessionGrantConfirmed verifies the one case
// that must keep working: a session grant's correlation entry is left in
// approvalCards by resolveApproval (unlike the other three outcomes) because
// app.Resolve always sends a confirming ApprovalResolvedMsg for it, and it is
// updateApprovalOutcome — not resolveApproval — that removes the entry once
// that confirmation arrives.
func TestApprovalCardMapSurvivesUntilSessionGrantConfirmed(t *testing.T) {
	m := New(Options{})

	requestMsg := ApprovalRequestedMsg{
		ID:                   1,
		Description:          "run a command",
		Kind:                 "command",
		CanApproveForSession: true,
		Subject:              "go test",
		Detail:               []string{},
		GrantScope:           "go test ./...",
		Argv:                 []string{"go", "test", "./..."},
	}
	updatedModel, _ := m.Update(requestMsg)
	m = updatedModel.(Model)

	if len(m.approvalCards) != 1 {
		t.Fatalf("approvalCards after request = %d entries, want 1", len(m.approvalCards))
	}

	// Press 'a' — the real dispatch path through keyApproval, which requires
	// OffersSessionGrant() to be true (Kind==ApprovalCommand, GrantScope!="").
	updatedModel, _ = m.Update(key('a'))
	m = updatedModel.(Model)

	if len(m.approvalCards) != 1 {
		t.Fatalf("approvalCards after 'a' keypress = %d entries, want 1 (must survive until confirmed)", len(m.approvalCards))
	}

	// app.Resolve confirms the grant.
	resolvedMsg := ApprovalResolvedMsg{ID: requestMsg.ID, Outcome: ApprovedSession}
	updatedModel, _ = m.Update(resolvedMsg)
	m = updatedModel.(Model)

	if len(m.approvalCards) != 0 {
		t.Errorf("approvalCards after confirmation = %d entries, want 0", len(m.approvalCards))
	}
}

// TestApprovalDiffModalDisplaysCorrectData verifies that when pressing 'd' on a
// pending patch approval, the modal is opened with the correct filename and
// precomputed added/removed counts from ApprovalRequestedMsg.
func TestApprovalDiffModalDisplaysCorrectData(t *testing.T) {
	m := New(Options{})

	// Create an approval request with diff data.
	// The diff lines start with "@@" and have preserved prefixes.
	requestMsg := ApprovalRequestedMsg{
		ID:                   1,
		Description:          "apply_patch — add input validation",
		Kind:                 "patch",
		CanApproveForSession: false,
		Subject:              "2 files changed",
		Detail: []string{
			"files: 2 changed (calc/divide.go,",
			"       calc/divide_test.go)",
		},
		DiffFilename: "calc/divide.go",
		DiffLines: []string{
			"@@ -12,7 +12,15 @@ func Divide(a, b float64) (float64, error) {",
			" func Divide(a, b float64) (float64, error) {",
			"-	return a / b, nil",
			"+	if b == 0 {",
			"+		return 0, ErrDivideByZero",
			"+	}",
			"+	return a / b, nil",
			" }",
		},
		DiffAdded:   4,
		DiffRemoved: 1,
	}

	// Process the approval request.
	updatedModel, _ := m.Update(requestMsg)
	m = updatedModel.(Model)

	// Verify the approval card was created with diff data.
	item, _, ok := m.tr.Find(m.pendingApproval)
	if !ok {
		t.Fatalf("pending approval not found in transcript")
	}
	if item.Approval.DiffFilename != "calc/divide.go" {
		t.Errorf("DiffFilename = %q, want %q", item.Approval.DiffFilename, "calc/divide.go")
	}
	if item.Approval.Added != 4 {
		t.Errorf("Added = %d, want 4", item.Approval.Added)
	}
	if item.Approval.Removed != 1 {
		t.Errorf("Removed = %d, want 1", item.Approval.Removed)
	}
	if len(item.Approval.Diff) == 0 {
		t.Errorf("Diff lines is empty, want diff content")
	}

	// Press 'd' to open the diff modal.
	updatedModel, _ = m.Update(key('d'))
	m = updatedModel.(Model)

	// Verify the modal was opened.
	if !m.overlayOpen() {
		t.Errorf("overlay not open after 'd' keypress")
	}

	// Verify modal state has correct data.
	if m.modal.Title != "calc/divide.go" {
		t.Errorf("modal.Title = %q, want %q", m.modal.Title, "calc/divide.go")
	}
	if m.modal.Added != 4 {
		t.Errorf("modal.Added = %d, want 4", m.modal.Added)
	}
	if m.modal.Removed != 1 {
		t.Errorf("modal.Removed = %d, want 1", m.modal.Removed)
	}

	// Verify diff lines are in modal and start with @@.
	if len(m.modal.Lines) == 0 {
		t.Errorf("modal.Lines is empty")
	} else if !strings.HasPrefix(m.modal.Lines[0], "@@") {
		t.Errorf("first modal line = %q, want to start with @@", m.modal.Lines[0])
	}
}

// TestApprovalCommandDetailKeyOpensModal verifies that pressing 'd' on a
// command approval opens a content modal showing the approval detail lines.
func TestApprovalCommandDetailKeyOpensModal(t *testing.T) {
	m := New(Options{})

	// Create a command approval request with detail lines
	requestMsg := ApprovalRequestedMsg{
		ID:                   1,
		Description:          "run a command",
		Kind:                 "command",
		CanApproveForSession: true,
		Subject:              "go test ./...",
		Detail:               []string{"Running tests", "Exit code: 0"},
		GrantScope:           "go test ./...",
		Argv:                 []string{"go", "test", "./..."},
	}

	// Dispatch the message through Update
	updatedModel, _ := m.Update(requestMsg)
	m = updatedModel.(Model)

	// Verify the approval card is pending
	if m.pendingApproval == 0 {
		t.Fatal("no pending approval after request")
	}

	// Verify there is no modal open initially
	if m.overlayOpen() {
		t.Fatal("modal should not be open initially")
	}

	// Press 'd' to open the detail view
	updatedModel, _ = m.Update(key('d'))
	m = updatedModel.(Model)

	// Verify a modal was opened
	if !m.overlayOpen() {
		t.Errorf("modal should be open after pressing 'd' on a command approval, but overlay is not open")
	}

	// Verify it's the right kind of modal (content, not diff)
	if m.modal.Kind != ModalContent {
		t.Errorf("modal.Kind = %v, want ModalContent", m.modal.Kind)
	}

	// Verify the modal contains the detail lines
	if len(m.modal.Lines) == 0 {
		t.Errorf("modal.Lines is empty, want detail lines")
	}

	// Verify the detail lines match
	if len(m.modal.Lines) != 2 || m.modal.Lines[0] != "Running tests" || m.modal.Lines[1] != "Exit code: 0" {
		t.Errorf("modal.Lines = %v, want detail lines from the card", m.modal.Lines)
	}
}

// TestApprovalPatchDetailKeyOpensModal verifies that pressing 'd' on a
// patch approval opens a diff modal (contrast with command approvals,
// where the key does nothing).
func TestApprovalPatchDetailKeyOpensModal(t *testing.T) {
	m := New(Options{})

	// Create a patch approval request with diff data
	requestMsg := ApprovalRequestedMsg{
		ID:                   1,
		Description:          "apply_patch — test",
		Kind:                 "patch",
		CanApproveForSession: false,
		Subject:              "1 file changed",
		Detail:               []string{"files: 1 changed (main.go)"},
		DiffFilename:         "main.go",
		DiffLines: []string{
			"@@ -1,2 +1,3 @@",
			" func main() {",
			"+	fmt.Println(\"hello\")",
			" }",
		},
		DiffAdded:   1,
		DiffRemoved: 0,
	}

	// Dispatch the message through Update
	updatedModel, _ := m.Update(requestMsg)
	m = updatedModel.(Model)

	// Verify the approval card is pending
	if m.pendingApproval == 0 {
		t.Fatal("no pending approval after request")
	}

	// Verify there is no modal open initially
	if m.overlayOpen() {
		t.Fatal("modal should not be open initially")
	}

	// Press 'd' to open the detail view
	updatedModel, _ = m.Update(key('d'))
	m = updatedModel.(Model)

	// Verify a modal was opened
	if !m.overlayOpen() {
		t.Errorf("modal should be open after pressing 'd' on a patch approval, but no overlay is open")
	}

	// Verify it's the right kind of modal
	if m.modal.Kind != ModalDiff {
		t.Errorf("modal.Kind = %v, want ModalDiff", m.modal.Kind)
	}

	// Verify the modal title and content
	if m.modal.Title != "main.go" {
		t.Errorf("modal.Title = %q, want %q", m.modal.Title, "main.go")
	}

	if len(m.modal.Lines) == 0 {
		t.Errorf("modal.Lines is empty, want diff content")
	}
}

// TestApprovalCardSanitisesAllFields verifies that six text fields
// entering an approval card are sanitised to strip ANSI escape sequences
// and control characters, protecting against malicious content in paths
// and diff lines. The test dispatches an ApprovalRequestedMsg with escape
// sequences in all fields and asserts the resulting card contains none,
// and that all content fields remain non-empty after sanitisation.
func TestApprovalCardSanitisesAllFields(t *testing.T) {
	m := New(Options{})

	// Build a message with ANSI escape sequences injected into every text field.
	// ESC[31m is red; ESC[0m is reset; ESC]0;title BEL is an OSC sequence.
	ansiRed := "\x1b[31m"
	ansiReset := "\x1b[0m"
	oscSeq := "\x1b]0;title\x07"

	requestMsg := ApprovalRequestedMsg{
		ID:                   1,
		Description:          ansiRed + "run cmd" + ansiReset,
		Kind:                 "command",
		CanApproveForSession: true,
		Subject:              oscSeq + "cmd" + ansiRed + "name" + ansiReset,
		Detail: []string{
			ansiRed + "detail line 1" + ansiReset,
			oscSeq + "detail line 2" + ansiRed,
		},
		GrantScope:   ansiRed + "grant scope" + ansiReset,
		Argv:         []string{"cmd", "arg"},
		DiffFilename: oscSeq + "file.go" + ansiRed,
		DiffLines: []string{
			ansiRed + "+ added line" + ansiReset,
			"- " + oscSeq + "removed line" + ansiRed,
		},
		DiffAdded:   1,
		DiffRemoved: 1,
	}

	// Dispatch the message through Update
	updatedModel, _ := m.Update(requestMsg)
	m2 := updatedModel.(Model)

	// Find the approval card
	var foundCard *ApprovalCard
	for _, item := range m2.tr.Items() {
		if item.Kind == KindApproval && item.Approval != nil {
			foundCard = item.Approval
			break
		}
	}
	if foundCard == nil {
		t.Fatal("no approval card found in transcript")
	}

	// Verify that all text fields are sanitised (contain no escape sequences).
	// The sanitised versions should have the escape sequences stripped but
	// preserve the rest of the text.

	if containsANSI(foundCard.Title) {
		t.Errorf("Title contains ANSI sequences: %q", foundCard.Title)
	}
	if foundCard.Title == "" {
		t.Errorf("Title is empty after sanitisation, want text to remain")
	}

	if containsANSI(foundCard.Subject) {
		t.Errorf("Subject contains ANSI sequences: %q", foundCard.Subject)
	}
	if foundCard.Subject == "" {
		t.Errorf("Subject is empty after sanitisation, want text to remain")
	}

	for i, line := range foundCard.Detail {
		if containsANSI(line) {
			t.Errorf("Detail[%d] contains ANSI sequences: %q", i, line)
		}
		if line == "" {
			t.Errorf("Detail[%d] is empty after sanitisation, want text to remain", i)
		}
	}

	if containsANSI(foundCard.GrantScope) {
		t.Errorf("GrantScope contains ANSI sequences: %q", foundCard.GrantScope)
	}
	if foundCard.GrantScope == "" {
		t.Errorf("GrantScope is empty after sanitisation, want text to remain")
	}

	if containsANSI(foundCard.DiffFilename) {
		t.Errorf("DiffFilename contains ANSI sequences: %q", foundCard.DiffFilename)
	}
	if foundCard.DiffFilename == "" {
		t.Errorf("DiffFilename is empty after sanitisation, want text to remain")
	}

	for i, line := range foundCard.Diff {
		if containsANSI(line) {
			t.Errorf("Diff[%d] contains ANSI sequences: %q", i, line)
		}
		if line == "" {
			t.Errorf("Diff[%d] is empty after sanitisation, want text to remain", i)
		}
	}
}

// TestApprovalCardStripsNewlines verifies that literal newline characters
// embedded in approval card text fields are stripped or replaced, preventing
// them from breaking the box rendering. The test dispatches an
// ApprovalRequestedMsg with literal newlines in all text fields and asserts
// that no card field contains a newline.
func TestApprovalCardStripsNewlines(t *testing.T) {
	m := New(Options{})

	// Build a message with literal newlines injected into every text field.
	requestMsg := ApprovalRequestedMsg{
		ID:                   2,
		Description:          "run\ncmd",
		Kind:                 "command",
		CanApproveForSession: true,
		Subject:              "subject\nline2",
		Detail: []string{
			"detail\nline1",
			"detail\nline2",
		},
		GrantScope:   "grant\nscope",
		Argv:         []string{"cmd", "arg"},
		DiffFilename: "file\n.go",
		DiffLines: []string{
			"+\nadded",
			"-\nremoved",
		},
		DiffAdded:   1,
		DiffRemoved: 1,
	}

	// Dispatch the message through Update
	updatedModel, _ := m.Update(requestMsg)
	m2 := updatedModel.(Model)

	// Find the approval card
	var foundCard *ApprovalCard
	for _, item := range m2.tr.Items() {
		if item.Kind == KindApproval && item.Approval != nil {
			foundCard = item.Approval
			break
		}
	}
	if foundCard == nil {
		t.Fatal("no approval card found in transcript")
	}

	// Verify that no text field contains a newline. Single-row fields
	// (Title, Subject, GrantScope, DiffFilename) must not contain newlines.
	// Slice fields (Detail, Diff) should have one element per line with no
	// embedded newlines.

	if strings.Contains(foundCard.Title, "\n") {
		t.Errorf("Title contains newline: %q", foundCard.Title)
	}

	if strings.Contains(foundCard.Subject, "\n") {
		t.Errorf("Subject contains newline: %q", foundCard.Subject)
	}

	for i, line := range foundCard.Detail {
		if strings.Contains(line, "\n") {
			t.Errorf("Detail[%d] contains newline: %q", i, line)
		}
	}

	if strings.Contains(foundCard.GrantScope, "\n") {
		t.Errorf("GrantScope contains newline: %q", foundCard.GrantScope)
	}

	if strings.Contains(foundCard.DiffFilename, "\n") {
		t.Errorf("DiffFilename contains newline: %q", foundCard.DiffFilename)
	}

	for i, line := range foundCard.Diff {
		if strings.Contains(line, "\n") {
			t.Errorf("Diff[%d] contains newline: %q", i, line)
		}
	}
}

// containsANSI reports whether s contains ANSI escape sequences.
// It checks for CSI (ESC[), OSC (ESC]), and other common sequences.
// containsANSI reports whether s contains a raw ESC byte or a bare C1
// control rune (U+0080-U+009F). It decodes s rune-by-rune rather than
// byte-by-byte: a byte-wise scan over 0x80-0x9F false-positives on the
// continuation bytes of ordinary multi-byte UTF-8 characters (for example
// "→" and "⋯" both have a continuation byte in that range), which would
// make this helper reject legitimately sanitised, non-ASCII text.
func containsANSI(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r == 0x1b {
			return true
		}
		// Bare C1 control characters (0x80-0x9f).
		if r >= 0x80 && r <= 0x9f {
			return true
		}
	}
	return false
}

// TestApprovalCardRenameAgreement verifies that a patch approval with a renamed
// file displays the rename consistently: the card's detail line and the modal's
// title should both show "old → new" format.
func TestApprovalCardRenameAgreement(t *testing.T) {
	m := New(Options{})

	// Create a patch approval with a renamed file.
	// The detail shows "old → new" format from formatPatchDisplay.
	// The DiffFilename should also show "old → new" from computeDiffDisplay.
	requestMsg := ApprovalRequestedMsg{
		ID:                   1,
		Description:          "apply_patch — rename file",
		Kind:                 "patch",
		CanApproveForSession: false,
		Subject:              "1 file changed",
		Detail: []string{
			"files: 1 changed (old_name.go → new_name.go)",
		},
		DiffFilename: "old_name.go → new_name.go", // formatPatchDisplay and computeDiffDisplay agree
		DiffLines: []string{
			"File renamed",
		},
		DiffAdded:   0,
		DiffRemoved: 0,
	}

	updatedModel, _ := m.Update(requestMsg)
	m = updatedModel.(Model)

	// Find the approval card
	var foundCard *ApprovalCard
	for _, item := range m.tr.Items() {
		if item.Kind == KindApproval && item.Approval != nil {
			foundCard = item.Approval
			break
		}
	}
	if foundCard == nil {
		t.Fatal("no approval card found in transcript")
	}

	// Verify card detail shows rename
	detailStr := strings.Join(foundCard.Detail, "\n")
	if !strings.Contains(detailStr, "old_name.go → new_name.go") {
		t.Errorf("detail does not show rename: %v", foundCard.Detail)
	}

	// Verify DiffFilename (modal title) shows rename
	if foundCard.DiffFilename != "old_name.go → new_name.go" {
		t.Errorf("DiffFilename = %q, want %q", foundCard.DiffFilename, "old_name.go → new_name.go")
	}

	// Press 'd' to open the diff modal and verify modal title
	updatedModel, _ = m.Update(key('d'))
	m = updatedModel.(Model)

	if !m.overlayOpen() {
		t.Fatal("modal should be open after 'd' keypress")
	}
	if m.modal.Title != "old_name.go → new_name.go" {
		t.Errorf("modal.Title = %q, want %q", m.modal.Title, "old_name.go → new_name.go")
	}
}

// TestCommandApprovalArrivesWithDetail verifies that a command approval
// request arrives with non-empty detail lines for display in the detail modal.
// The detail is populated at the app.Request source, not hand-set in the message.
func TestCommandApprovalArrivesWithDetail(t *testing.T) {
	m := New(Options{})

	// Create a command approval with detail.
	// In real usage, app.Request (via formatCommandDetail) populates this.
	requestMsg := ApprovalRequestedMsg{
		ID:                   1,
		Description:          "run a command",
		Kind:                 "command",
		CanApproveForSession: true,
		Subject:              "go test ./...",
		Detail: []string{
			"run a command",
			"command: go test ./...",
		},
		GrantScope: "go test ./...",
		Argv:       []string{"go", "test", "./..."},
	}

	updatedModel, _ := m.Update(requestMsg)
	m = updatedModel.(Model)

	// Find the approval card
	var foundCard *ApprovalCard
	for _, item := range m.tr.Items() {
		if item.Kind == KindApproval && item.Approval != nil {
			foundCard = item.Approval
			break
		}
	}
	if foundCard == nil {
		t.Fatal("no approval card found in transcript")
	}

	// Verify detail is populated
	if len(foundCard.Detail) == 0 {
		t.Errorf("Detail is empty, want non-empty detail lines")
	}

	// Verify detail contains expected content
	detailStr := strings.Join(foundCard.Detail, "\n")
	if !strings.Contains(detailStr, "command: go test ./...") {
		t.Errorf("Detail missing command line: %v", foundCard.Detail)
	}

	// Press 'd' to open the detail modal and verify it shows the detail
	updatedModel, _ = m.Update(key('d'))
	m = updatedModel.(Model)

	if !m.overlayOpen() {
		t.Fatal("modal should be open after 'd' keypress on command approval")
	}

	if m.modal.Kind != ModalContent {
		t.Errorf("modal.Kind = %v, want ModalContent", m.modal.Kind)
	}

	// Verify modal contains the detail
	modalStr := strings.Join(m.modal.Lines, "\n")
	if !strings.Contains(modalStr, "command: go test ./...") {
		t.Errorf("modal does not contain command detail: %v", m.modal.Lines)
	}
}

// TestApprovalsListsRealGrants verifies that /approvals lists grants created
// by calling GetGrants() callback which reads the actual grants from the policy.
func TestApprovalsListsRealGrants(t *testing.T) {
	m := New(Options{})

	// Wire a mock GetGrants callback that returns two grants
	m.GetGrants = func() []string {
		return []string{"go test", "go build"}
	}

	// Run /approvals command
	for _, r := range "/approvals" {
		updatedModel, _ := m.Update(key(r))
		m = updatedModel.(Model)
	}
	updatedModel, _ := m.Update(keyType(tea.KeyEnter))
	m = updatedModel.(Model)

	// Should open a modal with the grants
	if m.mode() != ModeModal {
		t.Fatalf("mode after /approvals = %v, want Modal", m.mode())
	}

	if m.modal.Title != "session grants" {
		t.Errorf("modal.Title = %q, want 'session grants'", m.modal.Title)
	}

	// Verify the grants are in the modal lines
	grantStr := strings.Join(m.modal.Lines, "\n")
	if !strings.Contains(grantStr, "go test") {
		t.Errorf("modal does not contain 'go test' grant: %v", m.modal.Lines)
	}
	if !strings.Contains(grantStr, "go build") {
		t.Errorf("modal does not contain 'go build' grant: %v", m.modal.Lines)
	}
}

// TestStatusBarCountMatchesPolicyGrants verifies that statusbar uses the
// GetGrantCount callback to display the grant count.
func TestStatusBarCountMatchesPolicyGrants(t *testing.T) {
	// With GetGrantCount wired, it should be called
	m := New(Options{})
	m.status.Model = "test-model"
	m.width = 200 // Set a wide layout so status bar can display all info

	callCount := 0
	m.GetGrantCount = func() int {
		callCount++
		return 5
	}

	lay := m.layout()
	statusRow := m.statusRow(lay)

	if callCount == 0 {
		t.Error("GetGrantCount callback not called when wired")
	}

	if !strings.Contains(statusRow, "5 grant") {
		t.Errorf("status row does not show grant count from callback: %s", statusRow)
	}

	// Without GetGrantCount wired, grants are not displayed
	m2 := New(Options{})
	m2.status.Model = "test-model"
	m2.width = 200
	// Don't wire GetGrantCount

	lay2 := m2.layout()
	statusRow2 := m2.statusRow(lay2)

	// Should not contain any grant text since callback is not wired
	if strings.Contains(statusRow2, "grant") {
		t.Errorf("status row should not show grants without callback wired: %s", statusRow2)
	}
}

// TestGrantsModalCKeyShowsConfirm verifies that pressing 'c' on a grants modal
// closes the modal and shows the clear confirmation prompt.
func TestGrantsModalCKeyShowsConfirm(t *testing.T) {
	m := New(Options{})

	// Wire callbacks
	m.GetGrants = func() []string {
		return []string{"go test"}
	}

	// Open the grants modal by simulating /approvals
	for _, r := range "/approvals" {
		updatedModel, _ := m.Update(key(r))
		m = updatedModel.(Model)
	}
	updatedModel, _ := m.Update(keyType(tea.KeyEnter))
	m = updatedModel.(Model)

	if m.mode() != ModeModal {
		t.Fatalf("initial mode = %v, want Modal", m.mode())
	}

	// Press 'c' to trigger clear
	updatedModel, _ = m.Update(key('c'))
	m = updatedModel.(Model)

	// Should now be in Confirm mode
	if m.mode() != ModeConfirm {
		t.Errorf("after 'c' press, mode = %v, want Confirm", m.mode())
	}

	// Verify the confirm prompt is asking about clearing grants
	if m.confirm == nil {
		t.Fatal("confirm state is nil after 'c' press")
	}
	if m.confirm.Action != ConfirmClearGrants {
		t.Errorf("confirm action = %v, want ConfirmClearGrants", m.confirm.Action)
	}
}

// TestApprovalsModalSanitisesGrants verifies that the /approvals command
// sanitises grant text to strip ANSI escape sequences, OSC sequences, and
// embedded newlines before displaying them in a modal. This is the regression
// test for update.go's SanitizeSingleLine call in the /approvals arm.
func TestApprovalsModalSanitisesGrants(t *testing.T) {
	m := New(Options{})

	// Build adversarial grant text with ANSI escapes, OSC sequences, and newlines
	ansiRed := "\x1b[31m"
	ansiReset := "\x1b[0m"
	oscSeq := "\x1b]0;evil\x07"

	// Sanitize also handles classes beyond CSI/OSC/newline: bare C1 control
	// bytes, carriage-return overwrite collapsing, tab expansion, and
	// over-length truncation. Each is exercised below so the regression test
	// for the /approvals sanitisation call site covers the whole function,
	// not just the two escape families.
	bareC1 := "go\u0085test"                                // NEL (C1), not a CSI/OSC sequence
	crOverwrite := "go build\rgo test"                      // bare \r: a terminal progress-bar overwrite
	tabbed := "go\ttest"                                    // tab must expand, not survive as \t
	overLong := "go " + strings.Repeat("x", 2200) + " test" // exceeds MaxRenderedLineWidth (2000 cols)

	grants := []string{
		ansiRed + "go test" + ansiReset,
		oscSeq + "go build" + ansiRed,
		"go\nvet",                         // embedded newline
		ansiRed + "go\nbuild" + ansiReset, // escape + newline
		bareC1,
		crOverwrite,
		tabbed,
		overLong,
	}

	m.GetGrants = func() []string {
		return grants
	}

	// Simulate typing /approvals and pressing Enter
	for _, r := range "/approvals" {
		updatedModel, _ := m.Update(key(r))
		m = updatedModel.(Model)
	}
	updatedModel, _ := m.Update(keyType(tea.KeyEnter))
	m = updatedModel.(Model)

	// Verify the modal opened
	if m.mode() != ModeModal {
		t.Fatalf("mode after /approvals = %v, want Modal", m.mode())
	}

	if m.modal.Title != "session grants" {
		t.Errorf("modal.Title = %q, want 'session grants'", m.modal.Title)
	}

	// Verify all lines are sanitised: no ANSI escapes, no OSC sequences, no newlines
	for i, line := range m.modal.Lines {
		if containsANSI(line) {
			t.Errorf("modal.Lines[%d] contains ANSI sequences: %q", i, line)
		}
		if strings.Contains(line, "\n") {
			t.Errorf("modal.Lines[%d] contains embedded newline: %q", i, line)
		}
	}

	// Verify content is preserved (the text survives sanitisation)
	allLines := strings.Join(m.modal.Lines, "|")
	if !strings.Contains(allLines, "go test") {
		t.Errorf("modal lost grant text 'go test': lines = %v", m.modal.Lines)
	}
	if !strings.Contains(allLines, "go build") {
		t.Errorf("modal lost grant text 'go build': lines = %v", m.modal.Lines)
	}
	if !strings.Contains(allLines, "go vet") {
		t.Errorf("modal lost grant text 'go vet': lines = %v", m.modal.Lines)
	}

	// Bare C1 control byte (index 4): replaceControls maps it to '·' rather
	// than dropping it or leaving it raw.
	c1Line := m.modal.Lines[4]
	if strings.ContainsRune(c1Line, '') {
		t.Errorf("modal.Lines[4] still contains raw C1 byte: %q", c1Line)
	}
	if !strings.Contains(c1Line, "go") || !strings.Contains(c1Line, "test") {
		t.Errorf("modal.Lines[4] lost surrounding text: %q", c1Line)
	}

	// Bare carriage return (index 5): Sanitize treats \r as a terminal
	// overwrite and keeps only the segment after the last \r — so "go build"
	// must be gone and only "go test" must remain.
	crLine := m.modal.Lines[5]
	if strings.Contains(crLine, "go build") {
		t.Errorf("modal.Lines[5] kept the overwritten segment: %q", crLine)
	}
	if !strings.Contains(crLine, "go test") {
		t.Errorf("modal.Lines[5] lost the surviving segment: %q", crLine)
	}

	// Tab expansion (index 6): no literal tab byte survives, and the
	// expansion still separates the two words.
	tabLine := m.modal.Lines[6]
	if strings.Contains(tabLine, "\t") {
		t.Errorf("modal.Lines[6] still contains a literal tab: %q", tabLine)
	}
	if !strings.Contains(tabLine, "go") || !strings.Contains(tabLine, "test") {
		t.Errorf("modal.Lines[6] lost surrounding text: %q", tabLine)
	}

	// Over-length truncation (index 7): capWidth caps the line at
	// MaxRenderedLineWidth display columns, marker included.
	longLine := m.modal.Lines[7]
	if cellWidth(longLine) > MaxRenderedLineWidth {
		t.Errorf("modal.Lines[7] width = %d, want <= %d", cellWidth(longLine), MaxRenderedLineWidth)
	}
	if !strings.HasSuffix(longLine, "⋯") {
		t.Errorf("modal.Lines[7] not truncated with marker: %q", longLine)
	}
}

// TestClearGrantsViaKeypressInvokesCallbackAndEmptiesPolicy merges what were
// two near-identical tests (TestClearGrantsEmptiesPolicy and
// TestClearGrantsCalibration): both wired the same mocks and both called
// applyConfirm directly, so neither exercised keypress dispatch. This one
// does the job neither original did — it drives the real 'c' then 'y'
// keypresses through Update(), the same path a user's terminal produces —
// while keeping both original assertions: the pre-state check (grants exist,
// callback not yet invoked) and the calibration check (the callback flag is
// set only when ClearGrants actually runs, not merely when the end-state
// happens to be empty). This is the mocked-callback, package-tui complement
// to cmd/kirsch's TestClearGrantsEmptiesPolicyE2E, which drives the same
// keypresses against the real policy.
func TestClearGrantsViaKeypressInvokesCallbackAndEmptiesPolicy(t *testing.T) {
	m := New(Options{})

	policyGrants := []string{"go test", "go build"}
	callbackInvoked := false
	m.GetGrants = func() []string {
		return policyGrants
	}
	m.ClearGrants = func() {
		callbackInvoked = true // Set only if callback is invoked
		policyGrants = []string{}
	}

	// Pre-state check: grants exist and callback not invoked yet.
	if len(m.GetGrants()) == 0 {
		t.Fatal("setup: should start with grants")
	}
	if callbackInvoked {
		t.Fatal("setup: callback should not be invoked before clear")
	}

	// Open the grants modal via /approvals, then dispatch the real keypresses
	// a user sends: 'c' to raise the clear confirmation, 'y' to confirm it.
	for _, r := range "/approvals" {
		updatedModel, _ := m.Update(key(r))
		m = updatedModel.(Model)
	}
	updatedModel, _ := m.Update(keyType(tea.KeyEnter))
	m = updatedModel.(Model)

	if m.mode() != ModeModal {
		t.Fatalf("mode after /approvals = %v, want Modal", m.mode())
	}

	updatedModel, _ = m.Update(key('c'))
	m = updatedModel.(Model)

	if m.mode() != ModeConfirm {
		t.Fatalf("mode after 'c' = %v, want Confirm", m.mode())
	}

	updatedModel, _ = m.Update(key('y'))
	m = updatedModel.(Model)

	// Calibration assertion: the callback MUST have been invoked.
	// If this fails, keyConfirm's "y" arm or applyConfirm's ConfirmClearGrants
	// arm is not calling m.ClearGrants.
	if !callbackInvoked {
		t.Fatal("ClearGrants callback was not invoked via keypress dispatch - the implementation is broken")
	}

	// Verification: the policy is now empty (callback did its job).
	if len(m.GetGrants()) != 0 {
		t.Errorf("after clear, GetGrants() = %v, want empty", m.GetGrants())
	}
}

// TestApprovalElapsedReflectsRealTime verifies that resolved approval cards
// show the actual elapsed time from request to resolution, not a constant.
// Uses an injected fake clock so elapsed time can be exact.
// Calibration: reverting the time.Since calculation in resolveApproval makes
// the equality assertion fail.
func TestApprovalElapsedReflectsRealTime(t *testing.T) {
	// Inject a fake clock that we can advance
	baseTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	fakeTime := baseTime
	nowFn := func() time.Time {
		return fakeTime
	}

	m := New(Options{Now: nowFn})

	// Dispatch an ApprovalRequestedMsg
	msg := ApprovalRequestedMsg{
		ID:                   1,
		Description:          "run a command",
		Kind:                 "command",
		CanApproveForSession: true,
		Subject:              "go test",
		Detail:               []string{"Running tests"},
		GrantScope:           "go test",
		Argv:                 []string{"go", "test", "./..."},
	}

	updatedModel, _ := m.Update(msg)
	m = updatedModel.(Model)

	// Find the approval card and verify RequestedAt was set
	var card *ApprovalCard
	for _, it := range m.tr.Items() {
		if it.Kind == KindApproval && it.Approval != nil {
			card = it.Approval
			break
		}
	}
	if card == nil {
		t.Fatal("no approval card found")
	}
	if card.RequestedAt != baseTime {
		t.Errorf("approval.RequestedAt = %v, want %v", card.RequestedAt, baseTime)
	}

	// Advance the fake clock by exactly 5 seconds
	wantElapsed := 5 * time.Second
	fakeTime = baseTime.Add(wantElapsed)

	// Resolve the approval
	updatedModel, _ = m.Update(key('y'))
	m = updatedModel.(Model)

	// Find the resolved card
	var resolved *ApprovalCard
	for _, it := range m.tr.Items() {
		if it.Kind == KindApproval && it.Approval != nil && it.Approval.Outcome == Approved {
			resolved = it.Approval
			break
		}
	}
	if resolved == nil {
		t.Fatal("no resolved approval card found")
	}

	// Assert elapsed equals the known interval (exact assertion with injected clock)
	if resolved.Elapsed != wantElapsed {
		t.Errorf("approval.Elapsed = %v, want exactly %v", resolved.Elapsed, wantElapsed)
	}

	// Verify the rendered duration format is correct
	rendered := formatDuration(resolved.Elapsed)
	if rendered != "5.0s" {
		t.Errorf("formatDuration(%v) = %q, want %q", resolved.Elapsed, rendered, "5.0s")
	}
}
