package main

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	"html/template"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

func runWebServer() error {
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/serverstatus", serverstatusHandler)

	return http.ListenAndServe(":3000", nil)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	log.Info("New Web User Routed", "user", r.RemoteAddr, "loc", r.URL)
	tmpl := template.Must(template.ParseFiles("template.html"))
	tmpl.Execute(w, nil)
}

func serverstatusHandler(w http.ResponseWriter, r *http.Request) {
	log.Info("User requests serverstatus", "user", r.RemoteAddr)

	reqIp := strings.Split(r.RemoteAddr, ":")[0]
	var respIp string

	err := db.QueryRow("select ip from addresses where ip=? and dtm>datetime('now', 'localtime')", reqIp).Scan(&respIp)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Fatal(err)
	}

	tmpl := template.Must(template.ParseFiles("response.html"))

	if errors.Is(err, sql.ErrNoRows) {
		fmt.Fprint(w, "Server is Running!")
	} else {
		// fmt.Fprint(w, "Server has been Stopped!")
		tmpl.ExecuteTemplate(w, "popup", nil)
	}
}
