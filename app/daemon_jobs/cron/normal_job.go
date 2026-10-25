package cron

import (
	"context"
	"log/slog"
)

func (j *Jobs) oneCronJob(ctx context.Context) {
	slog.InfoContext(ctx, "this is job one")
	// you own job
}
