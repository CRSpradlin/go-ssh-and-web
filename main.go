package main

import (
	"context"
	"errors"
	//	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
)

const (
	host   = "localhost"
	port   = "2323"
	sitebg = "#333333"
	sitefg = "#22c55e"
)

func main() {
	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithMiddleware(
			bubbletea.Middleware(teaHandler),
			activeterm.Middleware(), // Bubble Tea apps usually require a PTY.
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Error("Could not start server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", host, "port", port)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop server", "error", err)
	}
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, _ := s.Pty()

	renderer := bubbletea.MakeRenderer(s)

	baseStyle := renderer.NewStyle().Background(lipgloss.Color(sitebg))

	txtStyle := renderer.NewStyle().
		Foreground(lipgloss.Color(sitefg)).
		Inherit(baseStyle)

	titleStyle := renderer.NewStyle().
		BorderBottom(true).
		BorderStyle(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color(sitefg)).
		BorderBackground(lipgloss.Color(sitebg)).
		Foreground(lipgloss.Color(sitefg)).
		MarginLeft(1).
		MarginTop(1).
		//MarginBackground(lipgloss.Color("#4c4c4c")).
		Inherit(baseStyle)

	m := State{
		term:       pty.Term,
		width:      pty.Window.Width,
		height:     pty.Window.Height,
		txtStyle:   txtStyle,
		titleStyle: titleStyle,
	}
	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

type State struct {
	term       string
	width      int
	height     int
	txtStyle   lipgloss.Style
	titleStyle lipgloss.Style
}

func (m State) Init() tea.Cmd {
	return nil
}

func (m State) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func BasicPage(m State, title string, body string) string {
	titleView := m.titleStyle.Render(title)

	styledTitle := lipgloss.Place(
		m.width,
		lipgloss.Height(titleView),
		lipgloss.Left,
		lipgloss.Top,
		m.titleStyle.Render(title),
		lipgloss.WithWhitespaceBackground(lipgloss.Color(sitebg)),
		lipgloss.WithWhitespaceForeground(lipgloss.Color(sitebg)),
	)

	styledBody := lipgloss.Place(
		m.width,
		m.height-lipgloss.Height(styledTitle),
		lipgloss.Center,
		lipgloss.Center,
		m.txtStyle.Render(body),
		lipgloss.WithWhitespaceBackground(lipgloss.Color(sitebg)),
	)

	return lipgloss.JoinVertical(lipgloss.Center, styledTitle, styledBody)
}

func (m State) View() string {
	//	s := fmt.Sprintf("Your term is %s\nYour window size is %dx%d", m.term, m.width, m.height)
	return BasicPage(m, "CRSPRADLIN.DEV ADMIN", "This is the admin site\npress 'q' to exit")
	/*lipgloss.
	 Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.txtStyle.Render(s)+"\n\n"+m.txtStyle.Render("Press 'q' to quit\n"),
			lipgloss.WithWhitespaceBackground(lipgloss.Color("#4c4c4c")))
	*/
}
