package engine

import (
	"errors"
	"fmt"
	// "io"
	"strings"

	"github.com/dgraph-io/badger/v4/options"

	"github.com/dgraph-io/badger/v4"
)

type BadgerDB struct {
	db *badger.DB
}

// Connect opens the Badger DB at given filepath
func (b *BadgerDB) Connect(filepath string) error {
	opts := badger.DefaultOptions(filepath)
	opts.Logger = nil
	opts.ValueThreshold = 65500       // Store most values inline
	opts.NumMemtables = 10            // More write buffering
	opts.NumLevelZeroTables = 10      // Delay compaction
	opts.NumLevelZeroTablesStall = 20 // Even more delay
	opts.Compression = options.None   // Space savings with fast compression
	opts.NumCompactors = 4            // Use more CPU for compaction
	opts.DetectConflicts = false      // Disable for single-writer perf

	db, err := badger.Open(opts)
	if err != nil {
		return err
	}
	b.db = db
	return nil
}

// Close closes the Badger DB
func (b *BadgerDB) Close() error {
	if b.db != nil {
		return b.db.Close()
	}
	return nil
}

// Set stores key-value pair (value must be []byte)
func (b *BadgerDB) Set(key string, value []byte) error {
	if value == nil {
		return errors.New("value must be non-nil []byte")
	}
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), value)
	})
}

// Get fetches string value by key
func (b *BadgerDB) Get(key string) (string, error) {
	var result string
	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		val, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		result = string(val)
		return nil
	})
	return result, err
}

// Delete removes a key
func (b *BadgerDB) Delete(key string) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// Exact returns whether a key exists
func (b *BadgerDB) Exact(key string) (bool, error) {
	err := b.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(key))
		return err
	})
	if err == badger.ErrKeyNotFound {
		return false, nil
	}
	return err == nil, err
}

// Scan returns keys with the given prefix
func (b *BadgerDB) Scan(prefix string) ([]string, error) {
	var keys []string
	err := b.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		prefixBytes := []byte(prefix)
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			keys = append(keys, string(it.Item().Key()))
		}
		return nil
	})
	return keys, err
}

func (b *BadgerDB) ScanWithValues(prefix string) (map[string]string, error) {
	result := make(map[string]string)
	err := b.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()

		prefixBytes := []byte(prefix)
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			key := string(item.Key())
			val, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			result[key] = string(val)
		}
		return nil
	})
	return result, err
}

// MGet gets multiple keys (simulated since Badger has no native MGet)
func (b *BadgerDB) MGet(keys []string) ([]string, error) {
	values := make([]string, len(keys))
	err := b.db.View(func(txn *badger.Txn) error {
		for i, key := range keys {
			item, err := txn.Get([]byte(key))
			if err != nil {
				if err == badger.ErrKeyNotFound {
					values[i] = ""
					continue
				}
				return err
			}
			val, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			values[i] = string(val)
		}
		return nil
	})
	return values, err
}

// FilterByPrefixWithOp filters all key-values with the given prefix and applies a comparison operator.
// valType should be one of: "string", "number", "timestamp".
// Inside engine.BadgerDB
// func (b *BadgerDB) FilterByPrefixWithOp(prefix, op, valType, value string) ([]string, error) {
// 	var matched []string
// 	err := b.db.View(func(txn *badger.Txn) error {
// 		it := txn.NewIterator(badger.DefaultIteratorOptions)
// 		defer it.Close()

// 		for it.Seek([]byte(prefix)); it.Valid(); it.Next() {
// 			item := it.Item()
// 			key := item.Key()
// 			keyStr := string(key)

// 			if !strings.HasPrefix(keyStr, prefix) {
// 				break
// 			}

// 			suffix := strings.TrimPrefix(keyStr, prefix)
// 			if !compare(suffix, value, op, valType) {
// 				continue
// 			}

// 			val, err := item.ValueCopy(nil)
// 			if err != nil {
// 				return err
// 			}
// 			matched = append(matched, string(val))
// 		}
// 		return nil
// 	})
// 	return matched, err
// }

func (b *BadgerDB) FilterByPrefixWithOp(prefix, op, valType, value string) ([]string, error) {
	var matched []string
	fmt.Println("DEBUG START --------")
	fmt.Println("Searching prefix:", prefix)
	fmt.Println("Operator:", op)
	fmt.Println("Value type:", valType)
	fmt.Println("Compare against value:", value)

	err := b.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Seek([]byte(prefix)); it.Valid(); it.Next() {
			item := it.Item()
			key := item.Key()
			keyStr := string(key)

			if !strings.HasPrefix(keyStr, prefix) {
				break
			}

			suffix := strings.TrimPrefix(keyStr, prefix)

			fmt.Println("KEY:", keyStr)
			fmt.Println(" → Suffix:", suffix)

			if !compare(suffix, value, op, valType) {
				fmt.Println(" ✘ Comparison failed")
				continue
			}

			val, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			fmt.Println(" ✔ Matched. Value:", string(val))
			matched = append(matched, string(val))
		}
		return nil
	})

	fmt.Println("DEBUG END ----------")
	return matched, err
}

// ConnectReadOnly opens Badger DB in read-only mode at the given filepath
func (b *BadgerDB) ConnectReadOnly(filepath string) error {
	opts := badger.DefaultOptions(filepath).
		WithReadOnly(true).
		WithLogger(nil) // Optional: silence logs
	db, err := badger.Open(opts)
	if err != nil {
		return err
	}
	b.db = db
	return nil
}

// Backup creates a backup of the database to the given writer.
// func (b *BadgerDB) Backup(w io.Writer) error {
// 	_, err := b.db.Backup(w, 0)
// 	return err
// }
