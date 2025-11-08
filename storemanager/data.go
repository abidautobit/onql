package storemanager

import (
	"encoding/json"
	"fmt"
	"onql/utils"
)

type Data struct {
	db       string
	table    string
	columns  map[string]any
	storage  map[string]string // storage type: "disk" or "ram"
	datatype map[string]string // column data types
}

type DataAlter struct {
	db        string
	table     string
	oldCol    string
	newCol    string
	action    string                       // "rename", "add", "remove", "changeDataType"
	oldSchema map[string]map[string]string // old column schema
	newSchema map[string]map[string]string // new column schema
}

// NewData creates a new Data instance with all fields set.
func NewData(db, table string, columns map[string]any, storage, datatype map[string]string) Data {
	return Data{
		db:       db,
		table:    table,
		columns:  columns,
		storage:  storage,
		datatype: datatype,
	}
}

func InsertData(data Data) (string, error) {
	// add primary key
	pk := utils.GenerateID()
	data.columns["id"] = pk
	storeKey := "row:" + data.db + ":" + data.table + ":" + pk
	//save to ram it will flush to disk auto
	jsonBytes, err := json.Marshal(data.columns)
	if err != nil {
		return "", err
	}
	SaveData(storeKey, jsonBytes)
	// save index here before returning
	// Save indexes (no previous data, so empty)
	if err := SaveIndex(data, Data{}); err != nil {
		return "", err
	}
	return pk, nil
}

func UpdateData(data Data) error {

	storeKey := "row:" + data.db + ":" + data.table + ":" + data.columns["id"].(string)

	// Step 1: Get previous data for index update
	prevRows, err := GetData(data.db, data.table, []string{data.columns["id"].(string)})
	if err != nil {
		return err
	}

	if len(prevRows) == 0 {
		return fmt.Errorf("no existing data found for update")
	}

	prevData := Data{
		db:       data.db,
		table:    data.table,
		columns:  prevRows[0],
		storage:  data.storage,
		datatype: data.datatype,
	}

	jsonBytes, err := json.Marshal(data.columns)
	if err != nil {
		return err
	}

	SaveData(storeKey, jsonBytes)

	// Save index before returning
	if err := SaveIndex(data, prevData); err != nil {
		return err
	}
	return nil
}

func GetData(db, table string, pks []string) ([]map[string]any, error) {
	result := make([]map[string]any, 0)
	for _, pk := range pks {
		storeKey := "row:" + db + ":" + table + ":" + pk
		row := make(map[string]any)

		// Fetch from disk
		data, ok := GetDataFromRamSync(storeKey)
		if ok {
			err := json.Unmarshal([]byte(data), &row)
			if err != nil {
				return nil, err
			}
		} else {
			diskStore, err := GetDiskStore(db + "/" + table + "/data/" + utils.GetIDPrefix(pk) + ".db")
			if err == nil {
				raw, err := diskStore.Get(storeKey)
				if err == nil {
					json.Unmarshal([]byte(raw), &row)
				}
			}
		}
		result = append(result, row)
	}
	return result, nil
}

func DeleteData(db string, table string, pks []string) error {
	for _, pk := range pks {
		storeKey := "row:" + db + ":" + table + ":" + pk

		// Step 1: Load full row for index cleanup
		dataRows, err := GetData(db, table, []string{pk})
		if err != nil {
			return err
		}
		if len(dataRows) == 0 {
			continue
		}
		fullRow := dataRows[0]

		// Step 2: Fetch schema types for index deletion
		schema, err := GetTableSchema(db, table)
		if err != nil {
			return err
		}
		datatypes := make(map[string]string)
		for col, meta := range schema {
			datatypes[col] = meta["type"]
		}
		oldData := Data{
			db:       db,
			table:    table,
			columns:  fullRow,
			datatype: datatypes,
		}
		// Delete indexes using new DeleteIndex
		if err := DeleteIndex(oldData); err != nil {
			return err
		}

		// Delete from RAM
		DeleteDataFromRamSync(storeKey)

		// Step 3: Delete from disk
		diskStore, err := GetDiskStore(db + "/" + table + "/data/" + utils.GetIDPrefix(pk) + ".db")
		if err != nil {
			return err
		}

		if err := diskStore.Delete(storeKey); err != nil {
			return err
		}

	}
	return nil
}

