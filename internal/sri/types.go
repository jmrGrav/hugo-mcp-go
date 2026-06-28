package sri

import (
	"context"

	"github.com/jmrGrav/hugo-mcp-go/internal/hooks"
	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/mutations"
)

type Request struct {
	AutoFix *bool
	DryRun  *bool
}

type Result struct {
	Plugin            string      `json:"plugin"`
	Success           bool        `json:"success"`
	ExitCode          int         `json:"exit_code"`
	AutoFixRequested  bool        `json:"auto_fix_requested"`
	DryRun            bool        `json:"dry_run"`
	Report            Report      `json:"report"`
	Downstream        map[string]any `json:"downstream,omitempty"`
}

type Report struct {
	Exit            int              `json:"exit"`
	Summary         string           `json:"summary"`
	Diagnostic      DiagnosticReport `json:"diagnostic"`
	AutoFix         AutoFixReport    `json:"auto_fix"`
	Incident        IncidentReport   `json:"incident"`
	HeartbeatPinged bool             `json:"heartbeat_pinged"`
	DryRun          bool             `json:"dry_run"`
}

type DiagnosticReport struct {
	HashMismatch   []string `json:"hash_mismatch"`
	MajorOutdated  []string `json:"major_outdated"`
	MinorOutdated   string   `json:"minor_outdated"`
	Other          []string `json:"other"`
}

type AutoFixReport struct {
	Ran     bool     `json:"ran"`
	Applied []string `json:"applied"`
	Failed  []string `json:"failed"`
	Skipped bool     `json:"skipped"`
	CFPurged bool    `json:"cf_purged"`
}

type IncidentReport struct {
	Created  *string  `json:"created"`
	Resolved []string `json:"resolved"`
}

type Builder interface {
	Build(context.Context, mutations.BuildRequest) (mutations.BuildResult, error)
}

type hookSummaryProvider interface {
	Process(context.Context, hooks.HookEvent) hooks.HookSummary
}

