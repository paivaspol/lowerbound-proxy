// Package lowerboundproxy implements additional capabilities of the proxy.
package lowerboundproxy

import (
	"log"
	"sync"
)

// RequestPriority represents the priority of a request
type RequestPriority int

const (
	High RequestPriority = iota
	Low
)

// ResourceQueue implements two queues for different resources.
type ResourceQueue struct {
	highPriMutex sync.Mutex
	highPriority []chan bool

	// Guards the access of the following fields
	lowPriMutex       sync.Mutex
	lowPriority       map[int]chan bool
	reqIDURLMap       map[int]string
	nextRequestID     int
	nextLowPriorityID int

	destroy chan bool
}

// NewResourceQueue creates a resource queue and starts a routine
// where it will re-prioritize the requests.
func NewResourceQueue() *ResourceQueue {
	rq := &ResourceQueue{
		highPriority:      []chan bool{},
		lowPriority:       make(map[int]chan bool),
		reqIDURLMap:       make(map[int]string),
		nextLowPriorityID: 0,
		nextRequestID:     0,
		destroy:           make(chan bool),
	}
	go func() {
		for {
			// There is something in the high priority queue.
			if len(rq.highPriority) > 0 {
				rq.highPriMutex.Lock()
				nextReqChan := rq.highPriority[0]
				var newHighPriority []chan bool
				if len(rq.highPriority) == 1 {
					newHighPriority = []chan bool{}
				} else {
					newHighPriority = rq.highPriority[1:]
				}
				rq.highPriority = newHighPriority
				rq.highPriMutex.Unlock()

				// Send the signal that to proceed with the request.
				nextReqChan <- true
				continue
			}

			if len(rq.lowPriority) > 0 {
				rq.lowPriMutex.Lock()
				nextReqChan, ok := rq.lowPriority[rq.nextLowPriorityID]
				if !ok {
					rq.lowPriMutex.Unlock()
					continue
				}
				nextReqChan <- true
				rq.nextLowPriorityID++
				rq.lowPriMutex.Unlock()
			}

			select {
			case <-rq.destroy:
				return
			default:
				// pass through
			}
		}
	}()
	return rq
}

// Cleanup cleans the existing state of this resource queue.
func (rq *ResourceQueue) Cleanup() {
	// Make sure to send a signal of all remaining channels in the queues.
	for _, highChan := range rq.highPriority {
		highChan <- true
	}
	for _, lowChan := range rq.lowPriority {
		lowChan <- true
	}

	// Terminate the goroutine
	rq.destroy <- true
}

// QueueRequest places the request on to a queue.
func (rq *ResourceQueue) QueueRequest(rp RequestPriority, url string, signalChan chan bool) {
	log.Printf("[ResourceQueue] queuing: %v with Priority: %v", url, rp)
	switch rp {
	case High:
		// Place this signal channel on the high priority queue.
		rq.highPriMutex.Lock()
		rq.highPriority = append(rq.highPriority, signalChan)
		rq.highPriMutex.Unlock()
	case Low:
		rq.lowPriMutex.Lock()
		reqID := rq.nextRequestID
		rq.nextRequestID++
		rq.lowPriority[reqID] = signalChan
		rq.reqIDURLMap[reqID] = url
		rq.lowPriMutex.Unlock()
	}
}
