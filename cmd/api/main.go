package main

import (
	"fmt"
	"goback/internal/server"
)

func main() {
	server := server.NewServer()

	err := server.ListenAndServeTLS("", "")
	if err != nil {
		panic(fmt.Sprintf("cannot start server: %s", err))
	}
}
