package ui

import (
	"strings"

	"github.com/xvertile/sshc/internal/config"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// K8s form input indices
const (
	k8sNameInput = iota
	k8sNamespaceInput
	k8sPodInput
	k8sContainerInput
	k8sContextInput
	k8sKubeconfigInput
	k8sShellInput
	k8sTagsInput
)

// Messages for k8s form communication
type k8sAddFormSubmitMsg struct {
	hostName string
	err      error
}

type k8sAddFormCancelMsg struct{}

type k8sEditFormSubmitMsg struct {
	hostName string
	err      error
}

type k8sEditFormCancelMsg struct{}

// k8sAddFormModel represents the form for adding a new k8s host
type k8sAddFormModel struct {
	inputs  []textinput.Model
	focused int
	err     string
	styles  Styles
	success bool
	width   int
	height  int
}

// NewK8sAddForm creates a new k8s add form
func NewK8sAddForm(styles Styles, width, height int) *k8sAddFormModel {
	inputs := make([]textinput.Model, 8)

	// Name input
	inputs[k8sNameInput] = textinput.New()
	inputs[k8sNameInput].Placeholder = "display-name"
	inputs[k8sNameInput].Focus()
	inputs[k8sNameInput].CharLimit = 50
	inputs[k8sNameInput].Width = 30

	// Namespace input
	inputs[k8sNamespaceInput] = textinput.New()
	inputs[k8sNamespaceInput].Placeholder = "default"
	inputs[k8sNamespaceInput].CharLimit = 63
	inputs[k8sNamespaceInput].Width = 30

	// Pod input
	inputs[k8sPodInput] = textinput.New()
	inputs[k8sPodInput].Placeholder = "pod-name"
	inputs[k8sPodInput].CharLimit = 253
	inputs[k8sPodInput].Width = 40

	// Container input (optional)
	inputs[k8sContainerInput] = textinput.New()
	inputs[k8sContainerInput].Placeholder = "(optional) container-name"
	inputs[k8sContainerInput].CharLimit = 63
	inputs[k8sContainerInput].Width = 30

	// Context input (optional)
	inputs[k8sContextInput] = textinput.New()
	inputs[k8sContextInput].Placeholder = "(optional) kubectl context"
	inputs[k8sContextInput].CharLimit = 100
	inputs[k8sContextInput].Width = 40

	// Kubeconfig input (optional)
	inputs[k8sKubeconfigInput] = textinput.New()
	inputs[k8sKubeconfigInput].Placeholder = "(optional) ~/.kube/config"
	inputs[k8sKubeconfigInput].CharLimit = 200
	inputs[k8sKubeconfigInput].Width = 50

	// Shell input
	inputs[k8sShellInput] = textinput.New()
	inputs[k8sShellInput].Placeholder = "/bin/bash"
	inputs[k8sShellInput].CharLimit = 100
	inputs[k8sShellInput].Width = 30

	// Tags input (optional)
	inputs[k8sTagsInput] = textinput.New()
	inputs[k8sTagsInput].Placeholder = "production, web, database"
	inputs[k8sTagsInput].CharLimit = 200
	inputs[k8sTagsInput].Width = 50

	return &k8sAddFormModel{
		inputs:  inputs,
		focused: k8sNameInput,
		styles:  styles,
		width:   width,
		height:  height,
	}
}

func (m *k8sAddFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *k8sAddFormModel) Update(msg tea.Msg) (*k8sAddFormModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.styles = NewStyles(m.width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, func() tea.Msg { return k8sAddFormCancelMsg{} }

		case "ctrl+s":
			return m, m.submitForm()

		case "tab", "shift+tab", "enter", "up", "down":
			return m, m.handleNavigation(msg.String())
		}

	case k8sAddFormSubmitMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.success = true
			m.err = ""
		}
		return m, nil
	}

	// Update inputs
	cmd := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmd[i] = m.inputs[i].Update(msg)
	}
	cmds = append(cmds, cmd...)

	return m, tea.Batch(cmds...)
}

