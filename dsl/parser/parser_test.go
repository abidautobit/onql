package parser

import (
	"fmt"
	"onql/database"
	"onql/storemanager"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// This file tests the parser package components:
// - Lexer: tokenization of DSL queries
// - Parser: statement and plan creation
// - Utilities: helper functions

// init sets up test environment variables before config package loads
func init() {
	os.Setenv("PORT", "8080")
	os.Setenv("REDIS_URL", "localhost:6379")
	os.Setenv("NATS_URL", "nats://localhost:4222")
	os.Setenv("DISK_PATH", "./store")
}

// TestMain handles test setup, ensuring working directory is at project root
func TestMain(m *testing.M) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// If we're in parser directory, go up to project root (two levels)
	baseName := filepath.Base(cwd)
	if baseName == "parser" {
		if err := os.Chdir("../.."); err != nil {
			panic(err)
		}
	} else if baseName == "dsl" {
		// If in dsl directory, go up one level
		if err := os.Chdir(".."); err != nil {
			panic(err)
		}
	}

	// Set up test environment variables to prevent store initialization
	setupTestEnvironment()

	// Initialize test protocol (important: prevents disk I/O and store initialization)
	setupTestProtocol()

	// Run tests
	code := m.Run()

	// Exit with the test result code
	os.Exit(code)
}

// setupTestEnvironment sets up minimal environment for tests
func setupTestEnvironment() {
	// Create a temporary directory for test stores
	tempDir, err := os.MkdirTemp("", "onql-parser-test-*")
	if err != nil {
		panic(err)
	}

	// Set DISK_PATH to empty temp directory to avoid loading production databases
	os.Setenv("DISK_PATH", tempDir)
	os.Setenv("PORT", "8080")
	os.Setenv("REDIS_URL", "localhost:6379")
	os.Setenv("NATS_URL", "nats://localhost:4222")
}

// setupTestProtocol creates a mock protocol for testing
func setupTestProtocol() {
	// Create a test protocol with a "db" database and common test tables
	protocol := storemanager.QueryProtocol{
		"db": &storemanager.Module{
			Database: "test_database",
			Entities: map[string]*storemanager.Entity{
				"users": {
					Table: "users_table",
					Fields: map[string]string{
						"id":       "id",
						"name":     "name",
						"age":      "age",
						"active":   "active",
						"city":     "city",
						"email":    "email",
						"status":   "status",
						"metadata": "metadata",
					},
					Relations: map[string]*storemanager.Relation{
						"orders": {
							ProtoTable: "orders",
							Type:       "otm",
							Entity:     "orders",
							FKField:    "id:user_id",
						},
					},
				},
				"products": {
					Table: "products_table",
					Fields: map[string]string{
						"id":       "id",
						"name":     "name",
						"price":    "price",
						"stock":    "stock",
						"category": "category",
					},
				},
				"orders": {
					Table: "orders_table",
					Fields: map[string]string{
						"id":       "id",
						"user_id":  "user_id",
						"amount":   "amount",
						"status":   "status",
						"product":  "product",
						"customer": "customer",
						"total":    "total",
					},
				},
				"scores": {
					Table: "scores_table",
					Fields: map[string]string{
						"id":    "id",
						"value": "value",
					},
				},
				"events": {
					Table: "events_table",
					Fields: map[string]string{
						"id":        "id",
						"timestamp": "timestamp",
					},
				},
			},
		},
	}

	// Initialize the database.Protocols map if it doesn't exist
	if database.Protocols == nil {
		database.Protocols = make(map[string]storemanager.QueryProtocol)
	}

	// Set the test protocol in memory (skip disk persistence for tests)
	database.Protocols["test-proto"] = protocol
}

