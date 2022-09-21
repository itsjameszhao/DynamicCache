package kv

import (
	"sync"
	"sync/atomic"
)

type ReadWriteLatency struct {
	ReadLatency  float64
	WriteLatency float64
}

/*
 * Internal state of the LatencyState
 * Shared across nodes
 */
type LatencyState struct {
	// Read/Write Latency stores latency AGGREGATES
	// SampleCount stores the number of aggregation
	// i.e. Average can be calculated as ReadLatency / Sample Count
	ReadLatency  map[int]float64
	WriteLatency map[int]float64
	SampleCount  map[int]int
}

type LatencyStore struct {
	// Internally we store the state as an atomic pointer, which allows us to
	// swap in a new value with a single atomic operation and avoid locks on
	// most codepaths
	state atomic.Value // stores the LatencyState struct

	// mutex protects the set of `updateChannels` which we send update notifications
	// to whenever U pdate() is called with a new state
	mutex sync.RWMutex
	// `updateChannels` is effectively a set of notification channels -- these channels
	// carry no data, just a signal that is passed whenever the state updates.
	//
	// We store them by an integer key so that they can be removed later
	// if the ShardMapListener is Closed
	updateChannels map[int]chan struct{}
	// Trivial counter which is used to generate unique keys into `updateChannels`
	// whenever a new Listener is created
	updateChannelCounter int
}

/*
 * Gets a pointer to the latest internal state via a thread-safe atomic load.
 */
func (ls *LatencyStore) GetState() *LatencyState {
	state := ls.state.Load().(*LatencyState)
	return state
}

func (ls *LatencyStore) MakeListener() P99Listener {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()
	ch := make(chan struct{})
	id := ls.updateChannelCounter
	if ls.updateChannels == nil {
		ls.updateChannels = make(map[int]chan struct{})
	}

	ls.updateChannelCounter += 1
	ls.updateChannels[id] = ch
	return P99Listener{
		ch:           ch,
		latencyStore: ls,
	}
}

// Pull Latency values from each node. There will be delay
func (ls *LatencyStore) PullLatencyStats() {

	ls.mutex.RLock()
	defer ls.mutex.RUnlock()
	for _, ch := range ls.updateChannels {
		ch <- struct{}{}
	}
}

// Print out the Latency values
func (ls *LatencyStore) ReadLatencyStats() ([]int, []int) {

	var read_latencies []int
	var write_latencies []int

	for key, value := range ls.GetState().SampleCount {
		if _, ok := ls.GetState().ReadLatency[key]; !ok {
			continue
		}
		// I think int will be more readable
		read_latency := int(ls.GetState().ReadLatency[key]) / value
		write_latency := int(ls.GetState().WriteLatency[key]) / value
		read_latencies = append(read_latencies, read_latency)
		write_latencies = append(write_latencies, write_latency)
		ls.GetState().ReadLatency[key] = 0
		ls.GetState().WriteLatency[key] = 0
		ls.GetState().SampleCount[key] = 0
	}

	return read_latencies, write_latencies
}

func (ls *LatencyStore) Initialize(state *LatencyState) {

	ls.state.Swap(state)
	ls.mutex.RLock()
	defer ls.mutex.RUnlock()
	for _, ch := range ls.updateChannels {
		ch <- struct{}{}
	}
}

/*
 * A single listener for updates to a given LatencyState. Updates
 * are given as notifications on a dummy channel.
 *
 * To use the updated value, simply read it from the underlying
 * LatencyState
 *
 * P99Listener.Close() must be called to safely stop
 * listening for updates.
 */
type P99Listener struct {
	latencyStore *LatencyStore
	ch           chan struct{}
}

func (listener *P99Listener) UpdateChannel() chan struct{} {
	return listener.ch
}

func (listener *P99Listener) UpdateP99(latencyMap map[int]ReadWriteLatency) {
	listener.latencyStore.mutex.Lock()
	defer listener.latencyStore.mutex.Unlock()

	for key, value := range latencyMap {
		if value.ReadLatency != value.ReadLatency || value.WriteLatency != value.WriteLatency {
			// Check if value == NaN
			continue
		}
		listener.latencyStore.GetState().ReadLatency[key] += value.ReadLatency
		listener.latencyStore.GetState().WriteLatency[key] += value.WriteLatency
		listener.latencyStore.GetState().SampleCount[key] += 1
	}

}

func (listener *P99Listener) Close() {
	close(listener.ch)
}
