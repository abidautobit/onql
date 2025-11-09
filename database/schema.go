package database

import (
	"errors"
	"fmt"
	storemanager "onql/storemanager"
	"strconv"
	"strings"
)

// Return list of databases names
// Example usage:
//
//	databases, err := database.GetDatabases()
func GetDatabases() ([]string, error) {
	return storemanager.GetDatabases()
}

// Return list of tables names in a database
// Example usage:
//
//	tables, err := database.GetTables("mydb")
func GetTables(db string) ([]string, error) {
	return storemanager.GetTables(db)
}

// Return columns mean schema of a table
// It returns a map structure:
// map[column][attribute]value
func GetTableSchema(db, table string) (map[string]map[string]string, error) {
	return storemanager.GetTableSchema(db, table)
}

// GetFullSchema retrieves the full schema of all databases and their tables.
// It returns a nested map structure:
// map[database][table][column][attribute]value
func GetFullSchema() (map[string]map[string]map[string]map[string]string, error) {
	databases, err := storemanager.GetDatabases()
	if err != nil {
		return nil, err
	}
	fullSchema := make(map[string]map[string]map[string]map[string]string)
	for _, db := range databases {
		tables, err := storemanager.GetTables(db)
		if err != nil {
			return nil, err
		}
		fullSchema[db] = make(map[string]map[string]map[string]string)
		for _, table := range tables {
			schema, err := storemanager.GetTableSchema(db, table)
			if err != nil {
				return nil, err
			}
			fullSchema[db][table] = schema
		}
	}
	return fullSchema, nil
}

// CreateDatabase creates a new database with the given name.
// Example usage:
//
//	err := database.CreateDatabase("mydb")
func CreateDatabase(name string) error {
	// validate if already exists
	if FullSchema[name] != nil {
		return fmt.Errorf("database %s already exists", name)
	}
	err := storemanager.CreateDatabase(name)
	if err == nil {
		// update in-memory schema
		schema, _ := GetFullSchema()
		SetFullSchema(schema)
	}

	return err
}

// TODO 2025-11-09: Break validation and creation of table in different function.
// TODO 2025-11-09: After breaking; write test cases for validation.

// Example usage:
//
//	err := database.CreateTable("mydb", "mytable", map[string]map[string]string{
//		"name": {
//			"type":    "string",
//			"storage": "ram",
//			"default": "",
//			"blank":   "no",
//		},
//		"age": {
//			"type":    "number",
//			"storage": "disk",
//			"default": "0",
//			"blank":   "no",
//		},
//	})
func CreateTable(db, table string, schema map[string]map[string]string) error {
	// validate if database exists
	if FullSchema[db] == nil {
		return fmt.Errorf("database %s does not exist", db)
	}

	// validate if table already exists
	if FullSchema[db][table] != nil {
		return fmt.Errorf("table %s already exists", table)
	}

	// valid sets
	validTypes := map[string]bool{
		"string":    true,
		"timestamp": true,
		"number":    true,
		"json":      true,
	}
	validStorage := map[string]bool{
		"disk": true,
		"ram":  true,
	}
	validBlank := map[string]bool{
		"yes": true,
		"no":  true,
	}

	// validate schema
	for col, colSchema := range schema {
		if strings.TrimSpace(col) == "" {
			return errors.New("column name cannot be empty")
		}
		if strings.Contains(col, ":") {
			return fmt.Errorf("column name '%s' cannot contain ':' character", col)
		}
		if colSchema == nil {
			return fmt.Errorf("column '%s' has no schema defined", col)
		}

		// check "type"
		colType := strings.ToLower(colSchema["type"])
		if colType == "" || !validTypes[colType] {
			return fmt.Errorf("column '%s' has invalid or missing 'type'", col)
		}

		// check "storage"
		storage := strings.ToLower(colSchema["storage"])
		if storage == "" || !validStorage[storage] {
			return fmt.Errorf("column '%s' must define 'storage' as 'disk' or 'ram'", col)
		}

		// check "default" (can be empty, so just ensure key exists)
		if _, ok := colSchema["default"]; !ok {
			return fmt.Errorf("column '%s' must define 'default'", col)
		}

		// check "precision" and "scale" for "number" type
		if colType == "number" {
			if precisionStr, hasPrecision := colSchema["precision"]; hasPrecision && precisionStr != "" {
				precision, err := strconv.Atoi(precisionStr)
				if err != nil || precision <= 0 {
					return fmt.Errorf("column '%s' has invalid precision value (must be positive integer)", col)
				}

				if scaleStr, hasScale := colSchema["scale"]; hasScale && scaleStr != "" {
					scale, err := strconv.Atoi(scaleStr)
					if err != nil || scale < 0 {
						return fmt.Errorf("column '%s' has invalid scale value (must be non-negative integer)", col)
					}

					if scale > precision {
						return fmt.Errorf("column '%s' has scale (%d) greater than precision (%d)", col, scale, precision)
					}
				}
			} else if _, hasScale := colSchema["scale"]; hasScale {
				return fmt.Errorf("column '%s' has scale specified without precision", col)
			}
		}

		if colSchema["default"] != "" {
			var err error
			var val any = colSchema["default"]

			switch colSchema["type"] {
			case "number":
				if val, err = strconv.ParseFloat(colSchema["default"], 64); err != nil {
					return fmt.Errorf("column '%s' has invalid default value", col)
				}
			case "timestamp":
				if val, err = strconv.ParseInt(colSchema["default"], 10, 64); err != nil {
					return fmt.Errorf("column '%s' has invalid default value", col)
				}
			}

			if err = validateValue(val, colSchema); err != nil {
				return err
			}
		}

		// check "blank"
		blank := strings.ToLower(colSchema["blank"])
		if blank == "" || !validBlank[blank] {
			return fmt.Errorf("column '%s' must define 'blank' as 'yes' or 'no'", col)
		}
	}

	// insert id in colschema here
	schema["id"] = map[string]string{
		"type":    "string",
		"storage": "disk",
		"default": "",
		"blank":   "no",
	}

	// passed all validation
	err := storemanager.CreateTable(db, table, schema)
	if err == nil {
		// update in-memory schema
		schema, _ := GetFullSchema()
		SetFullSchema(schema)
	}
	return err
}

