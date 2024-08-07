package main

import (
	"database/sql"
	"net"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
)

func runSSHServer() (*ssh.Server, error) {

	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, sshPort)),
		wish.WithMiddleware(
			bubbletea.Middleware(teaHandler),
			activeterm.Middleware(), // Bubble Tea apps usually require a PTY.
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Error("Could not start server", "error", err)
	}

	return s, s.ListenAndServe()
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, _ := s.Pty()

	renderer := bubbletea.MakeRenderer(s)

	outerStyle := renderer.NewStyle()//.Background(lipgloss.Color(sshbg))

	baseStyle := renderer.NewStyle().
		Foreground(lipgloss.Color(sshfg)).
		Inherit(outerStyle)

	txtStyle := renderer.NewStyle().
		Inherit(baseStyle)

	titleStyle := renderer.NewStyle().
		BorderStyle(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color(sshfg)).
		//BorderBackground(lipgloss.Color(sshbg)).
		BorderBottom(true).
		MarginLeft(1).
		MarginTop(1).
		Inherit(baseStyle)

	m := State{
		term:       pty.Term,
		width:      pty.Window.Width,
		height:     pty.Window.Height,
		outerStyle: outerStyle,
		txtStyle:   txtStyle,
		titleStyle: titleStyle,
		remoteAddr: strings.Split(s.RemoteAddr().String(), ":")[0],
	}
	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

func pauseServiceForIp(ip string) {
	var result string = ""
	db.QueryRow("select ip from addresses where ip=?", ip).Scan(&result)
	tx, err := db.Begin()
	checkErr(err)

	var stmt *sql.Stmt
	if result == "" {

		stmt, err = tx.Prepare("insert into addresses(ip, dtm) values(?, datetime('now', 'localtime','+30 second'))")
		checkErr(err)

	} else {
		stmt, err = tx.Prepare("update addresses set dtm=datetime('now', 'localtime', '+30 second') where ip=?")
		checkErr(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(ip)
	checkErr(err)

	err = tx.Commit()
	checkErr(err)

	log.Info("Stopped service for " + ip)
}

type State struct {
	term       string
	width      int
	height     int
	outerStyle lipgloss.Style
	txtStyle   lipgloss.Style
	titleStyle lipgloss.Style
	remoteAddr string
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
		case "r":
			pauseServiceForIp(m.remoteAddr)
			return m, nil
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
		//lipgloss.WithWhitespaceBackground(lipgloss.Color(sshbg)),
		//lipgloss.WithWhitespaceForeground(lipgloss.Color(sshbg)),
	)

	styledBody := lipgloss.Place(
		m.width,
		m.height-lipgloss.Height(styledTitle),
		lipgloss.Center,
		lipgloss.Center,
		m.txtStyle.Render(body),
		//lipgloss.WithWhitespaceBackground(lipgloss.Color(sshbg)),
		//lipgloss.WithWhitespaceForeground(lipgloss.Color(sshbg)),
	)

	return m.outerStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Center, 
			styledTitle, 
			styledBody,
		),
	)
}

func (m State) View() string {
	return BasicPage(m, "CRSPRADLIN.DEV ADMIN", "This is the admin site\npress 'r' to restart the site\npress 'q' to exit")
}
