package cmd

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the full pipeline: test → build → deploy",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPipeline()
	},
}

type pipelineStage struct {
	name string
	fn   func() error
}

func runPipeline() error {
	// Verify config exists before showing TUI
	if _, err := LoadConfig("deploy.yaml"); err != nil {
		return err
	}

	stages := []pipelineStage{
		{"Test", func() error { return testCmd.RunE(testCmd, nil) }},
		{"Build", func() error { return buildCmd.RunE(buildCmd, nil) }},
		{"Deploy", func() error { return deployCmd.RunE(deployCmd, nil) }},
	}

	app := tview.NewApplication()
	logView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetChangedFunc(func() { app.Draw() })
	logView.SetBorder(true).
		SetTitle(" Pipeline Log ").
		SetBorderColor(tcell.ColorDodgerBlue)

	table := tview.NewTable().SetBorders(false)
	table.SetBorder(true).
		SetTitle(" Pipeline: Test → Build → Deploy ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.ColorDodgerBlue)

	// Init table rows
	for i, s := range stages {
		table.SetCell(i, 0, tview.NewTableCell("  "+s.name).
			SetTextColor(tcell.ColorWhite).
			SetAttributes(tcell.AttrBold).
			SetExpansion(1))
		table.SetCell(i, 1, tview.NewTableCell("pending").
			SetTextColor(tcell.ColorGray).
			SetExpansion(1))
	}

	footer := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("Pipeline running... Press Esc to abort").
		SetTextColor(tcell.ColorGray)

	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(table, len(stages)+2, 0, false).
		AddItem(logView, 0, 1, true).
		AddItem(footer, 1, 0, false)

	var aborted bool
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			aborted = true
			app.Stop()
			return nil
		}
		return event
	})

	var pipelineErr error

	go func() {
		for i, stage := range stages {
			if aborted {
				break
			}

			// Mark running
			app.QueueUpdateDraw(func() {
				table.GetCell(i, 1).
					SetText("running...").
					SetTextColor(tcell.ColorYellow)
			})

			writeLog(logView, fmt.Sprintf("[dodgerblue]━━━ Stage: %s ━━━[-]\n", stage.name))
			start := time.Now()

			err := stage.fn()
			elapsed := time.Since(start).Round(time.Millisecond)

			if err != nil {
				app.QueueUpdateDraw(func() {
					table.GetCell(i, 1).
						SetText(fmt.Sprintf("FAILED (%s)", elapsed)).
						SetTextColor(tcell.ColorRed)
				})
				writeLog(logView, fmt.Sprintf("[red]✗ %s failed: %v[-]\n\n", stage.name, err))
				pipelineErr = fmt.Errorf("pipeline failed at %s: %w", stage.name, err)
				break
			}

			app.QueueUpdateDraw(func() {
				table.GetCell(i, 1).
					SetText(fmt.Sprintf("done (%s)", elapsed)).
					SetTextColor(tcell.ColorGreen)
			})
			writeLog(logView, fmt.Sprintf("[green]✓ %s completed (%s)[-]\n\n", stage.name, elapsed))
		}

		app.QueueUpdateDraw(func() {
			if pipelineErr != nil {
				footer.SetText("Pipeline FAILED. Press Esc or Q to exit.").SetTextColor(tcell.ColorRed)
			} else if aborted {
				footer.SetText("Pipeline aborted. Press Esc or Q to exit.").SetTextColor(tcell.ColorYellow)
			} else {
				footer.SetText("Pipeline completed successfully! Press Esc or Q to exit.").SetTextColor(tcell.ColorGreen)
			}
		})

		// Now allow Q to exit too
		app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyEscape || event.Rune() == 'q' || event.Rune() == 'Q' {
				app.Stop()
				return nil
			}
			return event
		})
	}()

	if err := app.SetRoot(layout, true).Run(); err != nil {
		return err
	}

	return pipelineErr
}

func writeLog(tv *tview.TextView, text string) {
	fmt.Fprint(tv, text)
}

func init() {
	rootCmd.AddCommand(runCmd)
}
