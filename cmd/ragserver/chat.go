package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/a-h/ragserver/client"
	"github.com/a-h/ragserver/models"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
)

var defaultSystemPrompt = `You are a chatbot that provides information from context that is provided to you.

Answer the questions using information from the context you have been given.
`

type ChatCommand struct {
	RAGServerURL     string `help:"The URL of the RAG server." env:"RAG_SERVER_URL" default:"http://localhost:9020"`
	RAGServerAPIKey  string `help:"The API key for the RAG server." env:"RAG_SERVER_API_KEY" default:""`
	SystemPromptFile string `help:"The system prompt to use." env:"SYSTEM_PROMPT" default:""`
	LogLevel         string `help:"The log level to use." env:"LOG_LEVEL" default:"info"`
}

func (c ChatCommand) Run(ctx context.Context) (err error) {
	rsc := client.New(c.RAGServerURL, c.RAGServerAPIKey)

	systemPrompt := defaultSystemPrompt
	if c.SystemPromptFile != "" {
		pfBytes, err := os.ReadFile(c.SystemPromptFile)
		if err != nil {
			return fmt.Errorf("failed to read system prompt file: %w", err)
		}
		systemPrompt = string(pfBytes)
	}

	var req models.ChatPostRequest
	req.Messages = append(req.Messages, models.ChatMessage{
		Type:    models.ChatMessageTypeSystem,
		Content: systemPrompt,
	})

	toLLM := make(chan models.ChatMessage)
	fromLLM := make(chan []models.ChatMessage)
	errors := make(chan error)
	defer close(toLLM)
	defer close(fromLLM)
	defer close(errors)

	go func() {
		for toSend := range toLLM {
			req.Messages = append(req.Messages, toSend)
			msgIndex := len(req.Messages)
			req.Messages = append(req.Messages, models.ChatMessage{
				Type:    models.ChatMessageTypeAI,
				Content: "",
			})

			buf := new(bytes.Buffer)
			f := func(ctx context.Context, chunk []byte) error {
				_, err = buf.Write(chunk)
				if err != nil {
					return err
				}
				req.Messages[msgIndex].Content = buf.String()
				fromLLM <- req.Messages
				return err
			}
			if err = rsc.ChatPost(ctx, req, f); err != nil {
				errors <- err
				return
			}
		}
	}()

	p := tea.NewProgram(newModel(ctx, toLLM, fromLLM, errors))
	if _, err = p.Run(); err != nil {
		return err
	}
	return nil
}

// Dracula color scheme.
var (
	Background  = lipgloss.Color("#282a36")
	CurrentLine = lipgloss.Color("#44475a")
	Selection   = lipgloss.Color("#44475a")
	Foreground  = lipgloss.Color("#f8f8f2")
	Comment     = lipgloss.Color("#6272a4")
	Cyan        = lipgloss.Color("#8be9fd")
	Green       = lipgloss.Color("#50fa7b")
	Orange      = lipgloss.Color("#ffb86c")
	Pink        = lipgloss.Color("#ff79c6")
	Purple      = lipgloss.Color("#bd93f9")
	Red         = lipgloss.Color("#ff5555")
	Yellow      = lipgloss.Color("#f1fa8c")
)

var headerStyle = lipgloss.NewStyle().Background(CurrentLine).Foreground(Purple).Bold(true).Margin(10).Padding(1).PaddingTop(0)

var header = `
 _______  __   __  _______  _______  _______  _______  _______ 
|       ||  | |  ||   _   ||       ||  _    ||       ||       |
|       ||  |_|  ||  |_|  ||_     _|| |_|   ||   _   ||_     _|
|       ||       ||       |  |   |  |       ||  | |  |  |   |  
|      _||       ||       |  |   |  |  _   | |  |_|  |  |   |  
|     |_ |   _   ||   _   |  |   |  | |_|   ||       |  |   |  
|_______||__| |__||__| |__|  |___|  |_______||_______|  |___|
`

