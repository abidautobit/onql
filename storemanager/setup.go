package storemanager

import (
	"fmt"
	"onql/config"
	"onql/engine"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var shutdownOnce sync.Once

var diskStore map[string]*engine.BadgerDB

var FullSchema map[string]map[string]map[string]map[string]string // map[database][table][column][attribute]value
var FullSchemaStorages map[string]map[string]map[string]string
var FullSchemaTypes map[string]map[string]map[string]string

func SetSchema(storages, types map[string]map[string]map[string]string, fullSchema map[string]map[string]map[string]map[string]string) {
	FullSchemaStorages = storages
	FullSchemaTypes = types
	FullSchema = fullSchema
}

func getStorageMap(db, table string) map[string]string {
	storageMap := make(map[string]string)
	if tableSchema, ok := FullSchema[db][table]; ok {
		for col, meta := range tableSchema {
			storageMap[col] = meta["storage"]
		}
	}
	return storageMap
}

// func getTypeMap(db, table string) map[string]string {
// 	typeMap := make(map[string]string)
// 	if tableSchema, ok := FullSchema[db][table]; ok {
// 		for col, meta := range tableSchema {
// 			typeMap[col] = meta["type"]
// 		}
// 	}
// 	return typeMap
// }

// for read only operations
// var readDiskStore map[string]*engine.BadgerDB

// var blockInterpretation bool = false // used to block interpretation of queries when schema is being changed
func CleanDiskPath(fullPath string) string {
	basePath := config.Env("DISK_PATH")
	rel, err := filepath.Rel(basePath, fullPath)
	if err != nil {
		return filepath.ToSlash(strings.Trim(fullPath, "/\\"))
	}
	return filepath.ToSlash(strings.Trim(rel, "/\\"))
}

func SetupStore() {
	// load configration
	diskPath := config.Env("DISK_PATH")
	// setup redis it will use pool in it so one connection appropriate
	// setup badger db
	//load all badger db folders paths
	diskStore = make(map[string]*engine.BadgerDB)
	fmt.Println(diskStore, "SETUP STORE")

	paths, err := GetAllBadgerDBFoldersRecursively(diskPath)
	if err != nil {
		panic(err)
	}
	//setup badger db instance for each path
	for _, path := range paths {
		db := engine.BadgerDB{}
		err = db.Connect(path)
		if err != nil {
			panic(err)
		}
		//normalize path to be relative to DISK_PATH
		// this is important to have consistent keys in diskStore map
		relPath := CleanDiskPath(path)
		diskStore[relPath] = &db
	}
	//Initiate data flusher
	InitDataFlushSystem()
	// Initialize index flusher
	InitIndexSystem()
}

// GetAllBadgerDBFoldersRecursively finds all folders under rootPath ending with .db
func GetAllBadgerDBFoldersRecursively(rootPath string) ([]string, error) {
	var dbFolders []string

	err := filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Skip if any error accessing path
			return nil
		}
		if d.IsDir() && strings.HasSuffix(d.Name(), ".db") {
			dbFolders = append(dbFolders, path)
		}
		return nil
	})

	return dbFolders, err
}

func GetDiskStore(path string) (*engine.BadgerDB, error) {
	// relPath := CleanDiskPath((path))
	// for p, _ := range diskStore {
	// 	fmt.Println("Checking disk store for path:", p)
	// }
	// fmt.Println("Getting disk rel store for path:", path)
	if db, exists := diskStore[path]; exists {
		return db, nil
	}

	fullPath := filepath.Join(config.Env("DISK_PATH"), path)
	db := &engine.BadgerDB{}
	if err := db.Connect(fullPath); err != nil {
		return nil, err
	}
	fmt.Println(diskStore, "DISKSTORE")
	diskStore[path] = db
	return db, nil
}

// func GetReadDiskStore(path string) (*engine.BadgerDB, error) {
// 	if db, exists := readDiskStore[path]; exists {
// 		return db, nil
// 	}

// 	fullPath := filepath.Join(config.Env("DISK_PATH"), path)
// 	db := &engine.BadgerDB{}
// 	if err := db.ConnectReadOnly(fullPath); err != nil {
// 		return nil, err
// 	}
// 	readDiskStore[path] = db
// 	return db, nil
// }

// getAllKeysFromStore retrieves all keys from the specified database and table in the disk store
func getAllKeysFromStore(store *engine.BadgerDB, db, table string) []string {
	pks := make([]string, 0)
	// Get all keys in the store
	keys, _ := store.Scan("row:" + db + ":" + table + ":")
	for _, key := range keys {
		pks = append(pks, strings.TrimPrefix(key, "row:"+db+":"+table+":"))
	}
	return pks
}

func getAllKeysFromIndexStore(store *engine.BadgerDB, db, table string) []string {
	pks := make([]string, 0)
	// Get all keys in the store
	keys, _ := store.Scan("index:" + db + ":" + table + ":")
	for _, key := range keys {
		pks = append(pks, strings.TrimPrefix(key, "index:"+db+":"+table+":"))
	}
	return pks
}

func getTableDataStores(db, table string) map[string]*engine.BadgerDB {
	tableStores := make(map[string]*engine.BadgerDB)
	expectedPrefix := filepath.ToSlash(db + "/" + table + "/data/")

	for file, store := range diskStore {
		if !strings.HasPrefix(file, expectedPrefix) {
			continue // skip irrelevant stores
		}
		tableStores[file] = store
	}
	return tableStores
}

func getTableIndexStores(db, table string) map[string]*engine.BadgerDB {
	indexStores := make(map[string]*engine.BadgerDB)
	expectedPrefix := filepath.ToSlash(db + "/" + table + "/index/")

	for file, store := range diskStore {
		if !strings.HasPrefix(file, expectedPrefix) {
			continue // skip irrelevant stores
		}
		indexStores[file] = store
	}
	return indexStores
}

func getDbStores(db string) map[string]*engine.BadgerDB {
	dbStores := make(map[string]*engine.BadgerDB)
	expectedPrefix := filepath.ToSlash(db + "/")

	for file, store := range diskStore {
		if !strings.HasPrefix(file, expectedPrefix) {
			continue // skip irrelevant stores
		}
		dbStores[file] = store
	}
	return dbStores
}

func getIndexStores(db, table, column string) map[string]*engine.BadgerDB {
	indexStores := make(map[string]*engine.BadgerDB)
	expectedPrefix := filepath.ToSlash(db + "/" + table + "/index/" + column + "/")

	for file, store := range diskStore {
		// fmt.Println("Checking index store for file:", file, " with expected prefix:", expectedPrefix)
		if strings.HasPrefix(file, expectedPrefix) {
			indexStores[file] = store
		}
	}
	return indexStores
}

func CloseAllStores() {
	shutdownOnce.Do(func() {
		for path, db := range diskStore {
			fmt.Println("Closing DB at:", path)
			if err := db.Close(); err != nil {
				fmt.Println("Error closing DB at", path, ":", err)
			}
		}
		//read only stores
		// for path, db := range readDiskStore {
		// 	fmt.Println("Closing ReadOnly DB at:", path)
		// 	if err := db.Close(); err != nil {
		// 		fmt.Println("Error closing read-only DB at", path, ":", err)
		// 	}
		// }

	})
}

func Destruct() {
	CloseAllStores()
}
