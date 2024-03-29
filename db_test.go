package deq

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/google/go-cmp/cmp"
	"gitlab.com/katcheCode/deq/internal/data"
)

func TestWriteEvent(t *testing.T) {

	dir, err := ioutil.TempDir("", "test-write-event")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	opts := badger.DefaultOptions
	opts.Dir = dir
	opts.ValueDir = dir
	db, err := badger.Open(opts)
	defer db.Close()
	if err != nil {
		t.Fatal("open db: ", err)
	}

	txn := db.NewTransaction(true)
	defer txn.Discard()

	// Setup existing channels - currently we have to ack an existing event on the
	// channels we want
	err = writeEvent(txn, &Event{
		Topic:      "topic",
		ID:         "event0",
		CreateTime: time.Now(),
		Payload:    []byte{1},
	})
	if err != nil {
		t.Fatal("write event: ", err)
	}
	err = writeEvent(txn, &Event{
		Topic:      "topic",
		ID:         "event00",
		CreateTime: time.Now(),
		Payload:    []byte{1},
	})
	if err != nil {
		t.Fatal("write event: ", err)
	}

	dequeuePayload := data.ChannelPayload{
		EventState: data.EventState_DEQUEUED_OK,
	}

	channelKey := data.ChannelKey{
		ID:      "event0",
		Topic:   "topic",
		Channel: "channel",
	}
	err = setChannelEvent(txn, channelKey, dequeuePayload)
	if err != nil {
		t.Fatalf("set event state: %v", err)
	}
	channelKey.ID = "event00"
	err = setChannelEvent(txn, channelKey, dequeuePayload)
	if err != nil {
		t.Fatalf("set event state: %v", err)
	}
	channelKey.Channel = "channel2"
	err = setChannelEvent(txn, channelKey, dequeuePayload)
	if err != nil {
		t.Fatalf("set event state: %v", err)
	}

	// Write actual event
	expected := &Event{
		Topic:        "topic",
		ID:           "event1",
		CreateTime:   time.Now(),
		Payload:      []byte{1, 2, 3},
		DefaultState: EventStateDequeuedOK,
		// Should be ignored.
		State: EventStateDequeuedError,
	}

	err = writeEvent(txn, expected)
	if err != nil {
		t.Fatal("write event: ", err)
	}

	expected.State = EventStateDequeuedOK

	actual, err := getEvent(txn, expected.Topic, expected.ID, "channel")
	if err != nil {
		t.Fatalf("get event on channel: %v", err)
	}
	if !cmp.Equal(actual, expected) {
		t.Errorf("\n%s", cmp.Diff(expected, actual))
	}
	actual, err = getEvent(txn, expected.Topic, expected.ID, "channel2")
	if err != nil {
		t.Fatalf("get event on channel2: %v", err)
	}
	if !cmp.Equal(actual, expected) {
		t.Errorf("\n%s", cmp.Diff(expected, actual))
	}

	expected.State = EventStateQueued

	actual, err = getEvent(txn, expected.Topic, expected.ID, "newchannel")
	if err != nil {
		t.Fatalf("get event on newchannel: %v", err)
	}
	if !cmp.Equal(actual, expected) {
		t.Errorf("\n%s", cmp.Diff(expected, actual))
	}

	err = txn.Commit(nil)
	if err != nil {
		t.Error("commit: ", err)
	}
}

func BenchmarkWriteEvent(b *testing.B) {

	dir, err := ioutil.TempDir("", "test-write-event")
	if err != nil {
		b.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	opts := badger.DefaultOptions
	opts.Dir = dir
	opts.ValueDir = dir
	db, err := badger.Open(opts)
	defer db.Close()
	if err != nil {
		b.Fatal("open db: ", err)
	}

	txn := db.NewTransaction(true)
	defer txn.Discard()

	expected := Event{
		Topic:        "topic",
		ID:           "event1",
		CreateTime:   time.Now(),
		Payload:      []byte{1, 2, 3},
		DefaultState: EventStateDequeuedOK,
		// Should be ignored.
		State: EventStateDequeuedError,
	}

	err = writeEvent(txn, &expected)
	if err != nil {
		b.Fatal("write event: ", err)
	}

	err = txn.Commit(nil)
	if err != nil {
		b.Error("commit: ", err)
	}
}
