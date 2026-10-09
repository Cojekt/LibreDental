package main

import (
	"context"

	"github.com/LibreDental/libredental/internal/services"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// reminderJobLifecycle runs the automatic reminder job for as long as the app runs. It is
// registered as its own Wails service with no exported methods besides the lifecycle hooks,
// which Wails never binds, so the frontend can't start or stop the job.
type reminderJobLifecycle struct {
	reminders *services.ReminderService
}

func newReminderJobLifecycle(reminders *services.ReminderService) *reminderJobLifecycle {
	return &reminderJobLifecycle{reminders: reminders}
}

// ServiceStartup starts the job. Wails cancels ctx just before shutdown.
func (l *reminderJobLifecycle) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	services.StartReminderJob(l.reminders, ctx)
	return nil
}

// ServiceShutdown waits for a pass in progress to finish. Wails calls it before App.Run
// returns, so before the database is closed.
func (l *reminderJobLifecycle) ServiceShutdown() error {
	services.StopReminderJob(l.reminders)
	return nil
}
