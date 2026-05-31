package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/mattlab76/ccsm-go/internal/claude"
)

// HandleSessionEnd is the SessionEnd hook entry point (invoked as
// `ccsm hook`). It reads Claude Code's hook payload as JSON from r and
// writes the spool file that ccsm picks up after the session ends — the
// same format the old bash hook produced, but without jq/python3.
//
// Everything is best-effort: any malformed input or write error is
// swallowed and nil is returned, so the hook can never block or fail
// Claude Code's shutdown. A missing session_id means there's nothing to
// record, so it's a silent no-op.
func HandleSessionEnd(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil || len(data) == 0 {
		return nil
	}

	var in struct {
		SessionID      string `json:"session_id"`
		CWD            string `json:"cwd"`
		TranscriptPath string `json:"transcript_path"`
	}
	if err := json.Unmarshal(data, &in); err != nil || in.SessionID == "" {
		return nil
	}

	subject, inTok, outTok := claude.ParseTranscript(in.TranscriptPath)

	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return nil
	}
	out, err := json.Marshal(map[string]string{
		"session_id": in.SessionID,
		"cwd":        in.CWD,
		"betreff":    subject,
		"transcript": in.TranscriptPath,
		"tokens":     fmt.Sprintf("%d/%d", inTok, outTok),
	})
	if err != nil {
		return nil
	}
	path := filepath.Join(tmpDir, fmt.Sprintf("session-%s.json", in.SessionID))
	_ = os.WriteFile(path, out, 0644)

	// Opportunistic housekeeping while we're here.
	CleanupOldTempFiles()
	return nil
}