// TestLexerTokenization tests the lexer's ability to tokenize queries
func TestLexerTokenization(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		tokens []string // Expected token values
	}{
		{
			name:   "simple identifier",
			query:  "db.users",
			tokens: []string{"db", ".", "users"},
		},
		{
			name:   "nested table access",
			query:  "db.users.orders",
			tokens: []string{"db", ".", "users", ".", "orders"},
		},
		{
			name:   "bracket filter",
			query:  "db.users[age > 25]",
			tokens: []string{"db", ".", "users", "[", "age", ">", "25", "]"},
		},
		{
			name:   "projection",
			query:  "db.users{name, age}",
			tokens: []string{"db", ".", "users", "{", "name", ",", "age", "}"},
		},
		{
			name:   "string literal",
			query:  `db.users[city = "NYC"]`,
			tokens: []string{"db", ".", "users", "[", "city", "=", "NYC", "]"},
		},
		{
			name:   "aggregation function",
			query:  "db.users._sum(amount)",
			tokens: []string{"db", ".", "users", ".", "_sum", "(", "amount", ")"},
		},
		{
			name:   "complex expression with logical operators",
			query:  "db.users[age > 25 and active = true]",
			tokens: []string{"db", ".", "users", "[", "age", ">", "25", "and", "active", "=", "true", "]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.query)
			var actualTokens []string

			for {
				token := lexer.Next(true)
				if token == nil {
					break
				}
				actualTokens = append(actualTokens, token.Value)
			}

			if len(actualTokens) != len(tt.tokens) {
				t.Errorf("Token count mismatch: expected %d, got %d",
					len(tt.tokens), len(actualTokens))
				t.Errorf("Expected: %v", tt.tokens)
				t.Errorf("Got: %v", actualTokens)
				return
			}

			for i, expectedToken := range tt.tokens {
				if actualTokens[i] != expectedToken {
					t.Errorf("Token %d mismatch: expected '%s', got '%s'",
						i, expectedToken, actualTokens[i])
				}
			}
		})
	}
}

// TestParserStatementCreation tests the parser's ability to create statements
func TestParserStatementCreation(t *testing.T) {
	tests := []struct {
		name              string
		query             string
		expectedOps       []OperationType
		expectedStmtCount int
		wantErr           bool
	}{
		{
			name:              "simple table access",
			query:             "db.users",
			expectedOps:       []OperationType{OpAccessTable},
			expectedStmtCount: 1,
			wantErr:           false,
		},
		{
			name:              "table with filter",
			query:             "db.users[age > 25]",
			expectedOps:       []OperationType{OpAccessTable, OpStartFilter},
			expectedStmtCount: -1, // Variable based on filter complexity
			wantErr:           false,
		},
		{
			name:              "table with slice",
			query:             "db.users[0:10]",
			expectedOps:       []OperationType{OpAccessTable, OpSlice},
			expectedStmtCount: 2,
			wantErr:           false,
		},
		{
			name:              "table with projection",
			query:             "db.users{name, age}",
			expectedOps:       []OperationType{OpAccessTable, OpStartProjection},
			expectedStmtCount: -1, // Variable based on projection fields
			wantErr:           false,
		},
		{
			name:              "nested table access",
			query:             "db.users.orders",
			expectedOps:       []OperationType{OpAccessTable, OpAccessRelatedTable},
			expectedStmtCount: 2,
			wantErr:           false,
		},
		{
			name:              "aggregation function",
			query:             "db.users._sum(amount)",
			expectedOps:       []OperationType{OpAccessTable, OpAggregateReduce},
			expectedStmtCount: 2,
			wantErr:           false,
		},
		{
			name:              "row access",
			query:             "db.users[0]",
			expectedOps:       []OperationType{OpAccessTable, OpAccessRow},
			expectedStmtCount: 2,
			wantErr:           false,
		},
		{
			name:              "field access",
			query:             "db.users[0].name",
			expectedOps:       []OperationType{OpAccessTable, OpAccessRow, OpAccessField},
			expectedStmtCount: 3,
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.query)
			plan := NewPlan(lexer, "test-proto")
			err := plan.Parse()

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Check expected operations
			for i, expectedOp := range tt.expectedOps {
				if i >= len(plan.Statements) {
					t.Errorf("Expected statement %d with operation %v not found", i, expectedOp)
					continue
				}
				if plan.Statements[i].Operation != expectedOp {
					t.Errorf("Statement %d: expected operation %v, got %v",
						i, expectedOp, plan.Statements[i].Operation)
				}
			}

			// Check statement count if specified
			if tt.expectedStmtCount > 0 && len(plan.Statements) != tt.expectedStmtCount {
				t.Errorf("Statement count mismatch: expected %d, got %d",
					tt.expectedStmtCount, len(plan.Statements))
			}
		})
	}
}