func RenameDatabase(oldName, newName string) error {
	// Check if the old database exists
	if FullSchema[oldName] == nil {
		return fmt.Errorf("database %s does not exist", oldName)
	}
	// Check if the new database name is valid
	if FullSchema[newName] != nil {
		return fmt.Errorf("database %s already exists", newName)
	}
	// Update the store manager
	err := storemanager.RenameDatabase(oldName, newName)
	if err != nil {
		return err
	}
	schema, _ := GetFullSchema()
	SetFullSchema(schema)
	return nil
}

func RenameTable(db, oldName, newName string) error {
	if !IsDatabaseExists(db) {
		return fmt.Errorf("database %s does not exist", db)
	}
	if !IsTableExists(db, oldName) {
		return fmt.Errorf("table %s does not exist", oldName)
	}
	if IsTableExists(db, newName) {
		return fmt.Errorf("table %s already exists", newName)
	}
	// Update the store manager
	err := storemanager.RenameTable(db, oldName, newName)
	if err != nil {
		return err
	}
	schema, _ := GetFullSchema()
	SetFullSchema(schema)
	return nil
}

func AlterTable(db, table string, alters map[string]map[string]string) error {
	// Check if database exists
	if !IsDatabaseExists(db) {
		return fmt.Errorf("database %s does not exist", db)
	}
	// Check if table exists
	if !IsTableExists(db, table) {
		return fmt.Errorf("table %s does not exist", table)
	}

	schema := FullSchema[db][table]

	for action, details := range alters {
		if details["name"] == "id" {
			return fmt.Errorf("column %s cannot be altered", details["name"])
		}
		switch action {
		case "changeBlank", "changeDefault":
			colName := details["name"]

			if colSchema, exists := schema[colName]; exists {
				if details["newDefault"] != "" {
					colSchema["default"] = details["newDefault"]
				}
				if details["newBlank"] != "" {
					colSchema["blank"] = details["newBlank"]
				}
				schema[colName] = colSchema
			}
		case "renameColumn":
			oldName := details["oldName"]
			newName := details["newName"]

			if colSchema, exists := schema[oldName]; exists {
				schema[newName] = colSchema
				delete(schema, oldName)
			} else {
				return fmt.Errorf("column %s does not exist", oldName)
			}
		case "addColumn":
			newCol := details["name"]

			schema[newCol] = map[string]string{
				"type":    details["type"],
				"storage": details["storage"],
				"default": details["default"],
				"blank":   details["blank"],
			}
		case "dropColumn":
			delete(schema, details["name"])
		default:
			return fmt.Errorf("unknown alter action: %s", action)
		}
	}
	// Validate and apply each alteration
	for col, colSchema := range schema {
		if err := ValidateColumnAlteration(db, table, col, colSchema); err != nil {
			return err
		}
	}
	// Update the store manager
	err := storemanager.AlterTable(db, table, alters)
	if err != nil {
		return err
	}
	fullSchema, _ := GetFullSchema()
	SetFullSchema(fullSchema)
	return nil
}

func DeleteDatabase(db string) error {
	// Check if the database exists
	if !IsDatabaseExists(db) {
		return fmt.Errorf("database %s does not exist", db)
	}
	// Update the store manager
	err := storemanager.DeleteDatabase(db)
	if err != nil {
		return err
	}
	schema, _ := GetFullSchema()
	SetFullSchema(schema)
	return nil
}

func DeleteTable(db, table string) error {
	// Check if the database exists
	if !IsDatabaseExists(db) {
		return fmt.Errorf("database %s does not exist", db)
	}
	// Check if the table exists
	if !IsTableExists(db, table) {
		return fmt.Errorf("table %s does not exist", table)
	}
	// Update the store manager
	err := storemanager.DeleteTable(db, table)
	if err != nil {
		return err
	}
	schema, _ := GetFullSchema()
	SetFullSchema(schema)
	return nil
}

func RefereshIndexes() error {
	return storemanager.RefereshIndex()
}
