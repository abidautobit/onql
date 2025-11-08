package database

import (
	"errors"
	"fmt"
	"onql/storemanager"
	"strconv"
)

// Update updates an existing record in the specified table.
// The record must include the "id" field to identify the row to update.
// Example usage:
// err := database.Update("mydb", "mytable", map[string]any{"id": "123", "name": "John", "age": 30})
// This will update the record with id "123" in "mytable" of "mydb" database.
// The record can include any fields defined in the table schema.
// it update with id field
func Update(db, table string, record map[string]any) error {
	// Check if database exists
	if !IsDatabaseExists(db) {
		return errors.New("database does not exist")
	}
	// Check if table exists
	if !IsTableExists(db, table) {
		return errors.New("table does not exist")
	}
	// Ensure the record has "id"
	id, ok := record["id"]
	if !ok || id == "" {
		return errors.New("missing or empty 'id' field in record")
	}

	//set default values on empty cols
	for col, colSchema := range FullSchema[db][table] {
		if record[col] == nil {
			if defaultVal, ok := colSchema["default"]; ok {
				record[col] = defaultVal
			} else if colSchema["blank"] == "no" {
				return errors.New("missing required field: " + col)
			} else {
				record[col] = ""
			}
		}
	}

	for col, colSchema := range FullSchema[db][table] {
		if colSchema["type"] == "timestamp" || colSchema["type"] == "number" {
			if f, ok := record[col].(string); ok {
				if colSchema["type"] == "timestamp" {
					val, err := strconv.ParseInt(f, 10, 64)
					if err != nil {
						return err
					}
					record[col] = val
				} else if colSchema["type"] == "number" {
					val, err := strconv.ParseFloat(f, 64)
					if err != nil {
						return err
					}
					record[col] = val
				}
			} else {
				if colSchema["type"] == "timestamp" {
					switch v := record[col].(type) {
					case float64:
						record[col] = int64(v)
					}
				}
			}
		}
	}

	// Validate the record against schema
	if err := ValidateRecord(db, table, record); err != nil {
		return err
	}

	// Construct Data object
	updateData := storemanager.NewData(
		db,
		table,
		record,
		FullSchemaStorages[db][table],
		FullSchemaTypes[db][table],
	)
	fmt.Println("Updating record with ID:", id, "in columns:", record["name"])
	// Perform update
	if err := storemanager.UpdateData(updateData); err != nil {
		return err
	}
	return nil
}

// UpdatePartial updates a subset of fields in an existing record.
// The record must include the "id" field to identify the row to update.
// Example usage:
// err := database.UpdatePartial("mydb", "mytable", map[string]any{"id": "123", "name": "John"})
func UpdatePartial(db, table string, record map[string]any) error {
	// Check if database exists
	if !IsDatabaseExists(db) {
		return errors.New("database does not exist")
	}
	// Check if table exists
	if !IsTableExists(db, table) {
		return errors.New("table does not exist")
	}
	// Ensure "id" is present
	idRaw, ok := record["id"]
	if !ok || idRaw == "" {
		return errors.New("missing or empty 'id' field in record")
	}
	id := idRaw.(string)

	for col, colSchema := range FullSchema[db][table] {
		if record[col] == nil {
			continue
		}
		if colSchema["type"] == "timestamp" || colSchema["type"] == "number" {
			if f, ok := record[col].(string); ok {
				if colSchema["type"] == "timestamp" {
					val, err := strconv.ParseInt(f, 10, 64)
					if err != nil {
						return err
					}
					record[col] = val
				} else if colSchema["type"] == "number" {
					val, err := strconv.ParseFloat(f, 64)
					if err != nil {
						return err
					}
					record[col] = val
				}
			} else {
				if colSchema["type"] == "timestamp" {
					switch v := record[col].(type) {
					case float64:
						record[col] = int64(v)
					}
				}
			}
		}
	}

	// Validate partial record
	if err := ValidatePartialRecord(db, table, record); err != nil {
		return err
	}

	// For disk or mixed, load full data, merge and update
	existingRows, err := storemanager.GetData(db, table, []string{id})
	if err != nil {
		return err
	}
	if len(existingRows) == 0 {
		return errors.New("record not found")
	}
	merged := existingRows[0]
	for k, v := range record {
		merged[k] = v
	}
	// Construct Data and update
	updateData := storemanager.NewData(
		db,
		table,
		merged,
		FullSchemaStorages[db][table],
		FullSchemaTypes[db][table],
	)
	return storemanager.UpdateData(updateData)
}
