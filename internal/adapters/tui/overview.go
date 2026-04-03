package tui

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const defaultOverviewWidth = 110

func (ui *TerminalUI) RenderOverview(ctx context.Context, overview shared.Overview, interactive bool) error {
	if interactive && ui.tty && !ui.ci {
		if err := runOverviewProgram(ctx, ui.in, ui.out, overview); err != nil {
			return err
		}
		return nil
	}

	_, err := fmt.Fprintln(ui.out, renderOverview(overview, defaultOverviewWidth, false))
	return err
}

type overviewModel struct {
	overview shared.Overview
	width    int
	height   int
}

func newOverviewModel(overview shared.Overview) overviewModel {
	return overviewModel{overview: overview, width: defaultOverviewWidth}
}

func (m overviewModel) Init() tea.Cmd { return nil }

func (m overviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m overviewModel) View() string {
	width := m.width
	if width <= 0 {
		width = defaultOverviewWidth
	}
	return renderOverview(m.overview, width, true)
}

func runOverviewProgram(_ context.Context, in io.Reader, out io.Writer, overview shared.Overview) error {
	p := tea.NewProgram(
		newOverviewModel(overview),
		tea.WithInput(in),
		tea.WithOutput(out),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run overview tui: %w", err)
	}
	return nil
}

func renderOverview(overview shared.Overview, width int, interactive bool) string {
	if width < 80 {
		width = 80
	}

	palette := defaultPalette
	contentWidth := width - 8
	if contentWidth < 72 {
		contentWidth = 72
	}

	header := palette.header.Width(contentWidth).Render(strings.Join([]string{
		palette.title.Render(overview.Title),
		palette.subtitle.Render(overview.Subtitle),
	}, "\n"))

	status := palette.panel.Width(panelWidth(contentWidth, 2)).Render(strings.Join([]string{
		palette.sectionTitle.Render("Release status"),
		fmt.Sprintf("Current version: %s", valueOrFallback(overview.CurrentVersion, "dev")),
		fmt.Sprintf("Latest release: %s", latestVersionLabel(overview)),
		fmt.Sprintf("Upgrade guidance: %s", upgradeGuidanceLabel(overview)),
	}, "\n"))

	about := palette.panel.Width(panelWidth(contentWidth, 2)).Render(strings.Join([]string{
		palette.sectionTitle.Render("About"),
		fmt.Sprintf("Developer: %s", valueOrFallback(overview.Developer, "Unknown")),
		fmt.Sprintf("GitHub: %s", valueOrFallback(overview.GitHubProfile, "Unavailable")),
	}, "\n"))

	var topRow string
	if contentWidth >= 96 {
		topRow = lipgloss.JoinHorizontal(lipgloss.Top, status, about)
	} else {
		topRow = lipgloss.JoinVertical(lipgloss.Left, status, about)
	}

	commandLines := make([]string, 0, len(overview.Commands)+1)
	commandLines = append(commandLines, palette.sectionTitle.Render("Commands"))
	for _, command := range overview.Commands {
		line := fmt.Sprintf("• %-18s %s", command.Name, command.Summary)
		if command.Example != "" {
			line += fmt.Sprintf(" (e.g. %s)", command.Example)
		}
		if command.Interactive {
			line += " [interactive]"
		}
		commandLines = append(commandLines, line)
	}
	commands := palette.panel.Width(contentWidth).Render(strings.Join(commandLines, "\n"))

	quickStartLines := make([]string, 0, len(overview.QuickStart)+1)
	quickStartLines = append(quickStartLines, palette.sectionTitle.Render("Quick start"))
	for _, step := range overview.QuickStart {
		quickStartLines = append(quickStartLines, "• "+step)
	}
	quickStart := palette.panel.Width(panelWidth(contentWidth, 2)).Render(strings.Join(quickStartLines, "\n"))

	docLines := make([]string, 0, len(overview.Documentation)+1)
	docLines = append(docLines, palette.sectionTitle.Render("Documentation"))
	for _, path := range overview.Documentation {
		docLines = append(docLines, "• "+path)
	}
	docs := palette.panel.Width(panelWidth(contentWidth, 2)).Render(strings.Join(docLines, "\n"))

	var bottomRow string
	if contentWidth >= 96 {
		bottomRow = lipgloss.JoinHorizontal(lipgloss.Top, quickStart, docs)
	} else {
		bottomRow = lipgloss.JoinVertical(lipgloss.Left, quickStart, docs)
	}

	helpText := "Run `celador about` for a plain-text overview."
	if interactive {
		helpText = "Press q, esc, or ctrl+c to exit. Use `celador about` for plain-text output in CI or scripts."
	}
	help := palette.help.Width(contentWidth).Render(helpText)

	return palette.page.Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, header, topRow, commands, bottomRow, help))
}

type overviewPalette struct {
	page         lipgloss.Style
	header       lipgloss.Style
	title        lipgloss.Style
	subtitle     lipgloss.Style
	panel        lipgloss.Style
	sectionTitle lipgloss.Style
	help         lipgloss.Style
}

func panelWidth(total int, columns int) int {
	width := (total - (columns-1)*2) / columns
	if width < 34 {
		return total
	}
	return width
}

func latestVersionLabel(overview shared.Overview) string {
	if overview.LatestVersion == "" {
		return "Unavailable (release check failed or was skipped)"
	}
	if overview.UpdateAvailable {
		return overview.LatestVersion + " (update available)"
	}
	return overview.LatestVersion + " (up to date)"
}

func upgradeGuidanceLabel(overview shared.Overview) string {
	if !overview.UpdateAvailable {
		return "No action required"
	}
	if overview.UpgradeCommand != "" {
		return overview.UpgradeCommand
	}
	return "See GitHub releases for upgrade instructions"
}

func valueOrFallback(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

var defaultPalette = overviewPalette{
	page: lipgloss.NewStyle().Padding(1, 2),
	header: lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2).
		MarginBottom(1),
	title:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")),
	subtitle: lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
	panel: lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("241")).
		Padding(1, 2).
		MarginRight(2).
		MarginBottom(1),
	sectionTitle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")),
	help:         lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Padding(0, 1),
}
