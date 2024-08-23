package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

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

type Match struct {
	Id                        int       `json:"id"`
	Data                      time.Time `json:"data"`
	Clube_Mandante            string    `json:"clube_mandante"`
	Placar_Mandante           int       `json:"placar_mandante"`
	Placar_Mandante_Penaltis  int       `json:"placar_mandante_penaltis"`
	Clube_Visitante           string    `json:"clube_visitante"`
	Placar_Visitante          int       `json:"placar_visitante"`
	Placar_Visitante_Penaltis int       `json:"placar_visitante_penaltis"`
	Campeonato                string    `json:"campeonato"`
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

func lastMatchPlayedHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	w.Header().Set("Contenty-Type", "application/json")

	vars := mux.Vars(r)
	idTeamString := vars["id"]

	idTeam, err := strconv.Atoi(idTeamString)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	query := `SELECT
				jogos.data, 
				campeonatos.nome AS 'campeonato',
				mandante.nome AS 'clube_mandante', 
				jogos.placar_mandante,
				jogos.placar_mandante_penaltis,
				visitante.nome AS 'clube_visitante',
				jogos.placar_visitante,
				jogos.placar_visitante_penaltis
			FROM jogos
			INNER JOIN campeonatos ON campeonatos.id = jogos.id_campeonato
			INNER JOIN clubes AS mandante ON mandante.id = jogos.id_clube_mandante
			INNER JOIN clubes AS visitante ON visitante.id = jogos.id_clube_visitante
			WHERE
				jogos.data < NOW() AND
				(
					jogos.id_clube_mandante = ? OR 
					jogos.id_clube_visitante = ?
				)
			ORDER BY jogos.data DESC
			LIMIT 1`
	row := db.QueryRow(query, idTeam, idTeam)

	var dataStr string
	var lastMatchPlayed Match
	if err := row.Scan(
		&dataStr,
		&lastMatchPlayed.Campeonato,
		&lastMatchPlayed.Clube_Mandante,
		&lastMatchPlayed.Placar_Mandante,
		&lastMatchPlayed.Placar_Mandante_Penaltis,
		&lastMatchPlayed.Clube_Visitante,
		&lastMatchPlayed.Placar_Visitante,
		&lastMatchPlayed.Placar_Visitante_Penaltis); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Último jogo não encontrado", http.StatusNotFound)
		} else {
			fmt.Printf("Database error: %v\n", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	lastMatchPlayed.Data, err = time.Parse("2006-01-02 15:04:05", dataStr)
	if err != nil {
		fmt.Printf("Time parsing error: %v\n", err)
		http.Error(w, "Erro ao processar a data", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(lastMatchPlayed)
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

	r.HandleFunc("/time/{id}/lastMatchPlayed", func(w http.ResponseWriter, r *http.Request) {
		lastMatchPlayedHandler(w, r, db)
	})

	fmt.Println("Server is running")
	if err := http.ListenAndServe(":8080", r); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}
