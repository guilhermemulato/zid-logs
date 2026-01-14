package state

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)

const checkpointBucket = "checkpoints"
const keySeparator = "\x1f"

type FileIdentity struct {
	Dev   uint64 `json:"dev"`
	Inode uint64 `json:"inode"`
}

type Checkpoint struct {
	Package        string       `json:"package"`
	LogID          string       `json:"log_id"`
	Path           string       `json:"path"`
	Identity       FileIdentity `json:"identity"`
	LastOffset     int64        `json:"last_offset"`
	LastSentAt     int64        `json:"last_sent_at"`
	LastError      string       `json:"last_error"`
	LastAttemptAt  int64        `json:"last_attempt_at"`
	LastStatusCode int          `json:"last_status_code"`
	LastBytesSent  int64        `json:"last_bytes_sent"`
	LastDurationMs int64        `json:"last_duration_ms"`
	LastRotateAt   int64        `json:"last_rotate_at"`
}

type State struct {
	path     string
	db       *bolt.DB
	readOnly bool
}

func Open(path string) (*State, error) {
	return openWithOptions(path, &bolt.Options{Timeout: time.Second})
}

func OpenReadOnly(path string) (*State, error) {
	return openWithOptions(path, &bolt.Options{ReadOnly: true, Timeout: time.Second})
}

func openWithOptions(path string, options *bolt.Options) (*State, error) {
	db, err := bolt.Open(path, 0600, options)
	if err != nil {
		return nil, err
	}

	st := &State{path: path, db: db, readOnly: options != nil && options.ReadOnly}
	if !st.readOnly {
		if err := st.ensureBuckets(); err != nil {
			_ = db.Close()
			return nil, err
		}
	}

	return st, nil
}

func (s *State) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *State) GetCheckpoint(pkg, logID, path string) (Checkpoint, bool, error) {
	var cp Checkpoint
	key := checkpointKey(pkg, logID, path)

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(checkpointBucket))
		if bucket == nil {
			if s.readOnly {
				return nil
			}
			return fmt.Errorf("bucket ausente: %s", checkpointBucket)
		}
		data := bucket.Get([]byte(key))
		if data == nil {
			return nil
		}
		return json.Unmarshal(data, &cp)
	})
	if err != nil {
		return Checkpoint{}, false, err
	}
	if cp.Package == "" && cp.LogID == "" && cp.Path == "" {
		return Checkpoint{}, false, nil
	}
	return cp, true, nil
}

func (s *State) SaveCheckpoint(cp Checkpoint) error {
	key := checkpointKey(cp.Package, cp.LogID, cp.Path)

	data, err := json.Marshal(cp)
	if err != nil {
		return err
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(checkpointBucket))
		if bucket == nil {
			return fmt.Errorf("bucket ausente: %s", checkpointBucket)
		}
		return bucket.Put([]byte(key), data)
	})
}

func (s *State) ensureBuckets() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(checkpointBucket))
		return err
	})
}

func checkpointKey(pkg, logID, path string) string {
	parts := []string{pkg, logID, path}
	return strings.Join(parts, keySeparator)
}
