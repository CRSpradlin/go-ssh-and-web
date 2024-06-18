package main

import (
	"context"
	"database/sql"
	"errors"

	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	_ "github.com/mattn/go-sqlite3"
)

const (
	host    = "localhost"
	sshPort = "2323"
	webPort = "3000"
	sshbg   = "#222222"
	sshfg   = "#22c55e"
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
			dtm datetime
		);
	`
	_, err = db.Exec(dbInit)
	checkErr(err)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Info("Starting SSH Server", "host", host, "port", sshPort)
	var s *ssh.Server
	go func() {
		if s, err = runSSHServer(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start ssh server", "error", err)
			done <- nil
		}
	}()

	log.Info("Starting Web Server", "host", host, "port", webPort)
	go func() {
		if err = runWebServer(); err != nil {
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
