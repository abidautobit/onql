package evaluator

import (
	"onql/dsl/parser"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

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

	// If we're in evaluator directory, go up to project root (two levels)
	baseName := filepath.Base(cwd)
	if baseName == "evaluator" {
		if err := os.Chdir("../.."); err != nil {
			panic(err)
		}
	} else if baseName == "dsl" {
		// If in dsl directory, go up one level
		if err := os.Chdir(".."); err != nil {
			panic(err)
		}
	}

	// Run tests
	code := m.Run()

	// Exit with the test result code
	os.Exit(code)
}

// TestNewEvaluator tests the evaluator constructor
func TestNewEvaluator(t *testing.T) {
	lexer := parser.NewLexer("users")
	plan := parser.NewPlan(lexer, "test-proto")
	_ = plan.Parse()

	tests := []struct {
		name          string
		contextKey    string
		contextValues []string
	}{
		{
			name:          "evaluator without context",
			contextKey:    "",
			contextValues: nil,
		},
		{
			name:          "evaluator with context",
			contextKey:    "user_id",
			contextValues: []string{"123"},
		},
		{
			name:          "evaluator with multiple context values",
			contextKey:    "filter",
			contextValues: []string{"value1", "value2", "value3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := NewEvaluator(plan, tt.contextKey, tt.contextValues)

			if ev == nil {
				t.Fatal("NewEvaluator returned nil")
			}

			if ev.Plan != plan {
				t.Error("Evaluator plan not set correctly")
			}

			if ev.ContextKey != tt.contextKey {
				t.Errorf("ContextKey = %v, want %v", ev.ContextKey, tt.contextKey)
			}

			if !reflect.DeepEqual(ev.ContextValues, tt.contextValues) {
				t.Errorf("ContextValues = %v, want %v", ev.ContextValues, tt.contextValues)
			}

			if ev.Memory == nil {
				t.Error("Memory map not initialized")
			}

			if ev.Result == nil {
				t.Error("Result not initialized")
			}
		})
	}
}

// TestSetMemoryValue tests the memory value setter with type narrowing
func TestSetMemoryValue(t *testing.T) {
	ev := &Evaluator{
		Memory: make(map[string]interface{}),
	}

	tests := []struct {
		name         string
		key          string
		value        interface{}
		expectedType string
	}{
		{
			name:         "string value",
			key:          "test_string",
			value:        "hello world",
			expectedType: "STRING",
		},
		{
			name:         "number value",
			key:          "test_number",
			value:        float64(42),
			expectedType: "NUMBER",
		},
		{
			name:         "boolean value",
			key:          "test_bool",
			value:        true,
			expectedType: "BOOL",
		},
		{
			name:         "null value",
			key:          "test_null",
			value:        nil,
			expectedType: "NULL",
		},
		{
			name:         "array of strings",
			key:          "test_arr_str",
			value:        []string{"a", "b", "c"},
			expectedType: "ARRAY_OF_STRING",
		},
		{
			name:         "array of numbers",
			key:          "test_arr_num",
			value:        []float64{1.0, 2.0, 3.0},
			expectedType: "ARRAY_OF_NUMBER",
		},
		{
			name:         "array of booleans",
			key:          "test_arr_bool",
			value:        []bool{true, false, true},
			expectedType: "ARRAY_OF_BOOL",
		},
		{
			name:         "row object",
			key:          "test_row",
			value:        map[string]interface{}{"id": float64(1), "name": "test"},
			expectedType: "ROW",
		},
		{
			name: "table array",
			key:  "test_table",
			value: []map[string]interface{}{
				{"id": float64(1), "name": "first"},
				{"id": float64(2), "name": "second"},
			},
			expectedType: "TABLE",
		},
		{
			name:         "mixed array",
			key:          "test_mixed",
			value:        []interface{}{"string", float64(1), true},
			expectedType: "ARRAY_OF_UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev.SetMemoryValue(tt.key, tt.value)

			// Check value was stored
			if _, exists := ev.Memory[tt.key]; !exists {
				t.Errorf("Memory value %s not stored", tt.key)
				return
			}

			// Check metadata type
			metaKey := tt.key + "_meta_type"
			metaType, exists := ev.Memory[metaKey]
			if !exists {
				t.Errorf("Metadata type for %s not stored", tt.key)
				return
			}

			if metaType != tt.expectedType {
				t.Errorf("Metadata type = %v, want %v", metaType, tt.expectedType)
			}

			// Verify stored value matches expected (after narrowing)
			stored := ev.Memory[tt.key]
			if tt.value == nil && stored != nil {
				t.Errorf("Expected nil, got %v", stored)
			}
		})
	}
}