func (m *k8sAddFormModel) handleNavigation(key string) tea.Cmd {
	// Handle form submission on last field
	if key == "enter" && m.focused == k8sTagsInput {
		return m.submitForm()
	}

	// Navigate between inputs
	if key == "up" || key == "shift+tab" {
		m.focused--
	} else {
		m.focused++
	}

	// Wrap around
	if m.focused >= len(m.inputs) {
		m.focused = 0
	} else if m.focused < 0 {
		m.focused = len(m.inputs) - 1
	}

	return m.updateFocus()
}

func (m *k8sAddFormModel) updateFocus() tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.inputs {
		if i == m.focused {
			cmds = append(cmds, m.inputs[i].Focus())
		} else {
			m.inputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (m *k8sAddFormModel) submitForm() tea.Cmd {
	return func() tea.Msg {
		// Get values
		name := strings.TrimSpace(m.inputs[k8sNameInput].Value())
		namespace := strings.TrimSpace(m.inputs[k8sNamespaceInput].Value())
		pod := strings.TrimSpace(m.inputs[k8sPodInput].Value())
		container := strings.TrimSpace(m.inputs[k8sContainerInput].Value())
		context := strings.TrimSpace(m.inputs[k8sContextInput].Value())
		kubeconfig := strings.TrimSpace(m.inputs[k8sKubeconfigInput].Value())
		shell := strings.TrimSpace(m.inputs[k8sShellInput].Value())
		tagsStr := strings.TrimSpace(m.inputs[k8sTagsInput].Value())

		// Validate required fields
		if name == "" {
			return k8sAddFormSubmitMsg{err: &validationError{field: "Name", message: "Name is required"}}
		}
		if namespace == "" {
			namespace = "default"
		}
		if pod == "" {
			return k8sAddFormSubmitMsg{err: &validationError{field: "Pod", message: "Pod name is required"}}
		}

		// Apply defaults
		if shell == "" {
			shell = "/bin/bash"
		}

		// Parse tags
		var tags []string
		if tagsStr != "" {
			for _, tag := range strings.Split(tagsStr, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					tags = append(tags, tag)
				}
			}
		}

		// Create k8s host
		host := config.K8sHost{
			Name:       name,
			Namespace:  namespace,
			Pod:        pod,
			Container:  container,
			Context:    context,
			Kubeconfig: kubeconfig,
			Shell:      shell,
			Tags:       tags,
		}

		// Add to config
		err := config.AddK8sHost(host)
		return k8sAddFormSubmitMsg{hostName: name, err: err}
	}
}

func (m *k8sAddFormModel) View() string {
	if m.success {
		return ""
	}

	var b strings.Builder

	b.WriteString(m.styles.FormTitle.Render("Add Kubernetes Host"))
	b.WriteString("\n\n")

	fields := []struct {
		index int
		label string
	}{
		{k8sNameInput, "Display Name *"},
		{k8sNamespaceInput, "Namespace *"},
		{k8sPodInput, "Pod Name *"},
		{k8sContainerInput, "Container"},
		{k8sContextInput, "Kubectl Context"},
		{k8sKubeconfigInput, "Kubeconfig Path"},
		{k8sShellInput, "Shell"},
		{k8sTagsInput, "Tags (comma-separated)"},
	}

	for _, field := range fields {
		fieldStyle := m.styles.FormField
		if m.focused == field.index {
			fieldStyle = m.styles.FocusedLabel
		}
		b.WriteString(fieldStyle.Render(field.label))
		b.WriteString("\n")
		b.WriteString(m.inputs[field.index].View())
		b.WriteString("\n\n")
	}

	if m.err != "" {
		b.WriteString(m.styles.Error.Render("Error: " + m.err))
		b.WriteString("\n\n")
	}

	b.WriteString(m.styles.FormHelp.Render("Tab/Shift+Tab: navigate • Enter on last field: submit"))
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render("Ctrl+S: save • Ctrl+C/Esc: cancel"))
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render("* Required fields"))

	return b.String()
}

// validationError for form validation
type validationError struct {
	field   string
	message string
}

