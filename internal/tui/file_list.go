package tui

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/lipgloss/v2"
)

type FileItem struct {
	Entry  os.DirEntry
	Status string
}

func (fi FileItem) Title() string       { return fi.Entry.Name() }
func (fi FileItem) Description() string { return "" }
func (fi FileItem) FilterValue() string { return fi.Entry.Name() }
func (fi FileItem) IsDir() bool         { return fi.Entry.IsDir() }

type FileDelegate struct {
	list.DefaultDelegate
	UseNerdFonts bool
}

var statusStyles = map[string]lipgloss.Style{
	"U": lipgloss.NewStyle().Foreground(lipgloss.Color("135")), // Unmerged
	"M": lipgloss.NewStyle().Foreground(lipgloss.Color("220")), // Modified
	"A": lipgloss.NewStyle().Foreground(lipgloss.Color("10")),  // Added
	"D": lipgloss.NewStyle().Foreground(lipgloss.Color("9")),   // Deleted
	"R": lipgloss.NewStyle().Foreground(lipgloss.Color("45")),  // Renamed
	"C": lipgloss.NewStyle().Foreground(lipgloss.Color("165")), // Copied
	"":  lipgloss.NewStyle().Foreground(lipgloss.Color("240")), // Default
}

func (d FileDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(FileItem)
	if !ok {
		return
	}

	name := it.Title()
	var icon string
	var baseStyle lipgloss.Style
	info, err := it.Entry.Info()
	isExecutable := err == nil && info.Mode().Perm()&0o111 != 0

	if it.IsDir() {
		// Color for directories
		baseStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	} else if isExecutable {
		// Color for executable files
		baseStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))
	} else {
		// Default color for normal files
		baseStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("253"))
	}

	if d.UseNerdFonts {
		icon = GetNerdFontIcon(name, it.IsDir())
	} else {
		if it.IsDir() {
			icon = "📁"
		} else {
			icon = "📄"
		}
	}

	statusText := " "
	statusStyle := statusStyles[""]
	if it.Status != "" {
		statusText = fmt.Sprintf("%s", it.Status)
		if style, found := statusStyles[it.Status]; found {
			statusStyle = style
		}
	}

	if index == m.Index() {
		selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
		fmt.Fprintf(
			w,
			"❯ %s %s %s",
			statusStyle.Render(statusText),
			selectedStyle.Render(icon),
			selectedStyle.
				Render(name),
		)
	} else {
		fmt.Fprintf(w, "  %s %s %s", statusStyle.Render(statusText), baseStyle.Render(icon),
			baseStyle.Render(name))
	}

	// fmt.Fprint(w, line)
}

func (d FileDelegate) Height() int  { return 1 }
func (d FileDelegate) Spacing() int { return 0 }

func NewFileList(pwd string, useNerdFont bool, gitData GitStatusData) (list.Model, error) {
	items, err := CreateFileItemsList(pwd, gitData)
	if err != nil {
		return list.Model{}, fmt.Errorf("Error changing list items %s", err)
	}

	fileList := list.New(items, FileDelegate{
		UseNerdFonts: useNerdFont,
	}, 0, 0)
	fileList.KeyMap.AcceptWhileFiltering = key.NewBinding(
		key.WithKeys("enter", "/", "ctrl+k", "ctrl+j"),
	)
	fileList.KeyMap.CancelWhileFiltering = key.NewBinding(key.WithKeys("/"))

	fileList.Title = fmt.Sprintf("Select a file or directory in %s", TruncatePath(pwd, 2))
	fileList.SetShowHelp(false)
	fileList.SetFilteringEnabled(true)
	fileList.SetShowPagination(true)
	fileList.SetStatusBarItemName("item", "items")
	fileList.StatusMessageLifetime = 5 * time.Second

	return fileList, nil
}
