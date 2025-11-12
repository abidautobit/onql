package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"onql/storemanager"
	"slices"
	"strconv"
	"strings"
)

// ValidateRecord validates a record against the schema of a specified table.
func ValidateRecord(db, table string, record map[string]any) error {
	// Get the table's schema
	schema := FullSchema[db][table]
	if schema == nil {
		return fmt.Errorf("table '%s' does not exist in database '%s'", table, db)
	}
	// Validate the record against the schema
	return validateRecord(record, schema)
}

func validateRecord(record map[string]any, schema map[string]map[string]string) error {
	for col, colSchema := range schema {
		if val, ok := record[col]; ok {
			if err := validateValue(val, colSchema); err != nil {
				return fmt.Errorf("invalid value for column '%s': %v", col, err)
			}
		} else {
			if colSchema["blank"] == "no" {
				return fmt.Errorf("missing required column '%s'", col)
			}
		}
	}
	for col := range record {
		if _, ok := schema[col]; !ok {
			return fmt.Errorf("column '%s' does not exist in schema", col)
		}
	}
	return nil
}

// ValidatePartialRecord validates only the provided fields in the record
// without requiring all fields to be present.
func ValidatePartialRecord(db, table string, record map[string]any) error {
	// Get the table's schema
	schema := FullSchema[db][table]
	if schema == nil {
		return fmt.Errorf("table '%s' does not exist in database '%s'", table, db)
	}
	return validatePartialRecord(record, schema)
}

func validatePartialRecord(record map[string]any, schema map[string]map[string]string) error {
	for col, val := range record {
		colSchema, exists := schema[col]
		if !exists {
			return fmt.Errorf("column '%s' does not exist in schema", col)
		}
		if err := validateValue(val, colSchema); err != nil {
			return fmt.Errorf("invalid value for column '%s': %v", col, err)
		}
	}
	return nil
}

func validatePrecisionScale(value float64, precision, scale int) error {
	valueStr := strconv.FormatFloat(value, 'f', -1, 64)

	parts := strings.Split(valueStr, ".")
	integerPart := parts[0]
	fractionalPart := ""
	if len(parts) > 1 {
		fractionalPart = parts[1]
	}

	// Remove minus sign
	integerPart = strings.ReplaceAll(integerPart, "-", "")

	totalDigits := len(integerPart) + len(fractionalPart)
	if totalDigits > precision {
		return fmt.Errorf("value exceeds precision (%d digits, max %d)", totalDigits, precision)
	}

	decimalPlaces := len(fractionalPart)
	if decimalPlaces > scale {
		return fmt.Errorf("value exceeds scale (%d decimal places, max %d)", decimalPlaces, scale)
	}

	return nil
}

func validateValue(value any, colSchema map[string]string) error {
	colType := colSchema["type"]

	switch colType {
	case "string", "timestamp":
	// Nothing to do in this case

	case "number":
		switch v := value.(type) {
		case float64:
			precisionStr, hasPrecision := colSchema["precision"]
			if !hasPrecision || precisionStr == "" {
				return nil
			}
			precision, _ := strconv.Atoi(precisionStr)

			scaleStr := colSchema["scale"]
			scale := 0
			if scaleStr != "" {
				scale, _ = strconv.Atoi(scaleStr)
			}

			if err := validatePrecisionScale(v, precision, scale); err != nil {
				return err
			}

		default:
			return fmt.Errorf("expected number, got %T", value)
		}

	case "json":
		// JSON should be stored as string (serialized JSON)
		switch v := value.(type) {
		case string:
			// Validate JSON
			var temp any
			if err := json.Unmarshal([]byte(v), &temp); err != nil {
				return fmt.Errorf("expected valid JSON string, got: %s", v)
			}
		case map[string]any, []any:
			// Valid JSON object/array - will be serialized to string
		default:
			return fmt.Errorf("expected JSON, got %T", value)
		}

	default:
		return fmt.Errorf("unknown column type '%s'", colType)
	}

	return nil
}

// is database exists
func IsDatabaseExists(db string) bool {
	_, exists := FullSchema[db]
	return exists
}

// IsTableExists checks if a table exists in the specified database.
func IsTableExists(db, table string) bool {
	if !IsDatabaseExists(db) {
		return false
	}
	_, exists := FullSchema[db][table]
	return exists
}

// IsColumnExists checks if a column exists in the specified table of a database.
func IsColumnExists(db, table, column string) bool {
	if !IsTableExists(db, table) {
		return false
	}
	_, exists := FullSchema[db][table][column]
	return exists
}

// for schemas Alter table
func ValidateColumnAlteration(db, table, column string, newSchema map[string]string) error {
	// Check if database exists
	if !IsDatabaseExists(db) {
		return errors.New("database does not exist")
	}
	// Check if table exists
	if !IsTableExists(db, table) {
		return errors.New("table does not exist")
	}
	// Check if column exists
	if !IsColumnExists(db, table, column) {
		return errors.New("column does not exist")
	}
	// Validate the new schema
	return validateColumnSchema(newSchema)
}

