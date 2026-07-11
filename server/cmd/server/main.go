package main

import (
	"log"

	"timeplanner/server/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