// TestNarrowTypes tests the type narrowing functionality
func TestNarrowTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		checkFn  func(interface{}) bool
		typeDesc string
	}{
		{
			name:  "narrow []interface{} of strings to []string",
			input: []interface{}{"a", "b", "c"},
			checkFn: func(v interface{}) bool {
				_, ok := v.([]string)
				return ok
			},
			typeDesc: "[]string",
		},
		{
			name:  "narrow []interface{} of numbers to []float64",
			input: []interface{}{float64(1), float64(2), float64(3)},
			checkFn: func(v interface{}) bool {
				_, ok := v.([]float64)
				return ok
			},
			typeDesc: "[]float64",
		},
		{
			name:  "narrow []interface{} of bools to []bool",
			input: []interface{}{true, false, true},
			checkFn: func(v interface{}) bool {
				_, ok := v.([]bool)
				return ok
			},
			typeDesc: "[]bool",
		},
		{
			name: "narrow []interface{} of maps to []map[string]interface{}",
			input: []interface{}{
				map[string]interface{}{"id": float64(1)},
				map[string]interface{}{"id": float64(2)},
			},
			checkFn: func(v interface{}) bool {
				_, ok := v.([]map[string]interface{})
				return ok
			},
			typeDesc: "[]map[string]interface{}",
		},
		{
			name:  "preserve mixed []interface{}",
			input: []interface{}{"string", float64(1), true},
			checkFn: func(v interface{}) bool {
				_, ok := v.([]interface{})
				return ok
			},
			typeDesc: "[]interface{}",
		},
		{
			name:  "narrow nested maps",
			input: map[string]interface{}{"nested": map[string]interface{}{"value": float64(1)}},
			checkFn: func(v interface{}) bool {
				m, ok := v.(map[string]interface{})
				if !ok {
					return false
				}
				nested, ok := m["nested"].(map[string]interface{})
				if !ok {
					return false
				}
				_, ok = nested["value"].(float64)
				return ok
			},
			typeDesc: "map with nested map",
		},
		{
			name:  "convert int to float64",
			input: int(42),
			checkFn: func(v interface{}) bool {
				f, ok := v.(float64)
				return ok && f == 42.0
			},
			typeDesc: "float64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := narrowTypes(tt.input)

			if !tt.checkFn(result) {
				t.Errorf("narrowTypes() did not produce expected type %s, got %T",
					tt.typeDesc, result)
			}
		})
	}
}

// TestGetStructureType tests the type detection function
func TestGetStructureType(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"null", nil, "NULL"},
		{"string", "test", "STRING"},
		{"boolean", true, "BOOL"},
		{"float64", float64(42), "NUMBER"},
		{"int", int(42), "NUMBER"},
		{"int32", int32(42), "NUMBER"},
		{"int64", int64(42), "NUMBER"},
		{"uint", uint(42), "NUMBER"},
		{"row", map[string]interface{}{"id": 1}, "ROW"},
		{"table", []map[string]interface{}{{"id": 1}}, "TABLE"},
		{"array_string", []string{"a", "b"}, "ARRAY_OF_STRING"},
		{"array_number", []float64{1.0, 2.0}, "ARRAY_OF_NUMBER"},
		{"array_bool", []bool{true, false}, "ARRAY_OF_BOOL"},
		{"array_unknown", []interface{}{1, "a"}, "ARRAY_OF_UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStructureType(tt.value)
			if result != tt.expected {
				t.Errorf("getStructureType(%T) = %s, want %s",
					tt.value, result, tt.expected)
			}
		})
	}
}

