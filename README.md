# In-Memory Key-Value Database with Queue Operations

A high-performance, concurrent in-memory key-value store with queue operations, implemented using advanced concurrent data structures and lock-free algorithms.

## Quick Start

1. Start the server:
```bash
go run main.go -port <port> -host "<host>"
```
Example:
```bash
go run main.go -port 8080 -host "localhost"
```

2. Make requests:
```bash
curl -X POST http://localhost:<port>/command -H "Content-Type: application/json" -d '{"command": "SET key1 value1"}'
```

All commands are sent as POST requests to the `/command` endpoint.

## Architecture Overview

This project implements an in-memory database supporting both key-value operations and queue operations through a REST API interface. The implementation focuses on high concurrency and performance through careful selection of data structures and synchronization mechanisms.

### Key-Value Store Implementation

The key-value store implementation evolved through several iterations to achieve optimal performance under concurrent access:

1. **Initial Approach: Simple Mutex**
   - Started with a basic mutex-protected map
   - Identified bottlenecks under high concurrent access
   - Found significant overhead due to lock contention

2. **RWMutex Implementation**
   - Upgraded to RWMutex to allow parallel reads
   - Improved read performance significantly
   - Still experienced write contention under heavy load

3. **Final Implementation: Sharded Map**
   - Implemented fine-grained locking using map sharding
   - Each shard has its own RWMutex
   - Benefits:
     - Reduced lock contention
     - Better write performance
     - Improved scalability
     - More granular concurrency control
   - Sharding strategy:
     - Uses hash of key to determine shard
     - Evenly distributes keys across shards

### Queue Implementation

The queue implementation underwent several iterations to achieve optimal concurrent performance:

Implemented with singly **Linked List**

1. **Initial Approach: Mutex-Protected Queue**
   - Basic queue with mutex protection
   - Simple but had high contention under load

2. **RWMutex Queue**
   - Allowed concurrent reads
   - Still had write contention issues

3. **Treiber Stack Implementation**
   - Lock-free implementation
   - Encountered ABA problem under high concurrency
   - ABA Problem: When a thread incorrectly assumes that a value hasn't changed because it sees the same value twice
   - Analogous to coffee shop counter example:
       - Customer visits coffee shop shopkeeper makes coffee.
       - Put it on the counter
       - Customer takes the coffee

4. **Can it be made more faster, YES! : Elimination-Backoff Stack**
   - Advanced lock-free implementation
   - Uses elimination array for direct push/pop exchange
   - Benefits:
     - Significantly reduced contention
     - Higher throughput under load
     - Better scalability
   - How it works:
     - Primary exchange through elimination array
     - Falls back to main stack if elimination fails
     - Analogous to coffee shop counter example:
       - Whatif Both Customer and shopkeeper meets at same time.
       - Shopkeeper is bringing the coffee and Customer demands the coffee
       - Why to use the counter ? simply **Exchange**


## Resources:

 - GopherCon talk on locks : https://www.youtube.com/watch?v=gNQ6j2Y2HFs
 - Karan Jetli Concurrent Stacks : https://www.youtube.com/watch?v=mnCp-mfgFuc