type model struct {
	viewport viewport.Model
	textarea textarea.Model
	err      error
	ctx      context.Context

	// Chatbot interactions.
	toLLM   chan models.ChatMessage
	fromLLM chan []models.ChatMessage
	errors  chan error
}

func newModel(ctx context.Context, toLLM chan models.ChatMessage, fromLLM chan []models.ChatMessage, errors chan error) model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 280

	ta.SetHeight(3)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	ta.ShowLineNumbers = false

	vp := viewport.New(80, 20)
	vp.SetContent(headerStyle.Render(header))

	ta.KeyMap.InsertNewline.SetEnabled(false)

	return model{
		ctx:      ctx,
		textarea: ta,
		viewport: vp,
		err:      nil,
		fromLLM:  fromLLM,
		toLLM:    toLLM,
		errors:   errors,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.subscribeToFromLLM(),
		m.subscribeToErrors(),
	)
}

func (m model) subscribeToFromLLM() tea.Cmd {
	return func() tea.Msg {
		select {
		case x := <-m.fromLLM:
			return x
		case <-m.ctx.Done():
			return nil
		}
	}
}

func (m model) subscribeToErrors() tea.Cmd {
	return func() tea.Msg {
		select {
		case x := <-m.errors:
			return x
		case <-m.ctx.Done():
			return nil
		}
	}
}

var messageTypeToStyle = map[models.ChatMessageType]lipgloss.Style{
	models.ChatMessageTypeSystem: lipgloss.NewStyle().Padding(1).Margin(1).MarginBottom(0).MaxWidth(90).Background(Background).Foreground(Green),
	models.ChatMessageTypeHuman:  lipgloss.NewStyle().Padding(1).Margin(1).MarginBottom(0).Background(Background).Foreground(Pink),
	models.ChatMessageTypeAI:     lipgloss.NewStyle().Padding(1).Margin(1).MarginBottom(0).Background(Background).Foreground(Cyan),
}

var messageTypeToIcon = map[models.ChatMessageType]string{
	models.ChatMessageTypeSystem: "🤖",
	models.ChatMessageTypeHuman:  "🥷",
	models.ChatMessageTypeAI:     "✨",
}

func formatMessage(msg models.ChatMessage) string {
	style, ok := messageTypeToStyle[msg.Type]
	if !ok {
		return msg.Content
	}
	icon, ok := messageTypeToIcon[msg.Type]
	if !ok {
		icon = "🤷"
	}
	wrapped := wordwrap.String(strings.TrimSpace(icon+" "+msg.Content), 80)
	return style.Render(wrapped)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case error:
		m.err = msg
		return m, m.subscribeToErrors()
	case []models.ChatMessage:
		var sb strings.Builder
		for _, cm := range msg {
			sb.WriteString(formatMessage(cm))
			sb.WriteString("\n")
		}
		m.viewport.SetContent(sb.String())
		m.viewport.GotoBottom()
		return m, m.subscribeToFromLLM()
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - m.textarea.Height() - 3
		m.textarea.SetWidth(msg.Width)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			// Quit.
			fmt.Println(m.textarea.Value())
			return m, tea.Quit
		case "enter":
			v := m.textarea.Value()

			if v == "" {
				// Don't send empty messages.
				return m, nil
			}

			// Simulate sending a message. In your application you'll want to
			// also return a custom command to send the message off to
			// a server.
			m.textarea.Reset()
			m.toLLM <- models.ChatMessage{
				Type:    models.ChatMessageTypeHuman,
				Content: v,
			}
			return m, nil
		default:
			// Send all other keypresses to the textarea.
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}

	case cursor.BlinkMsg:
		// Textarea should also process cursor blinks.
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd

	default:
		return m, nil
	}
}

func (m model) View() string {
	return fmt.Sprintf("%s\n\n%s",
		m.viewport.View(),
		m.textarea.View(),
	) + "\n\n"
}
