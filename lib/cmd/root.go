package cmd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
)

var noWelcome bool

var rootCmd = &cobra.Command{
	Use:   "gondola",
	Short: "Deploy Go applications to Linux servers via SSH",
	RunE: func(cmd *cobra.Command, args []string) error {
		return mainLoop()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().BoolVar(&noWelcome, "no-welcome", false, "skip the welcome guide on startup")
}

const welcomeText = `[dodgerblue::b]Gondola[-] [white::b]— Deploy Go apps to Linux servers[white]

Gondola помогает собирать, тестировать и деплоить Go-приложения
на Linux-серверы через SSH — прямо из терминала.

[yellow::b]Как начать:[white]
  1. Запустите [dodgerblue]Init[-] чтобы создать конфиг [dodgerblue]deploy.yaml[-]
  2. Укажите хост сервера, SSH-пользователя и путь к ключу
  3. Запустите [dodgerblue]Run Pipeline[-] для полного цикла: тесты, сборка, деплой

[yellow::b]Доступные действия:[white]
  [dodgerblue]Run Pipeline[-]   Полный цикл: тесты -> сборка -> деплой
  [dodgerblue]Init[-]           Создать deploy.yaml через интерактивный мастер
  [dodgerblue]Test[-]           Запустить тесты
  [dodgerblue]Build[-]          Кросс-компиляция Go-бинарника
  [dodgerblue]Deploy[-]         Загрузить бинарник и настроить systemd-сервис
  [dodgerblue]Status[-]         Проверить состояние и здоровье сервиса
  [dodgerblue]Stop / Restart[-] Управление systemd-сервисом на сервере
  [dodgerblue]Rollback[-]       Откатиться к предыдущей версии

[gray]Совет: команды можно запускать напрямую, например "gondola deploy"
Отключить это окно: --no-welcome[-]`

func showWelcome() {
	app := tview.NewApplication()

	body := tview.NewTextView().
		SetDynamicColors(true).
		SetText(welcomeText).
		SetScrollable(true)
	body.SetBorder(true).
		SetTitle(" Gondola — Краткое руководство ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorDodgerBlue)

	footer := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[dodgerblue]Нажмите Enter или Esc чтобы перейти в главное меню")

	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(body, 0, 1, true).
		AddItem(footer, 1, 0, false)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter || event.Key() == tcell.KeyEscape || event.Rune() == ' ' {
			app.Stop()
			return nil
		}
		return event
	})

	_ = app.SetRoot(layout, true).EnableMouse(true).Run()
}

type menuItem struct {
	label  string
	desc   string
	run    func() error
	isTUI  bool // if true, no "press enter" pause needed after
}

func mainLoop() error {
	if !noWelcome {
		showWelcome()
	}

	for {
		action, quit := showMainMenu()
		if quit {
			return nil
		}
		if action == nil {
			return nil
		}

		err := action.run()

		// For non-TUI commands, pause so user can read output before menu redraws
		if !action.isTUI {
			if err != nil {
				fmt.Printf("\n[error] %v\n", err)
			}
			fmt.Print("\nPress Enter to return to menu...")
			bufio.NewReader(os.Stdin).ReadBytes('\n')
		}
	}
}

func showMainMenu() (*menuItem, bool) {
	items := []menuItem{
		{"Run Pipeline", "Test -> Build -> Deploy (full pipeline)", func() error { return runPipeline() }, true},
		{"Init", "Generate deploy.yaml via interactive TUI", func() error { return initCmd.RunE(initCmd, nil) }, true},
		{"Test", "Run tests defined in deploy.yaml", func() error { return testCmd.RunE(testCmd, nil) }, false},
		{"Build", "Build the Go application", func() error { return buildCmd.RunE(buildCmd, nil) }, false},
		{"Deploy", "Deploy binary to remote server via SSH", func() error { return deployCmd.RunE(deployCmd, nil) }, false},
		{"Status", "Show deployed service status and health", func() error { return statusCmd.RunE(statusCmd, nil) }, true},
		{"Stop", "Stop the remote service", func() error { return stopCmd.RunE(stopCmd, nil) }, false},
		{"Restart", "Restart the remote service", func() error { return restartCmd.RunE(restartCmd, nil) }, false},
		{"Rollback", "Rollback to the previous version", func() error { return rollbackCmd.RunE(rollbackCmd, nil) }, false},
	}

	var selected int = -1

	app := tview.NewApplication()

	// Title
	title := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[dodgerblue::b]Gondola[-] — Deploy Go apps to Linux servers")

	// Build the list
	list := tview.NewList()
	shortcuts := []rune{'r', 'i', 't', 'b', 'd', 's', 'x', 'e', 'k'}
	for i, item := range items {
		idx := i
		list.AddItem(item.label, item.desc, shortcuts[i], func() {
			selected = idx
			app.Stop()
		})
	}
	list.AddItem("Quit", "Exit gondola", 'q', func() {
		app.Stop()
	})

	list.SetBorder(true).
		SetTitle(" Select Action ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.ColorDodgerBlue)
	list.SetSelectedBackgroundColor(tcell.ColorDodgerBlue)
	list.SetMainTextStyle(tcell.StyleDefault.Bold(true))

	// Config status indicator
	configStatus := "[red]deploy.yaml not found — run Init first"
	if _, err := os.Stat("deploy.yaml"); err == nil {
		cfg, err := LoadConfig("deploy.yaml")
		if err == nil {
			configStatus = fmt.Sprintf("[green]deploy.yaml loaded[white] — project: [dodgerblue]%s[white] | target: [dodgerblue]%s@%s",
				cfg.Project.Name, cfg.Deploy.User, cfg.Deploy.Host)
		}
	}

	statusBar := tview.NewTextView().
		SetDynamicColors(true).
		SetText("  " + configStatus)

	footer := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("Up/Down navigate  |  Enter select  |  Shortcut key to jump  |  Q quit").
		SetTextColor(tcell.ColorGray)

	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(title, 1, 0, false).
		AddItem(statusBar, 1, 0, false).
		AddItem(list, 0, 1, true).
		AddItem(footer, 1, 0, false)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			app.Stop()
			return nil
		}
		return event
	})

	if err := app.SetRoot(layout, true).EnableMouse(true).Run(); err != nil {
		return nil, true
	}

	if selected < 0 {
		return nil, true
	}

	return &items[selected], false
}
