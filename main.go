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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Team struct {
	Id        int    `json:"id"`
	Full_Name string `json:"full_name"`
	Name      string `json:"name"`
	Surname   string `json:"surname"`
	Acronym   string `json:"acronym"`
}

type Match struct {
	Id                             int       `json:"id"`
	Championship                   string    `json:"championship"`
	Stadium                        string    `json:"stadium"`
	Data                           time.Time `json:"date"`
	Team_Principal                 string    `json:"team_principal"`
	Scoreboard_Principal           int       `json:"scoreboard_principal"`
	Scoreboard_Principal_Penalties int       `json:"scoreboard_principal_penalties"`
	Team_Visitor                   string    `json:"team_visitor"`
	Scoreboard_Visitor             int       `json:"scoreboard_visitor"`
	Scoreboard_Visitor_Penalties   int       `json:"scoreboard_visitor_penalties"`
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

	query := "SELECT full_name, name, surname, acronym FROM teams WHERE id = ?"
	row := db.QueryRow(query, id)

	var team Team
	if err := row.Scan(&team.Full_Name, &team.Name, &team.Surname, &team.Acronym); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Team not found", http.StatusNotFound)
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
				matches.date, 
				championships.name AS 'championship',
				principal.name AS 'team_principal', 
				matches.scoreboard_principal,
				matches.scoreboard_principal_penalties,
				visitor.name AS 'team_visitor',
				matches.scoreboard_visitor,
				matches.scoreboard_visitor_penalties
			FROM matches
			INNER JOIN championships ON championships.id = matches.id_championship
			INNER JOIN teams AS principal ON principal.id = matches.id_team_principal
			INNER JOIN teams AS visitor ON visitor.id = matches.id_team_visitor
			WHERE
				matches.date < NOW() AND
				(
					matches.id_team_principal = ? OR 
					matches.id_team_visitor = ?
				)
			ORDER BY matches.date DESC
			LIMIT 1`
	row := db.QueryRow(query, idTeam, idTeam)

	var dataStr string
	var lastMatchPlayed Match
	if err := row.Scan(
		&dataStr,
		&lastMatchPlayed.Championship,
		&lastMatchPlayed.Team_Principal,
		&lastMatchPlayed.Scoreboard_Principal,
		&lastMatchPlayed.Scoreboard_Principal_Penalties,
		&lastMatchPlayed.Team_Visitor,
		&lastMatchPlayed.Scoreboard_Visitor,
		&lastMatchPlayed.Scoreboard_Visitor_Penalties); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Last match played not found", http.StatusNotFound)
		} else {
			fmt.Printf("Database error: %v\n", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	lastMatchPlayed.Data, err = time.Parse("2006-01-02 15:04:05", dataStr)
	if err != nil {
		fmt.Printf("Time parsing error: %v\n", err)
		http.Error(w, "Time parsing error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(lastMatchPlayed)
}

var httpRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	},
	[]string{"method", "status_code", "path"},
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
}

func recordMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &statusCapturingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		httpRequestsTotal.WithLabelValues(r.Method, fmt.Sprintf("%d", rw.statusCode), r.URL.Path).Inc()
	})
}

type statusCapturingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *statusCapturingResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
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
		log.Fatalf("Error pinging database: %v", err)
	}

	r := mux.NewRouter()
	r.Handle("/metrics", promhttp.Handler())
	r.HandleFunc("/team/{id}", func(w http.ResponseWriter, r *http.Request) {
		teamHandler(w, r, db)
	})

	r.HandleFunc("/team/{id}/lastMatchPlayed", func(w http.ResponseWriter, r *http.Request) {
		lastMatchPlayedHandler(w, r, db)
	})

	r.Use(recordMetrics)

	fmt.Println("Server is running")
	if err := http.ListenAndServe(":8080", r); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}
