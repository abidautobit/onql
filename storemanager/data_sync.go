package storemanager

import (
	"fmt"
	"onql/utils"
	"strings"
	"sync"
	"time"
)

// dbdata is a map representing a single data record
type dbdata []byte

var (
	dataRamStorage      = make(map[string]dbdata) // map[dataKey]dbdata
	dataRamStorageMutex sync.Mutex
	dataBufferLimit     = 500
	dataFlushInterval   = 500 * time.Millisecond
)

// InitIndexSystem initializes background flushers for both insert and delete index buffers.
func InitDataFlushSystem() {
	go func() {
		for {
			time.Sleep(dataFlushInterval)
			flushDataRamStorageToDisk()
		}
	}()
	// go func() {
	// 	for {
	// 		time.Sleep(deleteFlushInterval)
	// 		flushIndexDeleteBuffer()
	// 	}
	// }()
}

// SaveData sets the value for a given dataKey, replacing any previous value
func SaveData(dataKey string, value dbdata) {
	dataRamStorageMutex.Lock()
	defer dataRamStorageMutex.Unlock()
	dataRamStorage[dataKey] = value
	if len(dataRamStorage) >= dataBufferLimit {
		go flushDataRamStorageToDisk()
	}
}

// GetData retrieves the value for a given dataKey
func GetDataFromRamSync(dataKey string) (dbdata, bool) {
	dataRamStorageMutex.Lock()
	defer dataRamStorageMutex.Unlock()
	val, ok := dataRamStorage[dataKey]
	return val, ok
}

// DeleteData removes the data for a given dataKey
func DeleteDataFromRamSync(dataKey string) {
	dataRamStorageMutex.Lock()
	defer dataRamStorageMutex.Unlock()
	delete(dataRamStorage, dataKey)
}

// flushDataRamStorageToDisk is a stub for persisting dataRamStorage to disk
func flushDataRamStorageToDisk() {
	dataRamStorageMutex.Lock()
	defer dataRamStorageMutex.Unlock()
	for dataKey, row := range dataRamStorage {
		// Expect dataKey format: row:db:table:pk
		parts := strings.Split(dataKey, ":")
		if len(parts) < 4 {
			continue // malformed key
		}
		db := parts[1]
		table := parts[2]
		pk := parts[3]
		// Save to disk
		diskStore, err := GetDiskStore(db + "/" + table + "/data/" + utils.GetIDPrefix(pk) + ".db")
		if err != nil {
			fmt.Println(err)
			continue
		}
		// jsonBytes, err := json.Marshal(row)
		// if err != nil {
		// fmt.Println(err)
		// continue
		// }
		fmt.Println("Flushing data to disk:", row)
		err = diskStore.Set(dataKey, row)
		if err != nil {
			fmt.Println(err)
			continue
		}
		//save in cashe also if allow
		// CasheDb.Set(dataKey, jsonBytes)

		delete(dataRamStorage, dataKey) // Clear from RAM after saving
	}
}
