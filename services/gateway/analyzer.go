package main

import (
	"encoding/json"
	"fmt"
)

// PlanNode represents a single node in the query execution plan
type PlanNode struct {
	NodeType            string     `json:"Node Type"`
	RelationName        string     `json:"Relation Name"`
	StartupCost         float64    `json:"Startup Cost"`
	TotalCost           float64    `json:"Total Cost"`
	PlanRows            int        `json:"Plan Rows"`
	ActualRows          int        `json:"Actual Rows"`
	ActualLoops         int        `json:"Actual Loops"`
	ActualTotalTime     float64    `json:"Actual Total Time"`
	RowsRemovedByFilter int        `json:"Rows Removed by Filter"`
	Plans               []PlanNode `json:"Plans"`
}

// AnalysisResult holds the final analysis we send back to the user
type AnalysisResult struct {
	NodeType     string           `json:"node_type"`
	RelationName string           `json:"relation_name"`
	TotalCost    float64          `json:"total_cost"`
	ActualTime   float64          `json:"actual_time"`
	ActualRows   int              `json:"actual_rows"`
	RowsFiltered int              `json:"rows_filtered"`
	IsBottleneck bool             `json:"is_bottleneck"`
	Warning      string           `json:"warning"`
	Suggestion   string           `json:"suggestion"`
	Children     []AnalysisResult `json:"children"`
}

// ParsePlan parses the raw EXPLAIN JSON into our PlanNode struct
func ParsePlan(rawPlan []string) (*PlanNode, error) {
	if len(rawPlan) == 0 {
		return nil, fmt.Errorf("empty plan")
	}

	// The plan comes as a JSON array with one element
	var planWrapper []struct {
		Plan PlanNode `json:"Plan"`
	}

	err := json.Unmarshal([]byte(rawPlan[0]), &planWrapper)
	if err != nil {
		return nil, fmt.Errorf("error parsing plan: %v", err)
	}

	if len(planWrapper) == 0 {
		return nil, fmt.Errorf("no plan found")
	}

	plan := planWrapper[0].Plan
	return &plan, nil
}

// AnalyzePlan walks the plan tree and flags bottlenecks
func AnalyzePlan(node *PlanNode) AnalysisResult {
	result := AnalysisResult{
		NodeType:     node.NodeType,
		RelationName: node.RelationName,
		TotalCost:    node.TotalCost,
		ActualTime:   node.ActualTotalTime,
		ActualRows:   node.ActualRows,
		RowsFiltered: node.RowsRemovedByFilter,
		IsBottleneck: false,
	}

	// Rule 1: Sequential scan on large result filtered set = bottleneck
	if node.NodeType == "Seq Scan" && node.RowsRemovedByFilter > 1000 {
		result.IsBottleneck = true
		result.Warning = fmt.Sprintf(
			"Seq Scan removed %d rows — full table scan detected",
			node.RowsRemovedByFilter,
		)
		result.Suggestion = fmt.Sprintf(
			"Add an index on column used in WHERE clause of table '%s'",
			node.RelationName,
		)
	}

	// Rule 2: High cost node
	if node.TotalCost > 1000 {
		result.IsBottleneck = true
		result.Warning = fmt.Sprintf(
			"High cost node detected: %.2f",
			node.TotalCost,
		)
		result.Suggestion = "Consider breaking this query into smaller parts or adding indexes"
	}

	// Rule 3: Actual rows far exceed estimated rows (bad statistics)
	if node.PlanRows > 0 && node.ActualRows > node.PlanRows*10 {
		result.IsBottleneck = true
		result.Warning = fmt.Sprintf(
			"Row estimate was %d but actual was %d — stale statistics",
			node.PlanRows,
			node.ActualRows,
		)
		result.Suggestion = "Run ANALYZE on this table to update statistics"
	}

	// Recursively analyze child nodes
	for _, child := range node.Plans {
		childResult := AnalyzePlan(&child)
		result.Children = append(result.Children, childResult)
	}

	return result
}