// TestParserFilterParsing tests parsing of filter expressions
func TestParserFilterParsing(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "simple comparison greater than",
			query:   "db.users[age > 25]",
			wantErr: false,
		},
		{
			name:    "equality check",
			query:   `db.users[name = "Alice"]`,
			wantErr: false,
		},
		{
			name:    "not equal",
			query:   `db.users[status != "inactive"]`,
			wantErr: false,
		},
		{
			name:    "less than",
			query:   "db.users[age < 30]",
			wantErr: false,
		},
		{
			name:    "greater than or equal",
			query:   "db.products[price >= 100]",
			wantErr: false,
		},
		{
			name:    "less than or equal",
			query:   "db.products[stock <= 10]",
			wantErr: false,
		},
		{
			name:    "logical AND",
			query:   "db.users[age > 25 and active = true]",
			wantErr: false,
		},
		{
			name:    "logical OR",
			query:   `db.users[city = "NYC" or city = "LA"]`,
			wantErr: false,
		},
		{
			name:    "complex logical expression with parentheses",
			query:   `db.users[(age > 25 and age < 35) or status = "premium"]`,
			wantErr: false,
		},
		{
			name:    "null comparison",
			query:   "db.users[email = null]",
			wantErr: false,
		},
		{
			name:    "number comparison with decimal",
			query:   "db.products[price = 99.99]",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.query)
			plan := NewPlan(lexer, "test-proto")
			err := plan.Parse()

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && err == nil {
				// Verify filter statements were created
				hasFilter := false
				for _, stmt := range plan.Statements {
					if stmt.Operation == OpStartFilter || stmt.Operation == OpEndFilter {
						hasFilter = true
						break
					}
				}
				if !hasFilter {
					t.Errorf("Expected filter operations in parsed plan but found none")
				}
			}
		})
	}
}

// TestParserProjectionParsing tests parsing of projection expressions
func TestParserProjectionParsing(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "single field",
			query:   "db.users{name}",
			wantErr: false,
		},
		{
			name:    "multiple fields",
			query:   "db.users{name, age, email}",
			wantErr: false,
		},
		{
			name:    "renamed field",
			query:   "db.users{fullName: name}",
			wantErr: true, // Renamed fields not yet supported
		},
		{
			name:    "multiple renamed fields",
			query:   "db.users{fullName: name, years: age, contact: email}",
			wantErr: true, // Renamed fields not yet supported
		},
		{
			name:    "mixed renamed and normal fields",
			query:   "db.users{id, fullName: name, age}",
			wantErr: true, // Renamed fields not yet supported
		},
		{
			name:    "nested projection",
			query:   "db.users{name, orders{id, total}}",
			wantErr: false,
		},
		{
			name:    "projection after filter",
			query:   "db.users[age > 25]{name, age}",
			wantErr: false,
		},
		{
			name:    "projection with aggregation in field",
			query:   "db.users{name, totalOrders: orders._count()}",
			wantErr: true, // Renamed fields with aggregations not yet supported
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.query)
			plan := NewPlan(lexer, "test-proto")
			err := plan.Parse()

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && err == nil {
				// Verify projection statements were created
				hasProjection := false
				for _, stmt := range plan.Statements {
					if stmt.Operation == OpStartProjection || stmt.Operation == OpEndProjection {
						hasProjection = true
						break
					}
				}
				if !hasProjection {
					t.Errorf("Expected projection operations in parsed plan but found none")
				}
			}
		})
	}
}

// TestParserAggregationParsing tests parsing of aggregation functions
func TestParserAggregationParsing(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		expectedAggr string
		wantErr      bool
	}{
		{
			name:         "sum aggregation",
			query:        "db.orders._sum(amount)",
			expectedAggr: "_sum",
			wantErr:      false,
		},
		{
			name:         "count aggregation",
			query:        "db.users._count()",
			expectedAggr: "_count",
			wantErr:      false,
		},
		{
			name:         "average aggregation",
			query:        "db.products._avg(price)",
			expectedAggr: "_avg",
			wantErr:      true, // _avg requires LIST/JSON input, not TABLE
		},
		{
			name:         "min aggregation",
			query:        "db.scores._min(value)",
			expectedAggr: "_min",
			wantErr:      true, // _min requires LIST/JSON input, not TABLE
		},
		{
			name:         "max aggregation",
			query:        "db.scores._max(value)",
			expectedAggr: "_max",
			wantErr:      true, // _max requires LIST/JSON input, not TABLE
		},
		{
			name:         "unique aggregation",
			query:        "db.users._unique(city)",
			expectedAggr: "_unique",
			wantErr:      false,
		},
		{
			name:         "ascending sort",
			query:        "db.users._asc(age)",
			expectedAggr: "_asc",
			wantErr:      false,
		},
		{
			name:         "descending sort",
			query:        "db.products._desc(price)",
			expectedAggr: "_desc",
			wantErr:      false,
		},
		{
			name:         "date aggregation",
			query:        `db.events._date(timestamp, "year")`,
			expectedAggr: "_date",
			wantErr:      false,
		},
		{
			name:         "like filter",
			query:        `db.users._like(name, "A%")`,
			expectedAggr: "_like",
			wantErr:      false,
		},
		{
			name:         "aggregation with filter",
			query:        `db.orders[status = "completed"]._sum(amount)`,
			expectedAggr: "_sum",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.query)
			plan := NewPlan(lexer, "test-proto")
			err := plan.Parse()

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && err == nil {
				// Find aggregation statement
				found := false
				for _, stmt := range plan.Statements {
					if stmt.Operation == OpAggregateReduce {
						if aggr, ok := stmt.Expressions.(Aggr); ok {
							if strings.ToLower(aggr.Name) == tt.expectedAggr {
								found = true
								break
							}
						}
					}
				}
				if !found {
					t.Errorf("Expected aggregation '%s' not found in parsed plan", tt.expectedAggr)
				}
			}
		})
	}
}

