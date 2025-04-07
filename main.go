package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/snyk/snyk-code-review-exercise/api"
)

func main() {
	// REVIEW: Initialize a logger here and pass it into the API package.
	// Something like the following:
	// var logger = log.New(log.Config{
	//	  Format:    "json",
	//	  Level:     slog.LevelInfo,
	//	  AddSource: true,
	// })
	handler := api.New()
	fmt.Println("Server running on http://localhost:3000/")
	if err := http.ListenAndServe("localhost:3000", handler); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
