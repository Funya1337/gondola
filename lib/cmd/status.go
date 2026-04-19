package cmd

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show deployed service status and health",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfig("deploy.yaml")
		if err != nil {
			return err
		}

		dc := cfg.Deploy
		if dc.Host == "" || dc.User == "" {
			return fmt.Errorf("deploy.host and deploy.user are required in deploy.yaml")
		}
		if dc.Service.Name == "" {
			return fmt.Errorf("deploy.service.name is required for status check")
		}

		fmt.Printf("Connecting to %s@%s...\n", dc.User, dc.Host)

		client, err := sshConnect(dc)
		if err != nil {
			return err
		}
		defer client.Close()

		// Gather info
		svcStatus := getServiceStatus(client, dc.Service.Name)
		svcDetails := getServiceDetails(client, dc.Service.Name)
		binInfo := getBinaryInfo(client, dc.RemotePath)
		healthResult := checkHealth(dc.HealthCheck)

		// Display TUI
		showStatusTUI(cfg.Project.Name, dc.Host, dc.Service.Name, svcStatus, svcDetails, binInfo, healthResult)
		return nil
	},
}

type healthStatus struct {
	value string
	color tcell.Color
}

func getServiceStatus(client *ssh.Client, serviceName string) string {
	out, err := runRemoteCommandOutput(client, fmt.Sprintf("systemctl is-active %s 2>/dev/null", serviceName))
	if err != nil {
		return strings.TrimSpace(out)
	}
	return strings.TrimSpace(out)
}

func getServiceDetails(client *ssh.Client, serviceName string) string {
	out, _ := runRemoteCommandOutput(client, fmt.Sprintf("systemctl status %s 2>&1 || true", serviceName))
	return out
}

func getBinaryInfo(client *ssh.Client, remotePath string) string {
	// Try --version flag
	out, err := runRemoteCommandOutput(client, fmt.Sprintf("%s --version 2>/dev/null", remotePath))
	if err == nil && strings.TrimSpace(out) != "" {
		return strings.TrimSpace(strings.Split(out, "\n")[0])
	}

	// Fall back to file info (modification time + size)
	out, err = runRemoteCommandOutput(client, fmt.Sprintf("stat --format='%%y  %%s bytes' %s 2>/dev/null", remotePath))
	if err == nil && strings.TrimSpace(out) != "" {
		return strings.TrimSpace(out)
	}

	// Check if file exists at all
	_, err = runRemoteCommandOutput(client, fmt.Sprintf("test -f %s", remotePath))
	if err != nil {
		return "not found"
	}
	return "unknown"
}

func checkHealth(hc HealthCheckConfig) healthStatus {
	if hc.URL == "" {
		return healthStatus{value: "not configured", color: tcell.ColorGray}
	}

	timeout := time.Duration(hc.Timeout) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	httpClient := &http.Client{Timeout: timeout}
	resp, err := httpClient.Get(hc.URL)
	if err != nil {
		return healthStatus{value: fmt.Sprintf("FAIL: %v", err), color: tcell.ColorRed}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return healthStatus{value: fmt.Sprintf("OK (%d)", resp.StatusCode), color: tcell.ColorGreen}
	}
	return healthStatus{value: fmt.Sprintf("UNHEALTHY (%d)", resp.StatusCode), color: tcell.ColorYellow}
}

func showStatusTUI(projectName, host, serviceName, svcStatus, svcDetails, binaryInfo string, health healthStatus) {
	app := tview.NewApplication()

	// Header
	header := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText(fmt.Sprintf("[dodgerblue]━━━ Gondola Status: %s ━━━", projectName))

	// Status table
	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(false, false)

	row := 0
	addRow := func(label, value string, color tcell.Color) {
		table.SetCell(row, 0, tview.NewTableCell("  "+label).
			SetTextColor(tcell.ColorWhite).
			SetExpansion(1).
			SetAttributes(tcell.AttrBold))
		table.SetCell(row, 1, tview.NewTableCell(value).
			SetTextColor(color).
			SetExpansion(2))
		row++
	}

	addRow("Host", host, tcell.ColorWhite)
	addRow("Service", serviceName, tcell.ColorWhite)

	// Color-code service status
	svcColor := tcell.ColorRed
	switch strings.TrimSpace(svcStatus) {
	case "active":
		svcColor = tcell.ColorGreen
		svcStatus = "active (running)"
	case "inactive":
		svcColor = tcell.ColorYellow
		svcStatus = "inactive (stopped)"
	case "failed":
		svcColor = tcell.ColorRed
		svcStatus = "failed"
	case "activating":
		svcColor = tcell.ColorYellow
		svcStatus = "activating (starting)"
	}

	addRow("Status", svcStatus, svcColor)
	addRow("Binary", binaryInfo, tcell.ColorWhite)
	addRow("Health", health.value, health.color)

	table.SetBorder(true).
		SetTitle(" Service Status ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.ColorDodgerBlue)

	// Details view
	details := tview.NewTextView().
		SetDynamicColors(false).
		SetText(svcDetails).
		SetScrollable(true)
	details.SetBorder(true).
		SetTitle(" Service Details (systemctl status) ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.ColorDodgerBlue)

	// Footer
	footer := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("Press Q or Esc to exit  |  Arrow keys to scroll details").
		SetTextColor(tcell.ColorGray)

	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(table, row+2, 0, false).
		AddItem(details, 0, 1, true).
		AddItem(footer, 1, 0, false)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Rune() == 'q' || event.Rune() == 'Q' {
			app.Stop()
			return nil
		}
		return event
	})

	if err := app.SetRoot(layout, true).Run(); err != nil {
		panic(err)
	}
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