func validateColumnSchema(schema map[string]string) error {
	// Check for required fields
	requiredFields := []string{"type", "storage", "default", "blank"}
	for _, field := range requiredFields {
		if _, ok := schema[field]; !ok {
			return fmt.Errorf("missing required field '%s'", field)
		}
	}
	// Validate field values
	if err := validateFieldValue("type", schema["type"], []string{"string", "number", "timestamp", "json"}); err != nil {
		return err
	}
	if err := validateFieldValue("storage", schema["storage"], []string{"disk", "ram"}); err != nil {
		return err
	}
	if err := validateFieldValue("blank", schema["blank"], []string{"yes", "no"}); err != nil {
		return err
	}
	if schema["default"] != "" {
		if err := validateValue(schema["default"], schema); err != nil {
			return err
		}
	}
	return nil
}

func validateFieldValue(fieldName, value string, validValues []string) error {
	if slices.Contains(validValues, value) {
		return nil
	}

	return fmt.Errorf("invalid value for field '%s': %s", fieldName, value)
}

// Protocol
func ValidateProtocol(protocol storemanager.QueryProtocol) error {
	for modName, mod := range protocol {
		if modName == "" {
			return errors.New("db alias name is required")
		}
		if err := ValidateModule(modName, mod); err != nil {
			return fmt.Errorf("module '%s': %w", mod.Database, err)
		}
	}
	return nil
}

func ValidateModule(modName string, mod *storemanager.Module) error {
	if mod.Database == "" {
		return errors.New("missing database")
	}
	if !IsDatabaseExists(mod.Database) {
		return errors.New("database does not exist")
	}
	for entityName, entity := range mod.Entities {
		if entityName == "" {
			return errors.New("table alias name is required")
		}
		if err := ValidateEntity(entityName, entity, mod.Database); err != nil {
			return fmt.Errorf("entity '%s': %w", entity.Table, err)
		}
	}
	return nil
}

func ValidateEntity(entityName string, entity *storemanager.Entity, database string) error {
	if entity.Table == "" {
		return errors.New("missing table")
	}
	if !IsTableExists(database, entity.Table) {
		return errors.New("table does not exist")
	}
	for fieldName, field := range entity.Fields {
		if fieldName == "" {
			return errors.New("field name is required")
		}
		if err := ValidateField(database, entity.Table, field); err != nil {
			return fmt.Errorf("field '%s': %w", fieldName, err)
		}
	}
	//validate relations
	for relationName, relation := range entity.Relations {
		if relationName == "" {
			return errors.New("relation name is required")
		}
		if err := ValidateRelation(database, entity.Table, relation); err != nil {
			return fmt.Errorf("relation '%s': %w", relationName, err)
		}
	}
	//validate contexts
	for contextName, context := range entity.Context {
		if contextName == "" {
			return errors.New("context name is required")
		}
		if err := ValidateContext(database, entity.Table, contextName, context); err != nil {
			return fmt.Errorf("context '%s': %w", contextName, err)
		}
	}
	return nil
}

func ValidateField(database string, table string, field string) error {
	if field == "" {
		return errors.New("missing type")
	}
	if !IsColumnExists(database, table, field) {
		return errors.New("column does not exist")
	}
	return nil
}

func ValidateRelation(database string, parentTable string, relation *storemanager.Relation) error {
	if relation.Type == "" {
		return errors.New("missing relation type")
	}
	if !IsTableExists(database, relation.Entity) {
		return errors.New("missing target entity")
	}
	if relation.Type == "mtm" && !IsTableExists(database, relation.Through) {
		return errors.New("missing join table")
	}
	if relation.FKField == "" {
		return errors.New("missing foreign key field")
	}
	rfk := strings.Split(relation.FKField, ":")
	// validate foreign key field
	switch relation.Type {
	case "mto", "otm", "oto":
		if len(rfk) != 2 {
			return errors.New("invalid foreign key field")
		}
		if !IsColumnExists(database, parentTable, rfk[0]) || !IsColumnExists(database, relation.Entity, rfk[1]) {
			return errors.New("invalid foreign key field")
		}
	case "mtm":
		if len(rfk) != 4 {
			return errors.New("invalid foreign key field")
		}
		if !IsColumnExists(database, parentTable, rfk[0]) || !IsColumnExists(database, relation.Through, rfk[1]) || !IsColumnExists(database, relation.Through, rfk[2]) || !IsColumnExists(database, relation.Entity, rfk[3]) {
			return errors.New("invalid foreign key field")
		}
	default:
		return errors.New("invalid relation type")
	}
	return nil
}

func ValidateContext(database string, table string, contextName string, contextValue string) error {
	// if contextName == "" {
	// 	return errors.New("missing context name")
	// }
	// if contextValue == "" {
	// 	return errors.New("missing context value")
	// }
	// if !IsColumnExists(database, table, contextName) {
	// 	return errors.New("context column does not exist")
	// }
	return nil
}
