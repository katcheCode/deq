package deq

import (
	"fmt"
	"log"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"gitlab.com/katcheCode/deq/internal/data"
)

// upgradeDB upgrades the store's db to the current version.
// It is not safe to use the database concurrently with upgradeDB.
func (s *Store) upgradeDB(currentVersion string) error {

	txn := s.db.NewTransaction(true)
	defer txn.Discard()

	switch currentVersion {
	case dbCodeVersion:
		return nil
	case "1.0.0":
		log.Printf("[INFO] upgrading db from 1.0.0 to %s", dbCodeVersion)
		batchSize := 500
		u := &upgradeV1_0_0{}
		for txn := s.db.NewTransaction(true); u.NextBatch(txn, batchSize); txn = s.db.NewTransaction(true) {
			log.Printf("[INFO] %d indexes upgraded, %d indexes failed", u.updated, u.failed)
		}
		log.Printf("[INFO] %d indexes upgraded, %d indexes failed", u.updated, u.failed)
		log.Printf("[INFO] db upgraded to version %s", dbCodeVersion)

	default:
		return fmt.Errorf("unsupported on-disk version: %s", currentVersion)
	}

	err := txn.Set([]byte(dbVersionKey), []byte(dbCodeVersion))
	if err != nil {
		return err
	}

	err = txn.Commit(nil)
	if err != nil {
		return fmt.Errorf("commit db upgrade: %v", err)
	}

	return nil
}

func (s *Store) getDBVersion(txn *badger.Txn) (string, error) {
	item, err := txn.Get([]byte(dbVersionKey))
	if err == badger.ErrKeyNotFound {
		return "1.0.0", nil
	}
	if err != nil {
		return "", err
	}

	version, err := item.ValueCopy(nil)
	if err != nil {
		return "", err
	}

	return string(version), nil
}

type upgradeV1_0_0 struct {
	updated, failed int
	cursor          []byte
}

// NextBatch upgrades the database from v1.0.0 to the current version. It is the caller's
// responsibility to commit the Txn.
func (u *upgradeV1_0_0) NextBatch(txn *badger.Txn, batchSize int) bool {

	prefix := []byte{data.IndexTagV1_0_0, data.Sep}
	if len(u.cursor) == 0 {
		u.cursor = prefix
	}

	it := txn.NewIterator(badger.DefaultIteratorOptions)
	defer it.Close()

	var key []byte
	var i int
	for it.Seek(append(u.cursor, 0)); it.ValidForPrefix(prefix); it.Next() {
		i++
		if i >= batchSize {
			return true
		}

		item := it.Item()

		u.cursor = item.KeyCopy(u.cursor)

		var oldIndex data.IndexKeyV1_0_0
		err := data.UnmarshalIndexKeyV1_0_0(item.Key(), &oldIndex)
		if err != nil {
			log.Printf("unmarshal v1.0.0 index key: %v", err)
			continue
		}

		key, err = data.IndexKey{
			Topic: oldIndex.Topic,
			Value: oldIndex.Value,
		}.Marshal(key)
		if err != nil {
			log.Printf("marshal updated index: %v", err)
			continue
		}

		eTime, err := getEventTimePayload(txn, data.EventTimeKey{})
		if err != nil {
			log.Printf("get event time payload: %v", err)
			continue
		}

		buf, err := proto.Marshal(&data.IndexPayload{
			EventId:    oldIndex.ID,
			CreateTime: eTime.CreateTime,
		})

		err = txn.Set(key, buf)
		if err != nil {
			log.Printf("write new index: %v", err)
			continue
		}

		err = txn.Delete(item.Key())
		if err != nil {
			log.Printf("delete old index %v: %v", oldIndex, err)
			continue
		}

		u.updated++
	}
	u.failed = i - u.updated
	return false
}

const (
	dbVersionKey  = "___DEQ_DB_VERSION___"
	dbCodeVersion = "1.1.0"
)
