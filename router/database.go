package router

import (
	"encoding/json"
	"fmt"
	"onql/database"
	"onql/database/get"
	"onql/storemanager"
)

// DatabaseRequest represents the structure of the JSON payload
type DatabaseRequest struct {
	Function string            `json:"function"`
	Args     []json.RawMessage `json:"args"`
}

// CallDatabaseFuncDirect provides direct function calls without reflection for better performance.
// This eliminates the overhead of reflection by mapping function names to direct function calls.
func CallDatabaseFuncDirect(name string, args []json.RawMessage) (any, error) {
	switch name {
	// --- Schema Functions ---
	case "GetDatabases":
		if len(args) != 0 {
			return nil, fmt.Errorf("GetDatabases expects 0 args, got %d", len(args))
		}
		return database.GetDatabases()

	case "GetTables":
		if len(args) != 1 {
			return nil, fmt.Errorf("GetTables expects 1 arg, got %d", len(args))
		}
		var db string
		if err := json.Unmarshal(args[0], &db); err != nil {
			return nil, fmt.Errorf("invalid db arg: %w", err)
		}
		return database.GetTables(db)

	case "GetTableSchema":
		if len(args) != 2 {
			return nil, fmt.Errorf("GetTableSchema expects 2 args, got %d", len(args))
		}
		var db, table string
		if err := json.Unmarshal(args[0], &db); err != nil {
			return nil, fmt.Errorf("invalid db arg: %w", err)
		}
		if err := json.Unmarshal(args[1], &table); err != nil {
			return nil, fmt.Errorf("invalid table arg: %w", err)
		}
		return database.GetTableSchema(db, table)

	case "GetFullSchema":
		if len(args) != 0 {
			return nil, fmt.Errorf("GetFullSchema expects 0 args, got %d", len(args))
		}
		return database.GetFullSchema()

	case "CreateDatabase":
		if len(args) != 1 {
			return nil, fmt.Errorf("CreateDatabase expects 1 arg, got %d", len(args))
		}
		var name string
		if err := json.Unmarshal(args[0], &name); err != nil {
			return nil, fmt.Errorf("invalid name arg: %w", err)
		}
		return nil, database.CreateDatabase(name)

	case "CreateTable":
		if len(args) != 3 {
			return nil, fmt.Errorf("CreateTable expects 3 args, got %d", len(args))
		}
		var db, table string
		var schema map[string]map[string]string
		if err := json.Unmarshal(args[0], &db); err != nil {
			return nil, fmt.Errorf("invalid db arg: %w", err)
		}
		if err := json.Unmarshal(args[1], &table); err != nil {
			return nil, fmt.Errorf("invalid table arg: %w", err)
		}
		if err := json.Unmarshal(args[2], &schema); err != nil {
			return nil, fmt.Errorf("invalid schema arg: %w", err)
		}
		return nil, database.CreateTable(db, table, schema)

	case "RenameDatabase":
		if len(args) != 2 {
			return nil, fmt.Errorf("RenameDatabase expects 2 args, got %d", len(args))
		}
		var oldName, newName string
		if err := json.Unmarshal(args[0], &oldName); err != nil {
			return nil, fmt.Errorf("invalid oldName arg: %w", err)
		}
		if err := json.Unmarshal(args[1], &newName); err != nil {
			return nil, fmt.Errorf("invalid newName arg: %w", err)
		}
		return nil, database.RenameDatabase(oldName, newName)

	case "RenameTable":
		if len(args) != 3 {
			return nil, fmt.Errorf("RenameTable expects 3 args, got %d", len(args))
		}
		var db, oldName, newName string
		if err := json.Unmarshal(args[0], &db); err != nil {
			return nil, fmt.Errorf("invalid db arg: %w", err)
		}
		if err := json.Unmarshal(args[1], &oldName); err != nil {
			return nil, fmt.Errorf("invalid oldName arg: %w", err)
		}
		if err := json.Unmarshal(args[2], &newName); err != nil {
			return nil, fmt.Errorf("invalid newName arg: %w", err)
		}
		return nil, database.RenameTable(db, oldName, newName)

	case "AlterTable":
		if len(args) != 3 {
			return nil, fmt.Errorf("AlterTable expects 3 args, got %d", len(args))
		}
		var db, table string
		var alters map[string]map[string]string
		if err := json.Unmarshal(args[0], &db); err != nil {
			return nil, fmt.Errorf("invalid db arg: %w", err)
		}
		if err := json.Unmarshal(args[1], &table); err != nil {
			return nil, fmt.Errorf("invalid table arg: %w", err)
		}
		if err := json.Unmarshal(args[2], &alters); err != nil {
			return nil, fmt.Errorf("invalid alters arg: %w", err)
		}
		return nil, database.AlterTable(db, table, alters)

	case "DeleteDatabase":
		if len(args) != 1 {
			return nil, fmt.Errorf("DeleteDatabase expects 1 arg, got %d", len(args))
		}
		var db string
		if err := json.Unmarshal(args[0], &db); err != nil {
			return nil, fmt.Errorf("invalid db arg: %w", err)
		}
		return nil, database.DeleteDatabase(db)

	case "DeleteTable":
		if len(args) != 2 {
			return nil, fmt.Errorf("DeleteTable expects 2 args, got %d", len(args))
		}
		var db, table string
		if err := json.Unmarshal(args[0], &db); err != nil {
			return nil, fmt.Errorf("invalid db arg: %w", err)
		}
		if err := json.Unmarshal(args[1], &table); err != nil {
			return nil, fmt.Errorf("invalid table arg: %w", err)
		}
		return nil, database.DeleteTable(db, table)

	case "RefereshIndexes":
		return nil, database.RefereshIndexes()

	// --- Protocol Functions ---
	case "SetProtocol":
		if len(args) != 2 {
			return nil, fmt.Errorf("SetProtocol expects 2 args, got %d", len(args))
		}
		var password string
		var protocol storemanager.QueryProtocol
		if err := json.Unmarshal(args[0], &password); err != nil {
			return nil, fmt.Errorf("invalid password arg: %w", err)
		}
		if err := json.Unmarshal(args[1], &protocol); err != nil {
			return nil, fmt.Errorf("invalid protocol arg: %w", err)
		}
		return nil, database.SetProtocol(password, protocol)

	case "GetAllProtocols":
		if len(args) != 0 {
			return nil, fmt.Errorf("GetAllProtocols expects 0 args, got %d", len(args))
		}
		return database.GetAllProtocols()

	case "DeleteProtocolByPassword":
		if len(args) != 1 {
			return nil, fmt.Errorf("DeleteProtocolByPassword expects 1 arg, got %d", len(args))
		}
		var password string
		if err := json.Unmarshal(args[0], &password); err != nil {
			return nil, fmt.Errorf("invalid password arg: %w", err)
		}
		return nil, database.DeleteProtocolByPassword(password)

	case "GetProtocolPasswords":
		if len(args) != 0 {
			return nil, fmt.Errorf("GetProtocolPasswords expects 0 args, got %d", len(args))
		}
		return database.GetProtocolPasswords()

	// case "SetupProtocols":
	// 	if len(args) != 0 {
	// 		return nil, fmt.Errorf("SetupProtocols expects 0 args, got %d", len(args))
	// 	}
	// 	return nil, database.SetupProtocols()

	// --- Data Manipulation Functions ---
	case "Insert", "insert":
		if len(args) != 3 {
			return nil, fmt.Errorf("insert expects 3 args, got %d", len(args))
		}
		var db, table string
		var record map[string]any
		if err := json.Unmarshal(args[0], &db); err != nil {
			return nil, fmt.Errorf("invalid db arg: %w", err)
		}
		if err := json.Unmarshal(args[1], &table); err != nil {
			return nil, fmt.Errorf("invalid table arg: %w", err)
		}
		if err := json.Unmarshal(args[2], &record); err != nil {
			return nil, fmt.Errorf("invalid record arg: %w", err)
		}
		return database.Insert(db, table, record)

	case "Update":
		if len(args) != 3 {
			return nil, fmt.Errorf("update expects 3 args, got %d", len(args))
		}
		var db, table string
		var record map[string]any
		if err := json.Unmarshal(args[0], &db); err != nil {
			return nil, fmt.Errorf("invalid db arg: %w", err)
		}
		if err := json.Unmarshal(args[1], &table); err != nil {
			return nil, fmt.Errorf("invalid table arg: %w", err)
		}
		if err := json.Unmarshal(args[2], &record); err != nil {
			return nil, fmt.Errorf("invalid record arg: %w", err)
		}
		return nil, database.Update(db, table, record)

	case "UpdatePartial":
		if len(args) != 3 {
			return nil, fmt.Errorf("UpdatePartial expects 3 args, got %d", len(args))
		}
		var db, table string
		var record map[string]any
		if err := json.Unmarshal(args[0], &db); err != nil {
			return nil, fmt.Errorf("invalid db arg: %w", err)
		}
		if err := json.Unmarshal(args[1], &table); err != nil {
			return nil, fmt.Errorf("invalid table arg: %w", err)
		}
		if err := json.Unmarshal(args[2], &record); err != nil {
			return nil, fmt.Errorf("invalid record arg: %w", err)
		}
		return nil, database.UpdatePartial(db, table, record)

	case "Delete":
		if len(args) != 3 {
			return nil, fmt.Errorf("delete expects 3 args, got %d", len(args))
		}
		var db, table string
		var ids []string
		if err := json.Unmarshal(args[0], &db); err != nil {
			return nil, fmt.Errorf("invalid db arg: %w", err)
		}
		if err := json.Unmarshal(args[1], &table); err != nil {
			return nil, fmt.Errorf("invalid table arg: %w", err)
		}
		if err := json.Unmarshal(args[2], &ids); err != nil {
			return nil, fmt.Errorf("invalid ids arg: %w", err)
		}
		return nil, database.Delete(db, table, ids)

	// --- Get Package Functions ---
	case "GetWithPKs":
		if len(args) != 4 {
			return nil, fmt.Errorf("GetWithPKs expects 4 args, got %d", len(args))
		}
		var db, table string
		var pks, columns []string
		if err := json.Unmarshal(args[0], &db); err != nil {
			return nil, fmt.Errorf("invalid db arg: %w", err)
		}
		if err := json.Unmarshal(args[1], &table); err != nil {
			return nil, fmt.Errorf("invalid table arg: %w", err)
		}
		if err := json.Unmarshal(args[2], &pks); err != nil {
			return nil, fmt.Errorf("invalid pks arg: %w", err)
		}
		if err := json.Unmarshal(args[3], &columns); err != nil {
			return nil, fmt.Errorf("invalid columns arg: %w", err)
		}
		return get.GetWithPKs(db, table, pks)

	case "GetPksFromIndex":
		if len(args) != 3 {
			return nil, fmt.Errorf("GetPksFromIndex expects 3 args, got %d", len(args))
		}
		var db, table, index string
		if err := json.Unmarshal(args[0], &db); err != nil {
			return nil, fmt.Errorf("invalid db arg: %w", err)
		}
		if err := json.Unmarshal(args[1], &table); err != nil {
			return nil, fmt.Errorf("invalid table arg: %w", err)
		}
		if err := json.Unmarshal(args[2], &index); err != nil {
			return nil, fmt.Errorf("invalid index arg: %w", err)
		}
		return get.GetPksFromIndex(db, table, index)

	// case "GetRamData":
	// 	if len(args) != 3 {
	// 		return nil, fmt.Errorf("GetRamData expects 3 args, got %d", len(args))
	// 	}
	// 	var db, table string
	// 	var pks []string
	// 	if err := json.Unmarshal(args[0], &db); err != nil {
	// 		return nil, fmt.Errorf("invalid db arg: %w", err)
	// 	}
	// 	if err := json.Unmarshal(args[1], &table); err != nil {
	// 		return nil, fmt.Errorf("invalid table arg: %w", err)
	// 	}
	// 	if err := json.Unmarshal(args[2], &pks); err != nil {
	// 		return nil, fmt.Errorf("invalid pks arg: %w", err)
	// 	}
	// 	return get.GetRamData(db, table, pks)

	case "EvaluateQuery":
		if len(args) != 1 {
			return nil, fmt.Errorf("EvaluateQuery expects 1 arg, got %d", len(args))
		}
		var query get.Query
		if err := json.Unmarshal(args[0], &query); err != nil {
			return nil, fmt.Errorf("invalid query arg: %w", err)
		}
		return get.EvaluateQuery(query)

	default:
		return nil, fmt.Errorf("function %q not found", name)
	}
}

