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
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/serverstatus", serverstatusHandler)

	return http.ListenAndServe(":"+webPort, nil)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	log.Info("New Web User Routed", "user", r.RemoteAddr, "loc", r.URL)
	tmpl := template.Must(template.ParseFiles("template.html"))
	tmpl.Execute(w, nil)
}

func serverstatusHandler(w http.ResponseWriter, r *http.Request) {

	headerKeys := []string{}
	for key := range r.Header {
		headerKeys = append(headerKeys, key)
	}

	reqIp := r.Header.Get("X-Forwarded-For")

	if reqIp == "" {
		reqIp = strings.Split(r.RemoteAddr, ":")[0]
	}

	log.Info("User requests serverstatus", "user", r.RemoteAddr, "headers", headerKeys)

	var respIp string

	err := db.QueryRow("select ip from addresses where ip=? and dtm>datetime('now', 'localtime')", reqIp).Scan(&respIp)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Fatal(err)
	}

	tmpl := template.Must(template.ParseFiles("response.html"))

	if errors.Is(err, sql.ErrNoRows) {
		fmt.Fprint(w, "")
	} else {
		// fmt.Fprint(w, "Server has been Stopped!")
		tmpl.ExecuteTemplate(w, "popup", nil)
	}
}