func (e *validationError) Error() string {
	return e.message
}

// k8sEditFormModel represents the form for editing a k8s host
type k8sEditFormModel struct {
	inputs      []textinput.Model
	focused     int
	err         string
	styles      Styles
	success     bool
	width       int
	height      int
	originalName string
}

// NewK8sEditForm creates a new k8s edit form with existing host data
func NewK8sEditForm(hostName string, styles Styles, width, height int) (*k8sEditFormModel, error) {
	host, err := config.GetK8sHost(hostName)
	if err != nil {
		return nil, err
	}

	inputs := make([]textinput.Model, 8)

	// Name input
	inputs[k8sNameInput] = textinput.New()
	inputs[k8sNameInput].Placeholder = "display-name"
	inputs[k8sNameInput].SetValue(host.Name)
	inputs[k8sNameInput].Focus()
	inputs[k8sNameInput].CharLimit = 50
	inputs[k8sNameInput].Width = 30

	// Namespace input
	inputs[k8sNamespaceInput] = textinput.New()
	inputs[k8sNamespaceInput].Placeholder = "default"
	inputs[k8sNamespaceInput].SetValue(host.Namespace)
	inputs[k8sNamespaceInput].CharLimit = 63
	inputs[k8sNamespaceInput].Width = 30

	// Pod input
	inputs[k8sPodInput] = textinput.New()
	inputs[k8sPodInput].Placeholder = "pod-name"
	inputs[k8sPodInput].SetValue(host.Pod)
	inputs[k8sPodInput].CharLimit = 253
	inputs[k8sPodInput].Width = 40

	// Container input
	inputs[k8sContainerInput] = textinput.New()
	inputs[k8sContainerInput].Placeholder = "(optional) container-name"
	inputs[k8sContainerInput].SetValue(host.Container)
	inputs[k8sContainerInput].CharLimit = 63
	inputs[k8sContainerInput].Width = 30

	// Context input
	inputs[k8sContextInput] = textinput.New()
	inputs[k8sContextInput].Placeholder = "(optional) kubectl context"
	inputs[k8sContextInput].SetValue(host.Context)
	inputs[k8sContextInput].CharLimit = 100
	inputs[k8sContextInput].Width = 40

	// Kubeconfig input
	inputs[k8sKubeconfigInput] = textinput.New()
	inputs[k8sKubeconfigInput].Placeholder = "(optional) ~/.kube/config"
	inputs[k8sKubeconfigInput].SetValue(host.Kubeconfig)
	inputs[k8sKubeconfigInput].CharLimit = 200
	inputs[k8sKubeconfigInput].Width = 50

	// Shell input
	inputs[k8sShellInput] = textinput.New()
	inputs[k8sShellInput].Placeholder = "/bin/bash"
	inputs[k8sShellInput].SetValue(host.Shell)
	inputs[k8sShellInput].CharLimit = 100
	inputs[k8sShellInput].Width = 30

	// Tags input
	inputs[k8sTagsInput] = textinput.New()
	inputs[k8sTagsInput].Placeholder = "production, web, database"
	if len(host.Tags) > 0 {
		inputs[k8sTagsInput].SetValue(strings.Join(host.Tags, ", "))
	}
	inputs[k8sTagsInput].CharLimit = 200
	inputs[k8sTagsInput].Width = 50

	return &k8sEditFormModel{
		inputs:       inputs,
		focused:      k8sNameInput,
		styles:       styles,
		width:        width,
		height:       height,
		originalName: hostName,
	}, nil
}

func (m *k8sEditFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *k8sEditFormModel) Update(msg tea.Msg) (*k8sEditFormModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.styles = NewStyles(m.width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, func() tea.Msg { return k8sEditFormCancelMsg{} }

		case "ctrl+s":
			return m, m.submitForm()

		case "tab", "shift+tab", "enter", "up", "down":
			return m, m.handleNavigation(msg.String())
		}

	case k8sEditFormSubmitMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.success = true
			m.err = ""
		}
		return m, nil
	}

	// Update inputs
	cmd := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmd[i] = m.inputs[i].Update(msg)
	}
	cmds = append(cmds, cmd...)

	return m, tea.Batch(cmds...)
}