// handleDatabaseRequest processes a Message targeting the database using direct function calls.
// msg.Payload is a JSON object with structure: {"function":"FunctionName","args":[...]}
func handleDatabaseRequest(msg *Message) {
	// parse the payload into DatabaseRequest structure
	var dbReq DatabaseRequest
	if err := json.Unmarshal([]byte(msg.Payload), &dbReq); err != nil {
		payload := fmt.Sprintf("error: invalid payload format: %v", err)
		resmsg := &Message{
			ID:      "database",
			Target:  msg.ID,
			RID:     msg.RID,
			Payload: payload,
			Type:    "response",
		}
		Route(resmsg)
		return
	}
	// invoke direct call using the function name from payload
	res, err := CallDatabaseFuncDirect(dbReq.Function, dbReq.Args)

	// prepare payload string
	var payload string
	if err != nil {
		payload = fmt.Sprintf("error: %v", err)
	} else {
		b, e := json.Marshal(res)
		if e != nil {
			payload = fmt.Sprintf("error marshaling result: %v", e)
		} else {
			payload = string(b)
		}
	}

	// route response
	resmsg := &Message{
		ID:      "database",
		RID:     msg.RID,
		Target:  msg.ID,
		Payload: payload,
		Type:    "response",
	}
	Route(resmsg)
}
