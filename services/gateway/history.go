package main

import (
	"log"
	"time"
)

type QueryHistory struct {
	ID                   int       `json:"id"`
	Query                string    `json:"query"`
	NodeType             string    `json:"node_type"`
	TotalCost            float64   `json:"total_cost"`
	ActualTime           float64   `json:"actual_time"`
	ActualRows           int       `json:"actual_rows"`
	RowsFiltered         int       `json:"rows_filtered"`
	IsBottleneck         bool      `json:"is_bottleneck"`
	Warning              string    `json:"warning"`
	EstimatedImprovement string    `json:"estimated_improvement"`
	CreatedAt            time.Time `json:"created_at"`
}

// SaveHistory saves a query analysis to the database
func SaveHistory(query string, analysis AnalysisResult, improvement string) {
	_, err := db.Exec(`
		INSERT INTO query_history 
		(query, node_type, total_cost, actual_time, actual_rows, rows_filtered, is_bottleneck, warning, estimated_improvement)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		query,
		analysis.NodeType,
		analysis.TotalCost,
		analysis.ActualTime,
		analysis.ActualRows,
		analysis.RowsFiltered,
		analysis.IsBottleneck,
		analysis.Warning,
		improvement,
	)
	if err != nil {
		log.Println("Warning: could not save history:", err)
	}
}

// GetHistory returns the last 20 analyzed queries
func GetHistory() ([]QueryHistory, error) {
	rows, err := db.Query(`
		SELECT id, query, node_type, total_cost, actual_time, actual_rows, 
		rows_filtered, is_bottleneck, warning, estimated_improvement, created_at
		FROM query_history
		ORDER BY created_at DESC
		LIMIT 20
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []QueryHistory
	for rows.Next() {
		var h QueryHistory
		err := rows.Scan(
			&h.ID,
			&h.Query,
			&h.NodeType,
			&h.TotalCost,
			&h.ActualTime,
			&h.ActualRows,
			&h.RowsFiltered,
			&h.IsBottleneck,
			&h.Warning,
			&h.EstimatedImprovement,
			&h.CreatedAt,
		)
		if err != nil {
			log.Println("Warning: error scanning history row:", err)
			continue
		}
		history = append(history, h)
	}

	return history, nil
}
