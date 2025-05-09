package main

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

type status int

const (
	todo status = iota
	inProgress
	done
)

const divisor = 2

var conn pgx.Conn

/* STYLING */

var (
	columnStyle  = lipgloss.NewStyle().Padding(1, 2)
	focusedStyle = lipgloss.NewStyle().Padding(1, 2).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

var helpBar = helpStyle.Render(
	"←/→: Navigate  ↑/↓: Navigate  Enter: Move to right  n: Add new  Delete: Delete  q: Quit",
)

type Task struct {
	id          int
	status      status
	title       string
	description string
}

func NewTask(status status, title, description string) Task {
	return Task{
		status:      status,
		title:       title,
		description: description,
	}
}

func (t Task) FilterValue() string {
	return t.title
}

func (t Task) Title() string {
	return t.title
}

func (t Task) Description() string {
	return t.description
}

func (t *Task) Next() {
	if t.status == done {
		t.status = todo
	} else {
		t.status++
	}
}

func (t *Task) Prev() {
	if t.status == todo {
		t.status = done
	} else {
		t.status--
	}
}

var models []tea.Model

const (
	kan status = iota
	form
)

/* MAIN MODEL*/

type Model struct {
	focused  status
	lists    []list.Model
	err      error
	loaded   bool
	quitting bool
}

func New() *Model {
	return &Model{}
}

func (m *Model) Next() {
	if m.focused == done {
		m.focused = todo
	} else {
		m.focused++
	}
}

func (m *Model) Prev() {
	if m.focused == todo {
		m.focused = done
	} else {
		m.focused--
	}
}

type MovedTaskMsg struct{}
type DeletedTaskMsg struct{}

func (m *Model) MoveToNext() tea.Msg {
	selectedItem := m.lists[m.focused].SelectedItem()
	if selectedItem == nil {
		return nil
	}
	selectedTask := selectedItem.(Task)

	m.lists[selectedTask.status].RemoveItem(m.lists[m.focused].Index())

	selectedTask.Next()

	m.lists[selectedTask.status].InsertItem(len(m.lists[selectedTask.status].Items())-1, list.Item(selectedTask))

	_, err := conn.Exec(
		context.Background(),
		"update tasks set status=$1 where id=$2",
		selectedTask.status, selectedTask.id,
	)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error altering task in database: %v\n", err)
		return nil
	}

	return func() tea.Msg {
		return MovedTaskMsg{}
	}
}

func (m *Model) Delete() tea.Msg {
	selectedItem := m.lists[m.focused].SelectedItem()
	if selectedItem == nil {
		return nil
	}
	selectedTask := selectedItem.(Task)

	m.lists[selectedTask.status].RemoveItem(m.lists[m.focused].Index())

	_, err := conn.Exec(context.Background(), "delete from tasks where id=$1", selectedTask.id)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error altering task in database: %v\n", err)
		return nil
	}

	return func() tea.Msg {
		return DeletedTaskMsg{}
	}
}

func fetchTasks(s status) []Task {

	var tasks []Task
	rows, err := conn.Query(context.Background(), "select id, status, title, description from tasks where status=$1", s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Query failed: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	for rows.Next() {
		var task Task
		if err := rows.Scan(&task.id, &task.status, &task.title, &task.description); err != nil {
			fmt.Fprintf(os.Stderr, "Scan failed: %v\n", err)
			os.Exit(1)
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error during rows iteration: %v\n", err)
		os.Exit(1)
	}

	return tasks
}

func (m *Model) initLists(w, h int) {
	defaultList := list.New([]list.Item{}, list.NewDefaultDelegate(), w/divisor, h-(divisor*3))
	defaultList.SetShowHelp(false)
	m.lists = []list.Model{defaultList, defaultList, defaultList}

	m.lists[todo].Title = "To do"

	todoItems := fetchTasks(todo)

	todos := make([]list.Item, len(todoItems))
	for i, t := range todoItems {
		todos[i] = t
	}

	m.lists[todo].SetItems([]list.Item(todos))

	m.lists[inProgress].Title = "In Progress"

	inPItems := fetchTasks(inProgress)

	ps := make([]list.Item, len(inPItems))

	for i, t := range inPItems {
		ps[i] = t
	}

	m.lists[inProgress].SetItems(ps)

	m.lists[done].Title = "Done"

	doneItems := fetchTasks(done)

	dones := make([]list.Item, len(doneItems))

	for i, t := range doneItems {
		dones[i] = t
	}

	m.lists[done].SetItems(dones)
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.initLists(msg.Width, msg.Height)
		if !m.loaded {
			m.loaded = true
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "left", "h":
			m.Prev()
		case "right", "l":
			m.Next()
		case "enter":
			return m, m.MoveToNext
		case "delete":
			return m, m.Delete

		case "n":
			models[kan] = m
			models[form] = NewForm(m.focused)
			return models[form].Update(nil)
		}

	case Task:
		task := msg
		return m, m.lists[task.status].InsertItem(len(m.lists[task.status].Items()), task)

	case MovedTaskMsg:
		return m, nil
	}

	var cmd tea.Cmd
	m.lists[m.focused], cmd = m.lists[m.focused].Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		fmt.Println("quitting")
		return "See You Space Cowboy..."
	}

	if m.loaded {
		todoView := m.lists[todo].View()
		inProgressView := m.lists[inProgress].View()
		doneView := m.lists[done].View()

		var view string

		switch m.focused {
		case inProgress:
			view = lipgloss.JoinHorizontal(lipgloss.Left, columnStyle.Render(todoView), focusedStyle.Render(inProgressView), columnStyle.Render(doneView))

		case done:
			view = lipgloss.JoinHorizontal(lipgloss.Left, columnStyle.Render(todoView), columnStyle.Render(inProgressView), focusedStyle.Render(doneView))

		default:
			view = lipgloss.JoinHorizontal(lipgloss.Left, focusedStyle.Render(todoView), columnStyle.Render(inProgressView), columnStyle.Render(doneView))
		}

		return view + "\n\n" + helpBar
	} else {
		return "Loading..."
	}

}

type Form struct {
	focused     status
	title       textinput.Model
	description textarea.Model
}

func NewForm(focused status) *Form {
	form := &Form{}
	form.title = textinput.New()

	form.title.Focus()

	form.description = textarea.New()
	form.focused = focused
	return form
}

func (m Form) CreateTask() tea.Msg {
	task := NewTask(m.focused, m.title.Value(), m.description.Value())

	_, err := conn.Exec(
		context.Background(),
		"INSERT INTO tasks (status, title, description) VALUES ($1, $2, $3)",
		task.status, task.title, task.description,
	)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error inserting task into database: %v\n", err)
		return nil
	}

	return task
}

func (m Form) Init() tea.Cmd {
	return nil
}

func (m Form) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if m.title.Focused() {
				m.title.Blur()
				m.description.Focus()
				return m, textarea.Blink
			} else {
				models[form] = m
				return models[kan], m.CreateTask
			}

		}
	}

	if m.title.Focused() {
		m.title, cmd = m.title.Update(msg)
		return m, cmd
	} else {
		m.description, cmd = m.description.Update(msg)
		return m, cmd
	}
}

func (m Form) View() string {
	return lipgloss.JoinVertical(lipgloss.Left, m.title.View(), m.description.View())
}

func main() {
	_ = godotenv.Load()
	models = []tea.Model{New(), NewForm(todo)}
	conn = *startConn()
	m := models[kan]
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
	}

	defer conn.Close(context.Background())
}
