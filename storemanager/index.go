package storemanager

import (
	"encoding/json"
	"fmt"
	"onql/engine"
	"onql/utils"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"
)

// take object
func SaveIndex(data Data, prevData Data) error {
	// fmt.Println("Saving index for data:", data.columns["id"], "with name:", data.columns["name"])
	pk, ok := data.columns["id"].(string)
	if !ok || pk == "" {
		return fmt.Errorf("invalid or missing primary key in data")
	}

	// Step 1: Remove old index entries from previous data
	if len(prevData.columns) > 0 {
		oldIndexes := GenIndex(prevData)
		for _, oldIndex := range oldIndexes {
			 AddToDeleteBuffer(oldIndex, pk)
		}
	}

	// Step 2: Add new index entries from current data
	newIndexes := GenIndex(data)
	for _, newIndex := range newIndexes {
		// fmt.Println("Adding index:", newIndex, "for PK:", pk)
		 AddToIndexBuffer(newIndex, pk)
	}

	return nil
}

// DeleteIndex removes all index entries for the given data.
func DeleteIndex(data Data) error {
	pk, ok := data.columns["id"].(string)
	if !ok || pk == "" {
		return fmt.Errorf("invalid or missing primary key in data for deletion")
	}

	indexKeys := GenIndex(data)
	for _, indexKey := range indexKeys {
		 AddToDeleteBuffer(indexKey, pk) // Use goroutine
	}
	return nil
}

// GenIndex generates unique index keys from data columns for fast lookups.
//   - For non-JSON columns: "index:db:table:column:value"
//   - For JSON columns (map[string]interface{}): recursively flattens keys and generates:
//     "index:db:table:column:nestedKey:nestedKey2:value"
//   - Avoids duplicate index entries.
func GenIndex(data Data) []string {
	var indexes []string
	seen := make(map[string]bool)

	// Recursive helper to handle nested JSON structures
	var collect func(baseKey, nestedKey string, val interface{})
	collect = func(baseKey, nestedKey string, val interface{}) {
		switch v := val.(type) {
		case map[string]interface{}:
			// Recursively flatten JSON keys
			for k, subv := range v {
				newKey := k
				if nestedKey != "" {
					newKey = nestedKey + ":" + k
				}
				collect(baseKey, newKey, subv)
			}
		default:
			// Final index key with flattened path and value
			index := baseKey
			if nestedKey != "" {
				index += ":" + nestedKey
			}
			index += ":" + utils.ToString(v)
			if !seen[index] {
				indexes = append(indexes, index)
				seen[index] = true
			}
		}
	}

	for col, val := range data.columns {
		base := "index:" + data.db + ":" + data.table + ":" + col
		if data.datatype[col] != "json" {
			// Non-JSON columns
			index := base + ":" + utils.ToString(val)
			if !seen[index] {
				indexes = append(indexes, index)
				seen[index] = true
			}
		} else {
			// Flatten JSON columns recursively
			collect(base, "", val)
		}
	}

	return indexes
}

// GetIndexFileName determines the file bucket (like a-c.db, d-f.db) based on the first letter/number of the value.
// This helps split large index datasets across multiple files.
func GetIndexFileName(indexKey string) string {
	parts := strings.Split(indexKey, ":")
	if len(parts) < 2 {
		return "_etc.db"
	}

	value := parts[len(parts)-1]
	if len(value) == 0 {
		return "_etc.db"
	}

	firstChar := unicode.ToLower(rune(value[0]))

	switch {
	case firstChar >= 'a' && firstChar <= 'c':
		return "a-c.db"
	case firstChar >= 'd' && firstChar <= 'f':
		return "d-f.db"
	case firstChar >= 'g' && firstChar <= 'i':
		return "g-i.db"
	case firstChar >= 'j' && firstChar <= 'l':
		return "j-l.db"
	case firstChar >= 'm' && firstChar <= 'o':
		return "m-o.db"
	case firstChar >= 'p' && firstChar <= 'r':
		return "p-r.db"
	case firstChar >= 's' && firstChar <= 'u':
		return "s-u.db"
	case firstChar >= 'v' && firstChar <= 'z':
		return "v-z.db"
	case firstChar >= '0' && firstChar <= '9':
		return "0-9.db"
	default:
		return "_etc.db"
	}
}

func GetPksFromIndex(indexKey string) []string {
	spits := strings.Split(indexKey, ":")
	db := spits[1]
	table := spits[2]
	col := spits[3]

	diskStore, err := GetDiskStore(db + "/" + table + "/index/" + col + "/" + GetIndexFileName(indexKey))
	if err != nil {
		return nil
	}
	raw, err := diskStore.Get(indexKey)
	if err != nil {
		return nil
	}
	var raw2 []string
	err = json.Unmarshal([]byte(raw), &raw2)
	if err != nil {
		return nil
	}
	return raw2
}

