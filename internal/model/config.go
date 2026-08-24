package model

import (
	"sort"
	"strings"
	"time"
)

type AppID string

type GroupID string

type Key string

type Entry struct {
	Key     Key
	Value   string
	Version int64
}

type Snapshot struct {
	App        AppID
	Group      GroupID
	Revision   int64
	BatchID    string
	Entries    map[Key]string
	Checksum   string
	CapturedAt time.Time
}

func CopyEntries(src map[Key]string) map[Key]string {
	out := make(map[Key]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func CanonicalKeys(m map[Key]string) []Key {
	keys := make([]Key, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

func KeyLine(k Key, v string) string {
	return string(k) + "=" + v
}

func NamespaceKey(app AppID, group GroupID) string {
	return strings.ToLower(string(app)) + "/" + strings.ToLower(string(group))
}
