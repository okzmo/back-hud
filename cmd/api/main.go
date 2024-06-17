package main

import (
	"fmt"
	"goback/internal/server"
	"os"
)

func main() {
	server := server.NewServer()

	environment := os.Getenv("ENVIRONMENT")

	if environment == "DEV" {
		err := server.ListenAndServeTLS("", "")
		if err != nil {
			panic(fmt.Sprintf("cannot start server: %s", err))
		}
	} else {
		err := server.ListenAndServe()
		if err != nil {
			panic(fmt.Sprintf("cannot start server: %s", err))
		}
	}
}