// TestTypeConversionHelpers tests the type checking helper functions
func TestTypeConversionHelpers(t *testing.T) {
	t.Run("asFloat64", func(t *testing.T) {
		tests := []struct {
			name     string
			input    interface{}
			expected float64
			shouldOk bool
		}{
			{"float64", float64(42.5), 42.5, true},
			{"float32", float32(42.5), 42.5, true},
			{"int", int(42), 42.0, true},
			{"int64", int64(42), 42.0, true},
			{"uint", uint(42), 42.0, true},
			{"string", "not a number", 0, false},
			{"bool", true, 0, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, ok := asFloat64(tt.input)

				if ok != tt.shouldOk {
					t.Errorf("asFloat64(%T) ok = %v, want %v", tt.input, ok, tt.shouldOk)
				}

				if tt.shouldOk && result != tt.expected {
					t.Errorf("asFloat64(%v) = %v, want %v", tt.input, result, tt.expected)
				}
			})
		}
	})

	t.Run("allString", func(t *testing.T) {
		if !allString([]interface{}{"a", "b", "c"}) {
			t.Error("allString should return true for string slice")
		}
		if allString([]interface{}{"a", float64(1)}) {
			t.Error("allString should return false for mixed slice")
		}
		if !allString([]interface{}{}) {
			t.Error("allString should return true for empty slice")
		}
	})

	t.Run("allNumber", func(t *testing.T) {
		if !allNumber([]interface{}{float64(1), float64(2), float64(3)}) {
			t.Error("allNumber should return true for number slice")
		}
		if allNumber([]interface{}{float64(1), "string"}) {
			t.Error("allNumber should return false for mixed slice")
		}
		if !allNumber([]interface{}{}) {
			t.Error("allNumber should return true for empty slice")
		}
	})

	t.Run("allBool", func(t *testing.T) {
		if !allBool([]interface{}{true, false, true}) {
			t.Error("allBool should return true for bool slice")
		}
		if allBool([]interface{}{true, "string"}) {
			t.Error("allBool should return false for mixed slice")
		}
		if !allBool([]interface{}{}) {
			t.Error("allBool should return true for empty slice")
		}
	})

	t.Run("allMap", func(t *testing.T) {
		maps := []interface{}{
			map[string]interface{}{"id": 1},
			map[string]interface{}{"id": 2},
		}
		if !allMap(maps) {
			t.Error("allMap should return true for map slice")
		}
		if allMap([]interface{}{map[string]interface{}{}, "string"}) {
			t.Error("allMap should return false for mixed slice")
		}
		if !allMap([]interface{}{}) {
			t.Error("allMap should return true for empty slice")
		}
	})
}

// TestEvaluatorMemoryOperations tests memory get/set operations
func TestEvaluatorMemoryOperations(t *testing.T) {
	ev := &Evaluator{
		Memory: make(map[string]interface{}),
	}

	// Test setting multiple values
	testData := map[string]interface{}{
		"user":    map[string]interface{}{"id": float64(1), "name": "Alice"},
		"count":   float64(42),
		"active":  true,
		"tags":    []string{"tag1", "tag2"},
		"scores":  []float64{98.5, 87.3, 92.1},
		"records": []map[string]interface{}{{"id": float64(1)}, {"id": float64(2)}},
	}

	for key, value := range testData {
		ev.SetMemoryValue(key, value)
	}

	// Verify all values were stored
	for key := range testData {
		if _, exists := ev.Memory[key]; !exists {
			t.Errorf("Key %s not found in memory", key)
		}
		if _, exists := ev.Memory[key+"_meta_type"]; !exists {
			t.Errorf("Metadata for key %s not found in memory", key)
		}
	}

	// Verify memory contains expected number of entries (value + metadata for each)
	expectedEntries := len(testData) * 2 // value + metadata for each
	if len(ev.Memory) != expectedEntries {
		t.Errorf("Memory size = %d, want %d", len(ev.Memory), expectedEntries)
	}
}

// BenchmarkSetMemoryValue benchmarks the SetMemoryValue operation
func BenchmarkSetMemoryValue(b *testing.B) {
	ev := &Evaluator{
		Memory: make(map[string]interface{}),
	}

	testValues := []interface{}{
		"string value",
		float64(42),
		true,
		[]string{"a", "b", "c"},
		[]float64{1.0, 2.0, 3.0},
		map[string]interface{}{"id": float64(1), "name": "test"},
		[]map[string]interface{}{{"id": float64(1)}, {"id": float64(2)}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		value := testValues[i%len(testValues)]
		ev.SetMemoryValue("test_key", value)
	}
}

// BenchmarkNarrowTypes benchmarks the type narrowing operation
func BenchmarkNarrowTypes(b *testing.B) {
	testCases := []struct {
		name  string
		value interface{}
	}{
		{"strings", []interface{}{"a", "b", "c", "d", "e"}},
		{"numbers", []interface{}{float64(1), float64(2), float64(3), float64(4), float64(5)}},
		{"mixed", []interface{}{"string", float64(1), true, nil}},
		{"nested_map", map[string]interface{}{
			"level1": map[string]interface{}{
				"level2": map[string]interface{}{
					"value": float64(42),
				},
			},
		}},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = narrowTypes(tc.value)
			}
		})
	}
}
