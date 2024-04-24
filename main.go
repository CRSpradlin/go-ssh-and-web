package main

import (
	"github.com/gliderlabs/ssh"
	"io"
	"log"
)

func main() {
	ssh.Handle(func(s ssh.Session) {
		_, err := io.WriteString(s, "Hello World")
		if err != nil {
			log.Fatal("There was a problem")
		}
	})

	log.Fatal(ssh.ListenAndServe(":2222", nil))
}