func GetSpecificData(db, table string, pks, columns []string) ([]map[string]any, error) {
	schema := getStorageMap(db, table)
	var result []map[string]any
	for _, pk := range pks {
		storeKey := "row:" + db + ":" + table + ":" + pk
		row := make(map[string]any)
		// Fetch from disk
		diskStore, err := GetDiskStore(db + "/" + table + "/data/" + utils.GetIDPrefix(pk) + ".db")
		if err == nil {
			raw, err := diskStore.Get(storeKey)
			if err == nil {
				var diskData map[string]any
				json.Unmarshal([]byte(raw), &diskData)
				for col, val := range diskData {
					if len(columns) == 0 || utils.Contains(columns, col) {
						if schema[col] == "json" {
							b, _ := json.Marshal(val)
							var obj any
							json.Unmarshal(b, &obj)
							row[col] = obj
						} else {
							row[col] = val
						}
					}
				}
			}
		}

		result = append(result, row)
	}
	return result, nil
}

// GetAllPks returns all primary keys (ids) from all data stores for the given db and table.
func GetAllPks(db, table string) ([]string, error) {
	pks := make([]string, 0)
	stores := getTableDataStores(db, table)
	for _, store := range stores {
		keys := getAllKeysFromStore(store, db, table)
		pks = append(pks, keys...)
	}
	return pks, nil
}

// Call by schema
func alterData(da DataAlter) error {
	// get all stores for the table
	tableStores := getTableDataStores(da.db, da.table)

	switch da.action {
	case "renameColumn":
		// Handle renameColumn
		for _, store := range tableStores {
			// get all pk
			pks := getAllKeysFromStore(store, da.db, da.table)
			//get all data from that pk
			data, _ := GetData(da.db, da.table, pks)
			for _, row := range data {
				if oldValue, exists := row[da.oldCol]; exists {
					row[da.newCol] = oldValue
					delete(row, da.oldCol)
					// Update the row in the store
					storeKey := "row:" + da.db + ":" + da.table + ":" + row["id"].(string)
					jsonBytes, _ := json.Marshal(row)
					store.Set(storeKey, jsonBytes)
				}
			}
		}
	case "addColumn":
		// Handle addColumn
		for _, store := range tableStores {
			pks := getAllKeysFromStore(store, da.db, da.table)
			data, _ := GetData(da.db, da.table, pks)
			for _, row := range data {
				row[da.newCol] = da.newSchema[da.newCol]["default"]
				// Update the row in the store
				storeKey := "row:" + da.db + ":" + da.table + ":" + row["id"].(string)
				jsonBytes, _ := json.Marshal(row)
				store.Set(storeKey, jsonBytes)
			}
		}
	case "dropColumn":
		// Handle dropColumn
		for _, store := range tableStores {
			pks := getAllKeysFromStore(store, da.db, da.table)
			// get all data from that pk
			data, _ := GetData(da.db, da.table, pks)
			for _, row := range data {
				delete(row, da.oldCol)
				// Update the row in the store
				storeKey := "row:" + da.db + ":" + da.table + ":" + row["id"].(string)
				jsonBytes, _ := json.Marshal(row)
				store.Set(storeKey, jsonBytes)
			}
		}

	}
	return nil
}

// used when table or db renamed not column
func AlterDataKeys(db string, tables []string, new map[string]string) error {
	// If tables empty mean db change other wise table change
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
		tableStores := getTableDataStores(db, table)
		for _, store := range tableStores {
			keys := getAllKeysFromStore(store, db, table)
			for _, key := range keys {
				storeKey := "row:" + db + ":" + table + ":" + key
				row, _ := store.Get(storeKey)
				store.Delete(storeKey) // delete old key
				newStoreKey := "row:" + newDb + ":" + newTable + ":" + key
				jsonBytes, _ := json.Marshal(row)
				store.Set(newStoreKey, jsonBytes)
			}
		}
	}
	return nil
}