// chen schema changed
func AlterIndex(da DataAlter) error {
	// Step 1: Identify index directory for the table
	indexDir := filepath.Join("store", da.db, da.table, "index")

	// Step 2: Close all open BadgerDBs related to this index directory
	for path, store := range diskStore {
		if strings.HasPrefix(path, indexDir) {
			_ = store.Close()       // Close DB instance
			delete(diskStore, path) // Remove from global store map
		}
	}

	// Step 3: Delete index directory from disk
	if err := os.RemoveAll(indexDir); err != nil {
		return fmt.Errorf("failed to remove index directory: %v", err)
	}

	// Step 4: Fetch latest schema
	newSchema, err := GetTableSchema(da.db, da.table)
	if err != nil {
		return err
	}

	// Step 5: Rebuild indexes
	tableStores := getTableDataStores(da.db, da.table)
	for _, store := range tableStores {
		pks := getAllKeysFromStore(store, da.db, da.table)
		dataRows, _ := GetData(da.db, da.table, pks)
		for _, row := range dataRows {
			data := Data{
				db:       da.db,
				table:    da.table,
				columns:  row,
				datatype: map[string]string{},
			}
			for col, meta := range newSchema {
				data.datatype[col] = meta["type"]
			}
			_ = SaveIndex(data, Data{}) // Force refresh without needing old index
		}
	}

	return nil
}

func RefereshIndex() error {
	schema := FullSchema
	for db, tables := range schema {
		for table, _ := range tables {
			// Step 1: Identify index directory for the table
			indexDir := filepath.Join("store", db, table, "index")

			// Step 2: Close all open BadgerDBs related to this index directory
			for path, store := range diskStore {
				if strings.HasPrefix(path, indexDir) {
					_ = store.Close()       // Close DB instance
					delete(diskStore, path) // Remove from global store map
				}
			}

			// Step 3: Delete index directory from disk
			if err := os.RemoveAll(indexDir); err != nil {
				return fmt.Errorf("failed to remove index directory: %v", err)
			}

			// Step 4: Fetch latest schema
			newSchema, err := GetTableSchema(db, table)
			if err != nil {
				return err
			}

			// Step 5: Rebuild indexes
			tableStores := getTableDataStores(db, table)
			for _, store := range tableStores {
				pks := getAllKeysFromStore(store, db, table)
				dataRows, _ := GetData(db, table, pks)
				for _, row := range dataRows {
					data := Data{
						db:       db,
						table:    table,
						columns:  row,
						datatype: map[string]string{},
					}
					for col, meta := range newSchema {
						data.datatype[col] = meta["type"]
					}
					_ = SaveIndex(data, Data{}) // Force refresh without needing old index
				}
			}

		}
	}
	return nil
}

// use parallel for view all index files oriented to column
func FilterPksByIndex(db, table, column, op, valType string, value interface{}) []string {
	prefix := fmt.Sprintf("index:%s:%s:%s:", db, table, column)
	valueStr := utils.ToString(value)

	stores := getIndexStores(db, table, column)
	pkChan := make(chan []string, len(stores))
	// fmt.Printf("Filtering with prefix: %s, op: %s, valType: %s, value: %s\n", prefix, op, valType, valueStr)
	var wg sync.WaitGroup
	for _, store := range stores {
		wg.Add(1)
		go func(s *engine.BadgerDB) {
			defer wg.Done()
			rawList, err := s.FilterByPrefixWithOp(prefix, op, valType, valueStr)
			if err != nil {
				pkChan <- nil // always send to prevent blocking
				return
			}

			var all []string
			for _, raw := range rawList {
				var ids []string
				if err := json.Unmarshal([]byte(raw), &ids); err == nil {
					all = append(all, ids...)
				}
			}
			pkChan <- all // send even if all is empty
		}(store)
	}

	go func() {
		wg.Wait()
		close(pkChan)
	}()

	var final []string
	for part := range pkChan {
		if part != nil {
			final = append(final, part...)
		}
	}

	return final
}

// used when db or table renamed
func AlterIndexKeys(db string, tables []string, new map[string]string) error {
	//if tables empty mean db change other wise table change
	if len(tables) == 0 {
		// Change all tables in the database
		tables, _ = GetTables(db)
	}
	newDb := new[db+"_db"]
	if newDb == "" {
		newDb = db
	}
	// Get all keys from the store
	for _, table := range tables {
		newTable := new[table+"_table"]
		if newTable == "" {
			newTable = table
		}
		schema, err := GetTableSchema(db, table)
		if err != nil {
			return err
		}
		for col, _ := range schema {
			// Get all index stores for the table
			columnStores := getIndexStores(db, table, col)
			for _, store := range columnStores {
				keys := getAllKeysFromIndexStore(store, db, table)
				for _, key := range keys {
					storeKey := "index:" + db + ":" + table + ":" + key
					row, _ := store.Get(storeKey)
					store.Delete(storeKey) // delete old key
					// keySplit := strings.Split(key, ":")
					// keySplit[1] = newDb    // update db
					// keySplit[2] = newTable // update table
					newStoreKey := "index:" + newDb + ":" + newTable + ":" + key
					jsonBytes, _ := json.Marshal(row)
					store.Set(newStoreKey, jsonBytes)
				}
			}
		}

	}
	return nil
}
