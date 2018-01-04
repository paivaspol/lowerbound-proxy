// Package lowerboundproxy implements additional capabilities of the proxy.
package lowerboundproxy

import (
	"bufio"
	"log"
	"math"
	"os"
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

	requestOrderMutex sync.Mutex
	requestOrder      map[string]int
	curRequestPos     int

	destroy chan bool
}

// NewResourceQueue creates a resource queue and starts a routine
// where it will re-prioritize the requests.
func NewResourceQueue(requestOrderFile string) (*ResourceQueue, error) {
	requestOrder, err := getRequestOrder(requestOrderFile)
	if err != nil {
		return nil, err
	}
	rq := &ResourceQueue{
		highPriority:      []chan bool{},
		lowPriority:       make(map[int]chan bool),
		reqIDURLMap:       make(map[int]string),
		nextLowPriorityID: 0,
		nextRequestID:     0,
		requestOrder:      requestOrder,
		curRequestPos:     0,
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
					// Remove the first element.
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
				delete(rq.lowPriority, rq.nextLowPriorityID)
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
	return rq, nil
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
	if position, ok := rq.requestOrder[url]; ok {
		rq.requestOrderMutex.Lock()
		rq.curRequestPos = int(math.Max(float64(rq.curRequestPos), float64(position)))
		rq.reprioritize()
		rq.requestOrderMutex.Unlock()
	}
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

// getRequestOrder parses the file from the given file name and returns a mapping of
// URLs to the request order.
func getRequestOrder(scheduleFile string) (map[string]int, error) {
	file, err := os.Open(scheduleFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	requestOrder := make(map[string]int)
	scanner := bufio.NewScanner(file)
	counter := 0
	for scanner.Scan() {
		requestOrder[scanner.Text()] = counter
		counter++
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return requestOrder, nil
}

// reprioritize moves resources from low priority to high priority. This only happens when
// low priority resources supposed to already be discovered by the browser based on
// a previously discovered schedule.
func (rq *ResourceQueue) reprioritize() {
	rq.lowPriMutex.Lock()
	defer rq.lowPriMutex.Unlock()
	moveSet := []int{}
	for reqID, _ := range rq.lowPriority {
		url, ok := rq.reqIDURLMap[reqID]
		if !ok {
			// For some reason, we cannot find the associated URL for this reqID.
			// Proceed to the next URL.
			continue
		}
		if requestOrder, ok := rq.requestOrder[url]; ok {
			log.Printf("Reprioritizing: %v", url)
			if requestOrder < rq.curRequestPos {
				moveSet = append(moveSet, reqID)
			}
		}
	}

	rq.highPriMutex.Lock()
	defer rq.highPriMutex.Unlock()
	// Move the low priority channels.
	for _, reqToMove := range moveSet {
		rq.highPriority = append(rq.highPriority, rq.lowPriority[reqToMove])
		delete(rq.lowPriority, reqToMove)
	}
}
