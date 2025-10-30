package storemanager

import (
	"encoding/json"
	"strings"
	"sync"
	"time"
)

// indexEntry stores a set of primary keys (PKs) associated with an index key.
type indexEntry struct {
	PKs map[string]bool
}

var (
	// Insertion buffer
	indexBuffer      = make(map[string]*indexEntry) // map[indexKey]*indexEntry
	indexBufferMutex sync.Mutex
	bufferLimit      = 500
	flushInterval    = 500 * time.Millisecond

	// Deletion buffer
	indexDeleteBuffer      = make(map[string]*indexEntry)
	indexDeleteBufferMutex sync.Mutex
	deleteBufferLimit      = 500
	deleteFlushInterval    = 500 * time.Millisecond
)

// InitIndexSystem initializes background flushers for both insert and delete index buffers.
func InitIndexSystem() {
	go func() {
		for {
			time.Sleep(flushInterval)
			flushIndexBuffer()
		}
	}()
	go func() {
		for {
			time.Sleep(deleteFlushInterval)
			flushIndexDeleteBuffer()
		}
	}()
}

// AddToIndexBuffer queues a PK for a given index key into the insertion buffer.
func AddToIndexBuffer(indexKey, pk string) {
	indexBufferMutex.Lock()
	defer indexBufferMutex.Unlock()

	if indexBuffer[indexKey] == nil {
		indexBuffer[indexKey] = &indexEntry{PKs: make(map[string]bool)}
	}
	indexBuffer[indexKey].PKs[pk] = true

	if len(indexBuffer) >= bufferLimit {
		go flushIndexBuffer()
	}
}

// AddToDeleteBuffer queues a PK for deletion from a given index key into the deletion buffer.
func AddToDeleteBuffer(indexKey, pk string) {
	indexDeleteBufferMutex.Lock()
	defer indexDeleteBufferMutex.Unlock()

	if indexDeleteBuffer[indexKey] == nil {
		indexDeleteBuffer[indexKey] = &indexEntry{PKs: make(map[string]bool)}
	}
	indexDeleteBuffer[indexKey].PKs[pk] = true

	if len(indexDeleteBuffer) >= deleteBufferLimit {
		go flushIndexDeleteBuffer()
	}
}

// flushIndexBuffer flushes all buffered insert operations to disk.
func flushIndexBuffer() {
	indexBufferMutex.Lock()
	bufferCopy := indexBuffer
	indexBuffer = make(map[string]*indexEntry)
	indexBufferMutex.Unlock()

	for indexKey, entry := range bufferCopy {
		parts := strings.Split(indexKey, ":")
		if len(parts) < 4 {
			continue // malformed index key
		}
		db := parts[1]
		table := parts[2]
		column := parts[3]

		diskStore, err := GetDiskStore(db + "/" + table + "/index/" + column + "/" + GetIndexFileName(indexKey))
		if err != nil {
			continue
		}

		raw, err := diskStore.Get(indexKey)
		existingPKs := make(map[string]bool)
		if err == nil {
			var arr []string
			if json.Unmarshal([]byte(raw), &arr) == nil {
				for _, pk := range arr {
					existingPKs[pk] = true
				}
			}
		}

		changed := false
		for pk := range entry.PKs {
			if !existingPKs[pk] {
				existingPKs[pk] = true
				changed = true
			}
		}

		if changed {
			finalList := make([]string, 0, len(existingPKs))
			for pk := range existingPKs {
				finalList = append(finalList, pk)
			}
			data, _ := json.Marshal(finalList)
			_ = diskStore.Set(indexKey, data)
		}
	}
}

// flushIndexDeleteBuffer flushes all buffered delete operations to disk.
func flushIndexDeleteBuffer() {
	indexDeleteBufferMutex.Lock()
	bufferCopy := indexDeleteBuffer
	indexDeleteBuffer = make(map[string]*indexEntry)
	indexDeleteBufferMutex.Unlock()

	for indexKey, entry := range bufferCopy {
		parts := strings.Split(indexKey, ":")
		if len(parts) < 6 {
			continue // malformed index key
		}
		db := parts[1]
		table := parts[2]
		column := parts[3]

		diskStore, err := GetDiskStore(db + "/" + table + "/index/" + column + "/" + GetIndexFileName(indexKey))
		if err != nil {
			continue
		}

		raw, err := diskStore.Get(indexKey)
		if err != nil {
			continue
		}

		var pkList []string
		if err := json.Unmarshal([]byte(raw), &pkList); err != nil {
			continue
		}

		toDelete := entry.PKs
		newList := make([]string, 0, len(pkList))
		changed := false
		for _, pk := range pkList {
			if !toDelete[pk] {
				newList = append(newList, pk)
			} else {
				changed = true
			}
		}

		if changed {
			data, _ := json.Marshal(newList)
			_ = diskStore.Set(indexKey, data)
		}
	}
}

// func GetIndexFromRamSync(indexKey string){
// 	index , ok := indexBuffer[indexKey]
// 	if
// }
