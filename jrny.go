package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
)

const useHighPerformanceRenderer = false

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.Copy().BorderStyle(b)
	}()
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type model struct {
	content   string
	textInput textinput.Model
	journal   *os.File
	ready     bool
	viewport  viewport.Model
	filename  string
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			//Add to list
			tstamp := time.Now().Format("2006/01/02 15:04")
			journalLine := m.textInput.Value()
			if len(journalLine) > 0 {

				m.textInput.SetValue("")
				stampedline := fmt.Sprintf("%s-%s\n", tstamp, journalLine)
				n, err := m.journal.WriteString(stampedline)
				if err != nil {

					log.Fatalf("%d: %s", n, err)
				}
				m.textInput, cmd = m.textInput.Update(msg)
				m.content = m.content + stampedline
				m.viewport.SetContent(m.content)
				m.viewport.GotoBottom()

			}
		default:
			m.textInput, cmd = m.textInput.Update(msg)

		}
	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.HighPerformanceRendering = useHighPerformanceRenderer
			m.viewport.SetContent(m.content)
			m.ready = true
			// log.Printf("%+v", m.viewport.KeyMap)
			m.viewport.KeyMap.HalfPageDown.SetEnabled(false)
			m.viewport.KeyMap.HalfPageUp.SetEnabled(false)
			m.viewport.KeyMap.PageDown.SetEnabled(false)
			m.viewport.KeyMap.PageUp.SetEnabled(false)
			m.viewport.KeyMap.Down.SetKeys("down")
			m.viewport.KeyMap.Up.SetKeys("up")
			m.viewport.GotoBottom()
			// This is only necessary for high performance rendering, which in
			// most cases you won't need.
			//
			// Render the viewport one line below the header.
			m.viewport.YPosition = headerHeight + 1
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

		if useHighPerformanceRenderer {
			// Render (or re-render) the whole viewport. Necessary both to
			// initialize the viewport and when the window is resized.
			//
			// This is needed for high-performance rendering only.
			cmds = append(cmds, viewport.Sync(m.viewport))
		}
	}
	// Handle keyboard and mouse events in the viewport
	cmds = append(cmds, cmd)
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}
	return fmt.Sprintf("%s\n%s\n%s", m.headerView(), wordwrap.String(m.viewport.View(), m.viewport.Width-4), m.footerView())
}

func (m model) headerView() string {
	title := titleStyle.Render(m.filename)
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
	// return ""
}

func (m model) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return fmt.Sprintf("%s\n%s", lipgloss.JoinHorizontal(lipgloss.Center, line, info), m.textInput.View())
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	printMode := flag.Bool("l", false, "print journal then exit")
	appendMode := flag.String("a", "", "append to journal")
	args := flag.Args()
	flag.Parse()
	logf, err := os.OpenFile("journey.log", os.O_APPEND|os.O_CREATE, os.ModePerm)
	if err == nil {
		defer logf.Close()
		log.SetOutput(logf)
	}
	log.Println("Starting")
	journalfile := "journal.jrnl"
	if len(args) > 0 {
		journalfile = args[0]
	}
	content := ""
	file, err := os.OpenFile(journalfile, os.O_RDWR|os.O_APPEND, os.ModePerm)
	if err != nil {
		filename := path.Base(journalfile)
		directory := path.Dir(journalfile)
		os.MkdirAll(directory, os.ModePerm)
		file, err = os.Create(filename)
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		contentRaw, err := ioutil.ReadAll(file)
		if err != nil {
			log.Fatalln("Failed to parse file")
		}
		content = string(contentRaw)
	}
	defer file.Close()
	if *printMode {
		fmt.Printf("%s\n", content)
		return
	}
	if *appendMode != "" {
		tstamp := time.Now().Format("2006/01/02 15:04")
		stampedline := fmt.Sprintf("%s-%s\n", tstamp, *appendMode)
		file.WriteString(stampedline)
		return
	}
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	m := model{content: string(content), textInput: ti, journal: file, filename: journalfile}

	p := tea.NewProgram(m)

	if err := p.Start(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
