package store

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"confighub/internal/model"
)

func checksumOf(revision int64, batchID string, entries map[model.Key]string) string {
	h := sha256.New()
	fmt.Fprintf(h, "rev=%d\nbatch=%s\n", revision, batchID)
	for _, k := range model.CanonicalKeys(entries) {
		h.Write([]byte(model.KeyLine(k, entries[k])))
		h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func ChecksumOfSnapshot(s *model.Snapshot) string {
	if s == nil {
		return ""
	}
	return checksumOf(s.Revision, s.BatchID, s.Entries)
}

func ChecksumFor(revision int64, batchID string, entries map[model.Key]string) string {
	return checksumOf(revision, batchID, entries)
}
