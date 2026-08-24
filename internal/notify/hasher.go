package notify

import "github.com/cespare/xxhash/v2"

// Hasher maps subscription topics to fixed shard buckets so the console can
// show a stable grouping of active subscriptions.
type Hasher struct {
	shards uint64
}

// NewHasher builds a topic hasher with the given shard count.
func NewHasher(shards uint64) *Hasher {
	if shards == 0 {
		shards = 4
	}
	return &Hasher{shards: shards}
}

// Bucket returns the shard index for a topic key.
func (h *Hasher) Bucket(topic string) uint64 {
	return xxhash.Sum64String(topic) % h.shards
}

// Shards returns the configured shard count.
func (h *Hasher) Shards() uint64 {
	return h.shards
}

// TopicKey builds the subscription topic for a device.
func TopicKey(deviceID string) string {
	return "shadow/" + deviceID
}
