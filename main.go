package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

type Team struct {
	Id            int    `json:"id"`
	Nome_Completo string `json:"nome_completo"`
	Nome          string `json:"nome"`
	Apelido       string `json:"apelido"`
	Sigla         string `json:"sigla"`
}

func teamHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	w.Header().Set("Contenty-Type", "application/json")

	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	query := "SELECT nome_completo, nome, apelido, sigla FROM clubes WHERE id = ?"
	row := db.QueryRow(query, id)

	var team Team
	if err := row.Scan(&team.Nome_Completo, &team.Nome, &team.Apelido, &team.Sigla); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Time não encontrado", http.StatusNotFound)
		} else {
			fmt.Printf("Database error: %v\n", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	json.NewEncoder(w).Encode(team)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	dbHost := os.Getenv("DB_HOST")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbUser, dbPassword, dbHost, dbPort, dbName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		panic(err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/time/{id}", func(w http.ResponseWriter, r *http.Request) {
		teamHandler(w, r, db)
	})

	fmt.Println("Server is running")
	if err := http.ListenAndServe(":8080", r); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}
