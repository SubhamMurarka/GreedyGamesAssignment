// File: db.go
package DBCore

import (
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

var (
	ErrBlocked = fmt.Errorf("another client is already waiting on this queue")
	ErrEmpty   = fmt.Errorf("queue is empty")
)

const (
	numShards           = 256
	numEliminationSlots = 8
	eliminationTimeout  = 10 * time.Microsecond
)

// ----------- Entry Types ----------------------

type Entry struct {
	Value     interface{}
	ExpiresAt time.Time
}

type StackNode struct {
	Value interface{}
	Next  *StackNode
}

// ---------Elimination Array ------------

type EliminationArray struct {
	slots [numEliminationSlots]unsafe.Pointer
}

func (e *EliminationArray) exchange(value unsafe.Pointer) unsafe.Pointer {
	index := rand.Intn(numEliminationSlots)
	slot := &e.slots[index]

	previous := atomic.SwapPointer(slot, value)
	if previous != nil {
		atomic.StorePointer(slot, nil)
		return previous
	}

	time.Sleep(eliminationTimeout)
	atomic.CompareAndSwapPointer(slot, value, nil)
	return nil
}

// ------- Treiber Stack ------------

type TreiberStack struct {
	top         unsafe.Pointer
	elimination EliminationArray
}

func (s *TreiberStack) Push(value interface{}) {
	newNode := &StackNode{Value: value}
	valuePtr := unsafe.Pointer(newNode)

	if exchanged := s.elimination.exchange(valuePtr); exchanged == nil {
		for {
			oldTop := (*StackNode)(atomic.LoadPointer(&s.top))
			newNode.Next = oldTop
			if atomic.CompareAndSwapPointer(&s.top, unsafe.Pointer(oldTop), valuePtr) {
				return
			}
		}
	}
}

func (s *TreiberStack) Pop() (interface{}, error) {
	if exchanged := s.elimination.exchange(nil); exchanged != nil {
		return (*StackNode)(exchanged).Value, nil
	}

	for {
		oldTop := (*StackNode)(atomic.LoadPointer(&s.top))
		if oldTop == nil {
			return nil, ErrEmpty
		}
		newTop := oldTop.Next
		if atomic.CompareAndSwapPointer(&s.top, unsafe.Pointer(oldTop), unsafe.Pointer(newTop)) {
			return oldTop.Value, nil
		}
	}
}

// -------- Queue With Blocking Support ----------------------

type QueueWithBlocking struct {
	stack     *TreiberStack
	isBlocked int32
	readLock  sync.Mutex
	signal    chan struct{}
}

func NewQueueWithBlocking() *QueueWithBlocking {
	return &QueueWithBlocking{
		stack:  &TreiberStack{},
		signal: make(chan struct{}, 1),
	}
}

// ------------ Shards ------------------

type Shard struct {
	sync.RWMutex
	data map[string]Entry
}

type QueueShard struct {
	sync.RWMutex
	queues map[string]*QueueWithBlocking
}

// ------------ Database ------------------

type DB struct {
	shards      [numShards]*Shard
	queueShards [numShards]*QueueShard
	ttl         *time.Ticker
	stop        chan bool
}

func NewDB() *DB {
	db := &DB{
		stop: make(chan bool),
		ttl:  time.NewTicker(time.Minute),
	}

	for i := 0; i < numShards; i++ {
		db.shards[i] = &Shard{data: make(map[string]Entry)}
		db.queueShards[i] = &QueueShard{queues: make(map[string]*QueueWithBlocking)}
	}

	go db.cleanup()
	return db
}

func hashKey(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32() % numShards
}

// -------- Shard Getters ----------------

func (db *DB) getShard(key string) *Shard {
	return db.shards[hashKey(key)]
}

func (db *DB) getQueueShard(key string) *QueueShard {
	return db.queueShards[hashKey(key)]
}

// ------- Key-Value Operations -------

func (db *DB) Set(key string, value interface{}, expiration int, nx, xx bool) error {
	shard := db.getShard(key)
	shard.Lock()
	defer shard.Unlock()

	entry, exists := shard.data[key]
	if nx && exists {
		return errors.New("key already exists")
	}
	if xx && !exists {
		return errors.New("key does not exist")
	}
	if !xx && exists {
		return errors.New("key already exists")
	}

	var expiry = time.Now().Add(100 * 365 * 24 * time.Hour) // 100 years

	if xx {
		if expiration != -1 {
			expiry = time.Now().Add(time.Duration(expiration) * time.Second)
		}
		expiry = entry.ExpiresAt
	}

	shard.data[key] = Entry{Value: value, ExpiresAt: expiry}
	return nil
}

func (db *DB) Get(key string) (interface{}, bool) {
	shard := db.getShard(key)
	shard.RLock()
	defer shard.RUnlock()

	entry, exists := shard.data[key]
	if !exists || time.Now().After(entry.ExpiresAt) {
		return nil, false
	}
	return entry.Value, true
}

// ------------- Queue Operations ---------

func (db *DB) Push(key string, value interface{}) {
	shard := db.getQueueShard(key)

	shard.Lock()
	queue, exists := shard.queues[key]
	if !exists {
		queue = NewQueueWithBlocking()
		shard.queues[key] = queue
	}
	queueRef := queue // Get reference under lock
	shard.Unlock()

	queueRef.stack.Push(value)

	// Signal any waiting clients
	if atomic.LoadInt32(&queueRef.isBlocked) == 1 {
		select {
		case queueRef.signal <- struct{}{}:
		default:
		}
	}
}

func (db *DB) Pop(key string) (interface{}, error) {
	shard := db.getQueueShard(key)

	shard.RLock()
	queue, exists := shard.queues[key]
	shard.RUnlock()

	if !exists {
		return nil, errors.New("queue does not exist")
	}

	// Check if someone is waiting in a blocking operation
	if atomic.LoadInt32(&queue.isBlocked) == 1 {
		return nil, ErrBlocked
	}

	// Non-blocking pop
	return queue.stack.Pop()
}

func (db *DB) BQPOP(key string, timeout float64) (interface{}, error) {
	shard := db.getQueueShard(key)

	shard.Lock()
	queue, exists := shard.queues[key]
	if !exists {
		queue = NewQueueWithBlocking()
		shard.queues[key] = queue
	}
	shard.Unlock()

	if !queue.readLock.TryLock() {
		return nil, ErrBlocked
	}
	defer queue.readLock.Unlock()

	// Try immediate pop first
	if value, err := queue.stack.Pop(); err == nil {
		return value, nil
	}

	if timeout <= 0 {
		return nil, ErrEmpty
	}

	atomic.StoreInt32(&queue.isBlocked, 1)
	defer atomic.StoreInt32(&queue.isBlocked, 0)

	timeoutDuration := time.Duration(timeout * float64(time.Second))

	select {
	case <-time.After(timeoutDuration):
		return nil, ErrEmpty
	case <-queue.signal:
		return queue.stack.Pop()
	}
}

// --------------- Cleanup & Close ----------------

func (db *DB) cleanup() {
	for {
		select {
		case <-db.ttl.C:
			now := time.Now()
			for _, shard := range db.shards {
				shard.Lock()
				for key, entry := range shard.data {
					if now.After(entry.ExpiresAt) {
						delete(shard.data, key)
					}
				}
				shard.Unlock()
			}
		case <-db.stop:
			db.ttl.Stop()
			return
		}
	}
}

func (db *DB) Close() {
	db.stop <- true
}