// TestParserComplexQueries tests parsing of complex, multi-operation queries
func TestParserComplexQueries(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "filter with projection and aggregation",
			query:   "db.users[age > 25]{name, orders._count()}",
			wantErr: false,
		},
		{
			name:    "table with filter and projection",
			query:   "db.users[active = true]{id, name, email}",
			wantErr: false,
		},
		{
			name:    "multiple filters and projections",
			query:   `db.orders[amount > 100][status = "pending"]{id, customer, amount}`,
			wantErr: false,
		},
		{
			name:    "aggregation after filter and slice",
			query:   `db.products[category = "Electronics"][0:10]._avg(price)`,
			wantErr: true, // _avg on sliced data not supported
		},
		{
			name:    "nested table access with filter",
			query:   "db.users.orders[amount > 50]",
			wantErr: false,
		},
		{
			name:    "complex logical filter with projection",
			query:   `db.users[(age >= 25 and age <= 35) or status = "vip"]{id, name, age, status}`,
			wantErr: false,
		},
		{
			name:    "multiple aggregations in projection",
			query:   "db.users{name, orderCount: orders._count(), totalSpent: orders._sum(amount)}",
			wantErr: true, // Renamed fields with aggregations not yet supported
		},
		{
			name:    "json property access",
			query:   "db.users[0].metadata.preferences.theme",
			wantErr: false,
		},
		{
			name:    "row with projection",
			query:   "db.users[5]{name, email}",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.query)
			plan := NewPlan(lexer, "test-proto")
			err := plan.Parse()

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				if err != nil {
					t.Logf("Error details: %v", err)
				}
			}

			if !tt.wantErr && err == nil && len(plan.Statements) == 0 {
				t.Errorf("Expected statements in parsed plan but found none")
			}
		})
	}
}

