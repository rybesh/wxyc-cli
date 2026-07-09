package cmd

import (
	"strconv"

	"github.com/spf13/cobra"
)

var weekdays = []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

func newScheduleCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "schedule",
		Short: "Show the recurring show schedule",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			entries, raw, err := app.client.Schedule(cmd.Context())
			if err != nil {
				return err
			}
			rows := make([][]string, 0, len(entries))
			for _, e := range entries {
				day := strconv.Itoa(e.Day)
				if e.Day >= 0 && e.Day < len(weekdays) {
					day = weekdays[e.Day]
				}
				rows = append(rows, []string{day, e.StartTime, strconv.Itoa(e.ShowDuration) + "m"})
			}
			return app.render.EmitRaw(raw, []string{"DAY", "START", "DURATION"}, rows)
		},
	}
}
