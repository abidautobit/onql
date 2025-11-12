package database

import (
	"errors"
	"math"
	"onql/storemanager"
	"strconv"
)

func enforcePrecisionScale(value float64, precision, scale int) (float64, error) {
	pow := math.Pow(10, float64(scale))
	rounded := math.Round(value*pow) / pow

	if err := validatePrecisionScale(rounded, precision, scale); err != nil {
		return 0.0, err
	}

	return rounded, nil
}

func processNumber(val float64, colSchema map[string]string) (float64, error) {
	precisionStr, hasPrecision := colSchema["precision"]
	if !hasPrecision || precisionStr == "" {
		return val, nil
	}

	precision, _ := strconv.Atoi(precisionStr)

	scaleStr := colSchema["scale"]
	scale := 0
	if scaleStr != "" {
		scale, _ = strconv.Atoi(scaleStr)
	}

	return enforcePrecisionScale(val, precision, scale)
}

// Insert inserts a new record into the specified table.
// example usage:
// pk, err := database.Insert("mydb", "mytable", map[string]any{"name": "John", "age": 30}) or nested map for complex types
func Insert(db, table string, record map[string]any) (string, error) {
	// is db exists
	if !IsDatabaseExists(db) {
		return "", errors.New("database does not exist")
	}

	// is table exists
	if !IsTableExists(db, table) {
		return "", errors.New("table does not exist")
	}
	//format things
	// convert float64 to int64 for int fields (e.g., timestamp)
	schema := FullSchema[db][table]

	//set default values on empty cols
	for col, colSchema := range schema {
		if record[col] == nil {
			if defaultVal, ok := colSchema["default"]; ok {
				record[col] = defaultVal
			} else if colSchema["blank"] == "no" {
				return "", errors.New("missing required field: " + col)
			} else {
				record[col] = ""
			}
		}
	}

	for col, colSchema := range schema {
		if colSchema["type"] == "timestamp" || colSchema["type"] == "number" {
			if v, ok := record[col]; ok {
				if f, ok := v.(float64); ok {
					if colSchema["type"] == "timestamp" {
						record[col] = int64(f)
					} else if colSchema["type"] == "number" {
						val, err := processNumber(f, colSchema)
						if err != nil {
							return "", err
						}
						record[col] = val
					}
				} else {
					if f, ok := v.(string); ok {
						if colSchema["type"] == "timestamp" {
							val, err := strconv.ParseInt(f, 10, 64)
							if err != nil {
								return "", err
							}
							record[col] = val
						} else if colSchema["type"] == "number" {
							val, err := strconv.ParseFloat(f, 64)
							if err != nil {
								return "", err
							}

							val, err = processNumber(val, colSchema)
							if err != nil {
								return "", err
							}
							record[col] = val
						}
					}
				}
			}
		}
	}
	// Set the ID field before validate required columns
	record["id"] = ""
	// is record valid
	if err := ValidateRecord(db, table, record); err != nil {
		return "", err
	}

	// insert record
	insertData := storemanager.NewData(
		db,
		table,
		record,
		FullSchemaStorages[db][table],
		FullSchemaTypes[db][table],
	)

	pk, err := storemanager.InsertData(insertData)
	if err != nil {
		return "", err
	}

	return pk, nil
}
