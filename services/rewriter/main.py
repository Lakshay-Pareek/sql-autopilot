from fastapi import FastAPI
from pydantic import BaseModel
import sqlparse
from sqlparse.sql import Where
from sqlparse.tokens import Keyword, DML
import re

app = FastAPI(title="SQL Rewriter Service")

# Request model
class RewriteRequest(BaseModel):
    query: str
    bottleneck_type: str
    relation_name: str
    rows_filtered: int

# Response model
class RewriteResponse(BaseModel):
    original_query: str
    rewritten_query: str
    explanation: str
    estimated_improvement: str
    rules_applied: list[str]

@app.get("/health")
def health():
    return {"status": "ok"}

@app.post("/rewrite", response_model=RewriteResponse)
def rewrite_query(request: RewriteRequest):
    rules_applied = []
    rewritten = request.query
    explanation = ""
    estimated_improvement = ""
    index_statement = ""

    # Rule 1: Seq Scan with high filtered rows → suggest index
    if request.bottleneck_type == "Seq Scan" and request.rows_filtered > 1000:
        index_statement, rule, explanation, estimated_improvement = apply_index_suggestion(
            request.query,
            request.relation_name,
            request.rows_filtered
        )
        rules_applied.append(rule)

    # Rule 2: SELECT * → replace with specific columns
    if "SELECT *" in request.query.upper():
        rewritten, rule = apply_select_star_fix(rewritten)
        rules_applied.append(rule)

    # Rule 3: Missing LIMIT on large scan
    if request.rows_filtered > 5000 and "LIMIT" not in request.query.upper():
        rewritten, rule = apply_limit_suggestion(rewritten)
        rules_applied.append(rule)

    if not rules_applied:
        explanation = "No obvious rewrite rules apply. Query looks reasonable."
        estimated_improvement = "0%"

    # Build final rewritten output
    final_rewrite = ""
    if index_statement:
        final_rewrite += f"-- Step 1: Create this index first\n{index_statement}\n\n"
    final_rewrite += f"-- Step 2: Use this optimized query\n{rewritten}"

    return RewriteResponse(
        original_query=request.query,
        rewritten_query=final_rewrite,
        explanation=explanation,
        estimated_improvement=estimated_improvement,
        rules_applied=rules_applied
    )


def apply_index_suggestion(query: str, relation_name: str, rows_filtered: int):
    # Extract the WHERE clause column
    parsed = sqlparse.parse(query)[0]
    where_clause = ""
    for token in parsed.tokens:
        if isinstance(token, Where):
            where_clause = str(token)
            break

    # Extract column name from WHERE clause
    col_match = re.search(r'WHERE\s+(\w+)\s*[=><!]', where_clause, re.IGNORECASE)
    column_name = col_match.group(1) if col_match else "relevant_column"

    index_statement = f"CREATE INDEX idx_{relation_name}_{column_name} ON {relation_name}({column_name});"

    explanation = (
        f"Detected full table scan on '{relation_name}' — "
        f"{rows_filtered} rows were scanned and discarded. "
        f"Adding an index on '{column_name}' will allow PostgreSQL to jump "
        f"directly to matching rows instead of scanning the entire table."
    )

    speedup = min(round(rows_filtered / 100), 50)
    estimated_improvement = f"Up to {speedup}x faster after index creation"

    return index_statement, "add_index", explanation, estimated_improvement


def apply_select_star_fix(query: str):
    rewritten = re.sub(
        r'SELECT\s+\*',
        'SELECT id, customer_name, product, amount, created_at',
        query,
        flags=re.IGNORECASE
    )
    return rewritten, "replace_select_star"


def apply_limit_suggestion(query: str):
    rewritten = query.rstrip(";") + " LIMIT 1000;"
    return rewritten, "add_limit"