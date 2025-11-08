package get

import (
	"errors"
	"fmt"
	"onql/database"
	"onql/storemanager"
)

// get with pks
func GetWithPKs(db, table string, pks []string) ([]map[string]any, error) {
	// Check if database exists
	if !database.IsDatabaseExists(db) {
		return nil, errors.New("database does not exist")
	}
	// Check if table exists
	if !database.IsTableExists(db, table) {
		return nil, errors.New("table does not exist")
	}
	// Get data from storemanager
	// if len(columns) > 0 {
	// 	return storemanager.GetSpecificData(db, table, pks, columns)
	// }
	return storemanager.GetData(db, table, pks)
}

// get pks with index
// index string is col level index here
func GetPksFromIndex(db, table, index string) ([]string, error) {
	// Check if database exists
	if !database.IsDatabaseExists(db) {
		return nil, errors.New("database does not exist")
	}
	// Check if table exists
	if !database.IsTableExists(db, table) {
		return nil, errors.New("table does not exist")
	}
	indexKey := fmt.Sprintf("index:%s:%s:%s", db, table, index)
	return storemanager.GetPksFromIndex(indexKey), nil
}

// for fast access only to ram data
// func GetRamData(db, table string, pks []string) ([]map[string]any, error) {
// 	if !database.IsDatabaseExists(db) {
// 		return nil, errors.New("database does not exist")
// 	}
// 	if !database.IsTableExists(db, table) {
// 		return nil, errors.New("table does not exist")
// 	}

// 	// schema := database.FullSchemaStorages[db][table] // <-- Use pre-flattened storage schema

// 	var results []map[string]any
// 	for _, pk := range pks {
// 		row, err := storemanager.GetRamData(db, table, pk, []string{})
// 		if err != nil {
// 			return nil, err
// 		}
// 		results = append(results, row)
// 	}
// 	return results, nil
// }

func GetAllPks(db, table string) ([]string, error) {
	// Check if database exists
	if !database.IsDatabaseExists(db) {
		return nil, errors.New("database does not exist")
	}
	// Check if table exists
	if !database.IsTableExists(db, table) {
		return nil, errors.New("table does not exist")
	}
	pks, err := storemanager.GetAllPks(db, table)
	if err != nil {
		return nil, err
	}
	return pks, nil
}
