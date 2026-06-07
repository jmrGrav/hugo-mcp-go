package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func emitProgress(ctx context.Context, req *mcp.CallToolRequest, message string, progress, total float64) {
	if req == nil || req.Session == nil || req.Params.GetProgressToken() == nil {
		return
	}
	_ = req.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
		Message:       message,
		ProgressToken: req.Params.GetProgressToken(),
		Progress:      progress,
		Total:         total,
	})
}
