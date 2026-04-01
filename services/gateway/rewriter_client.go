package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// RewriteRequest is what we send to the Python rewriter
type RewriteRequest struct {
	Query          string `json:"query"`
	BottleneckType string `json:"bottleneck_type"`
	RelationName   string `json:"relation_name"`
	RowsFiltered   int    `json:"rows_filtered"`
}

// RewriteResponse is what we get back from the Python rewriter
type RewriteResponse struct {
	OriginalQuery        string   `json:"original_query"`
	RewrittenQuery       string   `json:"rewritten_query"`
	Explanation          string   `json:"explanation"`
	EstimatedImprovement string   `json:"estimated_improvement"`
	RulesApplied         []string `json:"rules_applied"`
}

// CallRewriter sends the analysis to Python and gets back a rewrite suggestion
func CallRewriter(query string, analysis AnalysisResult) (*RewriteResponse, error) {
	// Only call rewriter if we found a bottleneck
	if !analysis.IsBottleneck {
		return nil, nil
	}

	// Build the request
	rewriteReq := RewriteRequest{
		Query:          query,
		BottleneckType: analysis.NodeType,
		RelationName:   analysis.RelationName,
		RowsFiltered:   analysis.RowsFiltered,
	}

	// Convert to JSON
	body, err := json.Marshal(rewriteReq)
	if err != nil {
		return nil, fmt.Errorf("error building rewrite request: %v", err)
	}

	// Call the Python rewriter service
	resp, err := http.Post(
		"http://localhost:8000/rewrite",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, fmt.Errorf("error calling rewriter service: %v", err)
	}
	defer resp.Body.Close()

	// Parse the response
	var rewriteResp RewriteResponse
	err = json.NewDecoder(resp.Body).Decode(&rewriteResp)
	if err != nil {
		return nil, fmt.Errorf("error parsing rewriter response: %v", err)
	}

	return &rewriteResp, nil
}
