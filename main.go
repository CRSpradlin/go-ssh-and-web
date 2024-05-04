package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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

	"html/template"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

const (
	host    = "localhost"
	sshPort = "2323"
	webPort = "3000"
	sitebg  = "#222222"
	sitefg  = "#22c55e"
)

var db *sql.DB

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {

	var err error // error is scoped locally so that "=" can be used in the following line instead of ":=" which would overrride global "db"
	db, err = sql.Open("sqlite3", "./db.sqlite")
	checkErr(err)
	defer db.Close()

	dbInit := `
		create table if not exists addresses (
			id integer not null primary key,
			ip text not null,
			dtm date
		);
	`
	_, err = db.Exec(dbInit)
	checkErr(err)

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/serverstatus", serverstatusHandler)

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

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Info("Starting SSH Server", "host", host, "port", sshPort)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start ssh server", "error", err)
			done <- nil
		}
	}()

	log.Info("Starting Web Server", "host", host, "port", webPort)
	go func() {
		if err = http.ListenAndServe(":"+webPort, nil); err != nil {
			log.Error("Could not start web server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("Stopping Servers")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop server", "error", err)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	log.Info("New Web User Routed", "user", r.RemoteAddr, "loc", r.URL)
	tmpl := template.Must(template.ParseFiles("template.html"))
	tmpl.Execute(w, nil)
}

func serverstatusHandler(w http.ResponseWriter, r *http.Request) {
	log.Info("User requests serverstatus", "user", r.RemoteAddr)
	fmt.Fprint(w, "")
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
		remoteAddr: strings.Split(s.RemoteAddr().String(), ":")[0],
	}
	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

func pauseServiceForIp(ip string) {
	log.Info("Stopped service for " + ip)
}

type State struct {
	term       string
	width      int
	height     int
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
	return BasicPage(m, "CRSPRADLIN.DEV ADMIN", "This is the admin site\npress 'r' to restart the site\npress 'q' to exit")
}