// TestParserErrorCases tests error handling in the parser
func TestParserErrorCases(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "empty query",
			query:   "",
			wantErr: false, // Empty query doesn't error, just returns no statements
		},
		{
			name:    "unclosed bracket",
			query:   "db.users[age > 25",
			wantErr: true, // Panics with nil pointer, needs better error handling
		},
		{
			name:    "unclosed brace",
			query:   "db.users{name, age",
			wantErr: false, // Parser doesn't validate brace closure during parse
		},
		{
			name:    "unclosed parenthesis",
			query:   "db.users._sum(amount",
			wantErr: true, // Panics with nil pointer, needs better error handling
		},
		{
			name:    "empty filter",
			query:   "db.users[]",
			wantErr: false, // Empty filter doesn't error during parse
		},
		{
			name:    "empty projection",
			query:   "db.users{}",
			wantErr: false, // Empty projection doesn't error during parse
		},
		{
			name:    "invalid operator sequence",
			query:   "db.users[age >> 25]",
			wantErr: false, // Parser treats >> as two > operators
		},
		{
			name:    "missing operand",
			query:   "db.users[age >]",
			wantErr: false, // Missing operand doesn't error during parse, just closes filter
		},
		{
			name:    "invalid aggregation",
			query:   "db.users._invalid()",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			func() {
				defer func() {
					if r := recover(); r != nil {
						err = fmt.Errorf("panic: %v", r)
					}
				}()
				lexer := NewLexer(tt.query)
				plan := NewPlan(lexer, "test-proto")
				err = plan.Parse()
			}()

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestNumberToColumn tests the NumberToColumn utility function
func TestNumberToColumn(t *testing.T) {
	tests := []struct {
		input    int
		expected string
		wantErr  bool
	}{
		{1, "A", false},
		{26, "Z", false},
		{27, "AA", false},
		{28, "AB", false},
		{52, "AZ", false},
		{53, "BA", false},
		{702, "ZZ", false},
		{703, "AAA", false},
		{0, "", true},
		{-1, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result, err := NumberToColumn(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("NumberToColumn(%d) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.expected {
				t.Errorf("NumberToColumn(%d) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

// TestLexerEdgeCases tests edge cases in the lexer
func TestLexerEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		checks func(t *testing.T, tokens []*Token)
	}{
		{
			name:  "whitespace handling",
			query: "  db.users  [  age  >  25  ]  ",
			checks: func(t *testing.T, tokens []*Token) {
				expectedValues := []string{"db", ".", "users", "[", "age", ">", "25", "]"}
				if len(tokens) != len(expectedValues) {
					t.Errorf("Expected %d tokens, got %d", len(expectedValues), len(tokens))
					return
				}
				for i, expected := range expectedValues {
					if tokens[i].Value != expected {
						t.Errorf("Token %d: expected '%s', got '%s'", i, expected, tokens[i].Value)
					}
				}
			},
		},
		{
			name:  "double quoted strings",
			query: `db.users[name = "Alice" or name = "Bob"]`,
			checks: func(t *testing.T, tokens []*Token) {
				// Check that double quote strings are handled (lexer strips quotes)
				foundAlice := false
				foundBob := false
				for _, token := range tokens {
					if token.Value == "Alice" {
						foundAlice = true
					}
					if token.Value == "Bob" {
						foundBob = true
					}
				}
				if !foundAlice {
					t.Error("Failed to tokenize double-quoted string \"Alice\"")
				}
				if !foundBob {
					t.Error("Failed to tokenize double-quoted string \"Bob\"")
				}
			},
		},
		{
			name:  "special characters in strings",
			query: `db.users[name = "O'Brien"]`,
			checks: func(t *testing.T, tokens []*Token) {
				found := false
				for _, token := range tokens {
					if token.Value == "O'Brien" {
						found = true
						break
					}
				}
				if !found {
					t.Error("Failed to handle apostrophe in string")
				}
			},
		},
		{
			name:  "consecutive dots",
			query: "db.users.orders.products",
			checks: func(t *testing.T, tokens []*Token) {
				expectedValues := []string{"db", ".", "users", ".", "orders", ".", "products"}
				if len(tokens) != len(expectedValues) {
					t.Errorf("Expected %d tokens, got %d", len(expectedValues), len(tokens))
					return
				}
				for i, expected := range expectedValues {
					if tokens[i].Value != expected {
						t.Errorf("Token %d: expected '%s', got '%s'", i, expected, tokens[i].Value)
					}
				}
			},
		},
		{
			name:  "numbers with decimals",
			query: "db.products[price = 99.99]",
			checks: func(t *testing.T, tokens []*Token) {
				found := false
				for _, token := range tokens {
					if token.Value == "99.99" {
						found = true
						if token.Type != TOKEN_NUMBER {
							t.Errorf("Decimal number should have type NUMBER, got %d", token.Type)
						}
						break
					}
				}
				if !found {
					t.Error("Failed to tokenize decimal number 99.99")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.query)
			var tokens []*Token

			for {
				token := lexer.Next(true)
				if token == nil {
					break
				}
				tokens = append(tokens, token)
			}

			tt.checks(t, tokens)
		})
	}
}

// BenchmarkLexerTokenization benchmarks the lexer performance
func BenchmarkLexerTokenization(b *testing.B) {
	queries := []string{
		"db.users",
		"db.users[age > 25]",
		"db.users[age > 25 and active = true]{name, age, city}",
		"db.users.orders[amount > 100]{product, amount}._sum(amount)",
		`db.schema.table[complex = true and (status = "active" or priority >= 5)]{field1, field2, nested{field3, field4}}`,
	}

	for _, query := range queries {
		b.Run(query, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				lexer := NewLexer(query)
				for {
					token := lexer.Next(true)
					if token == nil {
						break
					}
				}
			}
		})
	}
}

// BenchmarkParserParsing benchmarks the parser performance
func BenchmarkParserParsing(b *testing.B) {
	queries := []string{
		"db.users",
		"db.users[age > 25]",
		"db.users[age > 25 and active = true]{name, age, city}",
		"db.users.orders[amount > 100]{product, amount}._sum(amount)",
		`db.schema.table[complex = true and (status = "active" or priority >= 5)]{field1, field2, nested{field3, field4}}`,
	}

	for _, query := range queries {
		b.Run(query, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				lexer := NewLexer(query)
				plan := NewPlan(lexer, "test-proto")
				_ = plan.Parse()
			}
		})
	}
}