func (m *k8sEditFormModel) handleNavigation(key string) tea.Cmd {
	if key == "enter" && m.focused == k8sTagsInput {
		return m.submitForm()
	}

	if key == "up" || key == "shift+tab" {
		m.focused--
	} else {
		m.focused++
	}

	if m.focused >= len(m.inputs) {
		m.focused = 0
	} else if m.focused < 0 {
		m.focused = len(m.inputs) - 1
	}

	return m.updateFocus()
}

func (m *k8sEditFormModel) updateFocus() tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.inputs {
		if i == m.focused {
			cmds = append(cmds, m.inputs[i].Focus())
		} else {
			m.inputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (m *k8sEditFormModel) submitForm() tea.Cmd {
	return func() tea.Msg {
		name := strings.TrimSpace(m.inputs[k8sNameInput].Value())
		namespace := strings.TrimSpace(m.inputs[k8sNamespaceInput].Value())
		pod := strings.TrimSpace(m.inputs[k8sPodInput].Value())
		container := strings.TrimSpace(m.inputs[k8sContainerInput].Value())
		context := strings.TrimSpace(m.inputs[k8sContextInput].Value())
		kubeconfig := strings.TrimSpace(m.inputs[k8sKubeconfigInput].Value())
		shell := strings.TrimSpace(m.inputs[k8sShellInput].Value())
		tagsStr := strings.TrimSpace(m.inputs[k8sTagsInput].Value())

		if name == "" {
			return k8sEditFormSubmitMsg{err: &validationError{field: "Name", message: "Name is required"}}
		}
		if namespace == "" {
			namespace = "default"
		}
		if pod == "" {
			return k8sEditFormSubmitMsg{err: &validationError{field: "Pod", message: "Pod name is required"}}
		}
		if shell == "" {
			shell = "/bin/bash"
		}

		var tags []string
		if tagsStr != "" {
			for _, tag := range strings.Split(tagsStr, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					tags = append(tags, tag)
				}
			}
		}

		host := config.K8sHost{
			Name:       name,
			Namespace:  namespace,
			Pod:        pod,
			Container:  container,
			Context:    context,
			Kubeconfig: kubeconfig,
			Shell:      shell,
			Tags:       tags,
		}

		err := config.UpdateK8sHost(m.originalName, host)
		return k8sEditFormSubmitMsg{hostName: name, err: err}
	}
}

func (m *k8sEditFormModel) View() string {
	if m.success {
		return ""
	}

	var b strings.Builder

	b.WriteString(m.styles.FormTitle.Render("Edit Kubernetes Host"))
	b.WriteString("\n\n")

	fields := []struct {
		index int
		label string
	}{
		{k8sNameInput, "Display Name *"},
		{k8sNamespaceInput, "Namespace *"},
		{k8sPodInput, "Pod Name *"},
		{k8sContainerInput, "Container"},
		{k8sContextInput, "Kubectl Context"},
		{k8sKubeconfigInput, "Kubeconfig Path"},
		{k8sShellInput, "Shell"},
		{k8sTagsInput, "Tags (comma-separated)"},
	}

	for _, field := range fields {
		fieldStyle := m.styles.FormField
		if m.focused == field.index {
			fieldStyle = m.styles.FocusedLabel
		}
		b.WriteString(fieldStyle.Render(field.label))
		b.WriteString("\n")
		b.WriteString(m.inputs[field.index].View())
		b.WriteString("\n\n")
	}

	if m.err != "" {
		b.WriteString(m.styles.Error.Render("Error: " + m.err))
		b.WriteString("\n\n")
	}

	b.WriteString(m.styles.FormHelp.Render("Tab/Shift+Tab: navigate • Enter on last field: submit"))
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render("Ctrl+S: save • Ctrl+C/Esc: cancel"))
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render("* Required fields"))

	return b.String()
}
