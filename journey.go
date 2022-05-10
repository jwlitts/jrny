package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"jlitts/journey/list"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type model struct {
	list      list.Model
	textInput textinput.Model
	journal   *os.File
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			//Add to list
			newIndex := len(m.list.Items())
			tstamp := time.Now().Format("2006/01/02 15:04")
			journalLine := m.textInput.Value()
			if len(journalLine) > 0 {

				m.list.InsertItem(newIndex, item{desc: tstamp, title: journalLine})
				m.list.Select(newIndex)
				m.textInput.SetValue("")

				n, err := m.journal.WriteString(fmt.Sprintf("%s-%s\n", tstamp, journalLine))
				if err != nil {

					log.Fatalf("%d: %s", n, err)
				}
				m.textInput, cmd = m.textInput.Update(msg)
			}
		default:
			m.textInput, cmd = m.textInput.Update(msg)

		}
	case tea.WindowSizeMsg:
		top, right, bottom, left := docStyle.GetMargin()
		log.Printf("%+v", msg)
		m.list.SetSize(msg.Width-left-right, msg.Height-top-bottom)
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return fmt.Sprintf("%s\n%s", docStyle.Render(m.list.View()), m.textInput.View())
}

func ParseFile(file *os.File) ([]list.Item, error) {
	items := []list.Item{}

	scanner := bufio.NewScanner(file)
	// optionally, resize scanner's capacity for lines over 64K, see next example
	for scanner.Scan() {
		line := scanner.Text()
		timestamp, text, ok := strings.Cut(line, "-")
		if ok {
			items = append(items, item{title: text, desc: timestamp})
		}
	}

	if err := scanner.Err(); err != nil {
		return items, err
	}
	return items, nil
}

func main() {
	logf, err := os.OpenFile("journey.log", os.O_APPEND|os.O_CREATE, os.ModePerm)
	if err == nil {
		defer logf.Close()
		log.SetOutput(logf)
	}
	log.Println("Starting")
	journalfile := "journal.jrnl"
	if len(os.Args) > 1 {
		journalfile = os.Args[1]
	}
	items := []list.Item{}
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
		items, err = ParseFile(file)
		if err != nil {
			log.Fatalln("Failed to parse file")
		}
	}
	defer file.Close()

	ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	m := model{list: list.New(items, list.NewDefaultDelegate(), 0, 0), textInput: ti, journal: file}
	m.list.Title = journalfile
	m.list.Select(len(items))

	p := tea.NewProgram(m)

	if err := p.Start(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
