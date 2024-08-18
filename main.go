package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Team struct {
	Name      string `json:"name"`
	City      string `json:"city"`
	FoundedIn int    `json:"founded_in"`
}

func teamHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Contenty-Type", "application/json")

	team := Team{
		Name:      "Corinthians",
		City:      "São Paulo",
		FoundedIn: 1910,
	}

	json.NewEncoder(w).Encode(team)
}

func main() {
	http.HandleFunc("/team", teamHandler)

	fmt.Println("Server is running")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}
