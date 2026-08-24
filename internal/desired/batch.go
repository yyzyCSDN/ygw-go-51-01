package desired

import (
	"sync"

	"deviceshadow/internal/model"
	"deviceshadow/internal/shadow"
	shadowsync "deviceshadow/internal/sync"
)

// BatchService delivers groups of desired writes and tracks partial failure
// so the shadow never claims success for devices that did not receive state.
type BatchService struct {
	store    *shadow.Store
	versions *shadowsync.VersionAllocator
	syncsvc  *shadowsync.Service

	mu         sync.Mutex
	retryQueue map[string]*model.Batch
	seq        int64
}

// NewBatchService wires the batch service to the store, allocator and sync.
func NewBatchService(store *shadow.Store, versions *shadowsync.VersionAllocator, syncsvc *shadowsync.Service) *BatchService {
	return &BatchService{
		store:      store,
		versions:   versions,
		syncsvc:    syncsvc,
		retryQueue: make(map[string]*model.Batch),
	}
}

// ApplyBatch commits and delivers every item. When one item fails, the batch
// moves to the partial state, the failed device shadow stays stale and the
// outstanding items are scheduled for retry.
func (s *BatchService) ApplyBatch(items []model.BatchItem, now int64) (*model.Batch, error) {
	s.mu.Lock()
	s.seq++
	batchID := "batch-" + itoa(s.seq)
	s.mu.Unlock()

	batch := model.NewBatch(batchID, items)
	for _, item := range items {
		version := s.versions.Next(item.DeviceID)
		op := model.DesiredOp{
			DeviceID: item.DeviceID,
			Props:    model.CloneMap(item.Props),
			Version:  version,
			Source:   "batch",
		}
		s.store.ApplyDesired(op, now)
		if s.syncsvc.Deliver(item.DeviceID, op) {
			batch.RecordApplied(item.DeviceID)
			continue
		}
		batch.MarkPartial(item.DeviceID)
	}

	if batch.State == model.BatchPartial {
		s.store.MarkBatchPartial(batch, now)
		s.mu.Lock()
		s.retryQueue[batch.ID] = batch
		s.mu.Unlock()
		return batch, nil
	}
	batch.MarkDone()
	return batch, nil
}

// RetryOutstanding re-delivers the failed items of a batch and closes it once
// every item succeeded.
func (s *BatchService) RetryOutstanding(batchID string, now int64) (*model.Batch, bool) {
	s.mu.Lock()
	batch, ok := s.retryQueue[batchID]
	s.mu.Unlock()
	if !ok {
		return nil, false
	}
	outstanding := batch.Outstanding()
	if len(outstanding) == 0 {
		s.mu.Lock()
		delete(s.retryQueue, batchID)
		s.mu.Unlock()
		batch.MarkDone()
		return batch, true
	}
	failedAgain := false
	for _, item := range outstanding {
		version := s.versions.Next(item.DeviceID)
		op := model.DesiredOp{
			DeviceID: item.DeviceID,
			Props:    model.CloneMap(item.Props),
			Version:  version,
			Source:   "batch-retry",
		}
		s.store.ApplyDesired(op, now)
		if s.syncsvc.Deliver(item.DeviceID, op) {
			batch.RecordApplied(item.DeviceID)
			continue
		}
		batch.MarkPartial(item.DeviceID)
		failedAgain = true
	}
	if !failedAgain {
		s.mu.Lock()
		delete(s.retryQueue, batchID)
		s.mu.Unlock()
		batch.MarkDone()
		return batch, true
	}
	s.store.MarkBatchPartial(batch, now)
	return batch, false
}

// PendingBatches returns the ids of batches still waiting for retry.
func (s *BatchService) PendingBatches() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]string, 0, len(s.retryQueue))
	for id := range s.retryQueue {
		ids = append(ids, id)
	}
	return ids
}

func itoa(value int64) string {
	if value == 0 {
		return "0"
	}
	var buf [20]byte
	index := len(buf)
	for value > 0 {
		index--
		buf[index] = byte('0' + value%10)
		value /= 10
	}
	return string(buf[index:])
}
