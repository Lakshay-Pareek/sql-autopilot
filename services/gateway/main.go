package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var db *sql.DB

func main() {
	// Load .env file
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Connect to PostgreSQL
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Error connecting to database:", err)
	}
	defer db.Close()

	// Test the connection
	err = db.Ping()
	if err != nil {
		log.Fatal("Cannot reach database:", err)
	}
	log.Println("Connected to PostgreSQL successfully!")

	// Set up routes
	r := mux.NewRouter()
	r.HandleFunc("/health", healthHandler).Methods("GET")
	r.HandleFunc("/analyze", analyzeHandler).Methods("POST")
	r.HandleFunc("/history", historyHandler).Methods("GET")

	// Start server
	port := os.Getenv("GATEWAY_PORT")
	log.Println("Gateway running on port", port)
	log.Fatal(http.ListenAndServe(":"+port, enableCORS(r)))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

func analyzeHandler(w http.ResponseWriter, r *http.Request) {
	// Read the SQL query from request body
	var request struct {
		Query string `json:"query"`
	}

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil || request.Query == "" {
		http.Error(w, "Invalid request. Send: {\"query\": \"your SQL here\"}", http.StatusBadRequest)
		return
	}

	// Run EXPLAIN ANALYZE on the query
	explainQuery := "EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) " + request.Query
	rows, err := db.Query(explainQuery)
	if err != nil {
		http.Error(w, "Error analyzing query: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Collect the EXPLAIN output
	var planLines []string
	for rows.Next() {
		var line string
		rows.Scan(&line)
		planLines = append(planLines, line)
	}

	// Parse the plan into our struct
	planNode, err := ParsePlan(planLines)
	if err != nil {
		http.Error(w, "Error parsing plan: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Analyze the plan and detect bottlenecks
	analysis := AnalyzePlan(planNode)

	// Call Python rewriter if bottleneck detected
	rewrite, err := CallRewriter(request.Query, analysis)
	if err != nil {
		log.Println("Warning: rewriter service unavailable:", err)
	}

	// Save to history
	improvement := ""
	if rewrite != nil {
		improvement = rewrite.EstimatedImprovement
	}
	SaveHistory(request.Query, analysis, improvement)

	// Build final response
	response := map[string]interface{}{
		"query":    request.Query,
		"analysis": analysis,
	}

	// Add rewrite suggestion if available
	if rewrite != nil {
		response["rewrite"] = rewrite
	}

	// Return everything as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func historyHandler(w http.ResponseWriter, r *http.Request) {
	history, err := GetHistory()
	if err != nil {
		http.Error(w, "Error fetching history: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"history": history,
	})
}
