package kvtest

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"

	"cs426.yale.edu/lab4/kv"
)

// Run the test individually.
func TestP99Basic(t *testing.T) {

	setup := MakeTestSetup(MakeBasicOneShard())

	// Latency Store:
	ls := kv.LatencyStore{}
	lstate := kv.LatencyState{
		ReadLatency:  make(map[int]float64),
		WriteLatency: make(map[int]float64),
		SampleCount:  make(map[int]int),
	}
	ls.Initialize(&lstate)
	for name := range setup.ShardMap.Nodes() {
		setup.Nodes[name].AddLatencyListener(&ls)
	}

	// Main test
	setup.NodeGet("n1", "abc")
	setup.NodeSet("n1", "abc", "123", 5*time.Second)
	setup.NodeGet("n1", "abc")

	ls.PullLatencyStats()
	time.Sleep(time.Second)
	ls.ReadLatencyStats()

	setup.Shutdown()

}

func TestP99MultiShard(t *testing.T) {

	setup := MakeTestSetup(MakeMultiShardSingleNode())

	// Latency Store:
	ls := kv.LatencyStore{}
	lstate := kv.LatencyState{
		ReadLatency:  make(map[int]float64),
		WriteLatency: make(map[int]float64),
		SampleCount:  make(map[int]int),
	}
	ls.Initialize(&lstate)
	for name := range setup.ShardMap.Nodes() {
		setup.Nodes[name].AddLatencyListener(&ls)
	}

	// Main test
	keys := RandomKeys(100, 10)
	vals := RandomKeys(100, 1024*1024)
	for i, k := range keys {
		val := vals[i]
		setup.NodeSet("n1", k, val, 10*time.Second)
	}

	for _, k := range keys {
		setup.NodeGet("n1", k)
	}

	ls.PullLatencyStats()
	time.Sleep(time.Second)
	ls.ReadLatencyStats()
	setup.Shutdown()

}

func TestP99TwoNodeMultiShard(t *testing.T) {

	setup := MakeTestSetup(MakeTwoNodeMultiShard())

	// Latency Store:
	ls := kv.LatencyStore{}
	lstate := kv.LatencyState{
		ReadLatency:  make(map[int]float64),
		WriteLatency: make(map[int]float64),
		SampleCount:  make(map[int]int),
	}
	ls.Initialize(&lstate)
	for name := range setup.ShardMap.Nodes() {
		setup.Nodes[name].AddLatencyListener(&ls)
	}

	// Main test
	keys := RandomKeys(100, 10)
	vals := RandomKeys(100, 1024*1024)
	keysToOriginalNode := make(map[string]string)

	for i, key := range keys {
		err := setup.NodeSet("n1", key, vals[i], 10*time.Second)
		if err == nil {
			keysToOriginalNode[key] = "n1"
		} else {
			setup.NodeSet("n2", key, vals[i], 10*time.Second)
			keysToOriginalNode[key] = "n2"
		}
	}

	for _, key := range keys {
		setup.NodeGet(keysToOriginalNode[key], key)
	}

	ls.PullLatencyStats()
	time.Sleep(time.Second)
	ls.ReadLatencyStats()
	setup.Shutdown()
}

func TestP99FourNodesFiveShard(t *testing.T) {

	setup := MakeTestSetup(MakeFourNodesWithFiveShards())

	// Latency Store:
	ls := kv.LatencyStore{}
	lstate := kv.LatencyState{
		ReadLatency:  make(map[int]float64),
		WriteLatency: make(map[int]float64),
		SampleCount:  make(map[int]int),
	}
	ls.Initialize(&lstate)
	for name := range setup.ShardMap.Nodes() {
		setup.Nodes[name].AddLatencyListener(&ls)
	}

	// Main test
	keys := RandomKeys(100, 10)
	vals := RandomKeys(100, 1024*1024)
	keysToOriginalNode := make(map[string]string)
	node_list := [4]string{"n1", "n2", "n3", "n4"}
	for i, key := range keys {
		rand_int := rand.Intn(4)
		node_name := node_list[rand_int]
		setup.NodeSet(node_name, key, vals[i], 10*time.Second)
		keysToOriginalNode[key] = node_name
	}

	for _, key := range keys {
		setup.NodeGet(keysToOriginalNode[key], key)
	}

	ls.PullLatencyStats()
	time.Sleep(time.Second)
	ls.ReadLatencyStats()
	setup.Shutdown()
}

func TestP99ConcurrentGetsAndSets(t *testing.T) {

	// Your server must be able to handle concurrent reads and writes,
	// though you may need to use exclusive locks to handle writes
	// (per-stripe or per-shard preferably).

	keys := RandomKeys(200, 10)
	setup := MakeTestSetup(MakeMultiShardSingleNode())

	// Latency Store:
	ls := kv.LatencyStore{}
	lstate := kv.LatencyState{
		ReadLatency:  make(map[int]float64),
		WriteLatency: make(map[int]float64),
		SampleCount:  make(map[int]int),
	}
	ls.Initialize(&lstate)
	for name := range setup.ShardMap.Nodes() {
		setup.Nodes[name].AddLatencyListener(&ls)
	}

	for _, k := range keys {
		setup.NodeSet("n1", k, "abcd", 100*time.Second)
	}

	var wg sync.WaitGroup
	goros := runtime.NumCPU() * 2
	readIters := 200
	writeIters := 100
	for i := 0; i < goros; i++ {
		wg.Add(2)
		// start a writing goro
		go func() {
			for j := 0; j < writeIters; j++ {
				for _, k := range keys {
					setup.NodeSet("n1", k, fmt.Sprintf("abc:%d", j), 100*time.Second)
				}
			}
			wg.Done()
		}()
		// start a reading goro
		go func() {
			for j := 0; j < readIters; j++ {
				for _, k := range keys {
					setup.NodeGet("n1", k)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()

	ls.PullLatencyStats()
	time.Sleep(time.Second)
	ls.ReadLatencyStats()
	setup.Shutdown()

}

func TestP99ConcurrentGetsAndSets2(t *testing.T) {

	// Your server must be able to handle concurrent reads and writes,
	// though you may need to use exclusive locks to handle writes
	// (per-stripe or per-shard preferably).

	keys := RandomKeys(200, 10)
	setup := MakeTestSetup(MakeManyNodesWithManyShardsSimple(5, 5))

	// Latency Store:
	ls := kv.LatencyStore{}
	lstate := kv.LatencyState{
		ReadLatency:  make(map[int]float64),
		WriteLatency: make(map[int]float64),
		SampleCount:  make(map[int]int),
	}
	ls.Initialize(&lstate)
	for name := range setup.ShardMap.Nodes() {
		setup.Nodes[name].AddLatencyListener(&ls)
	}

	for _, k := range keys {
		setup.Set(k, "abcd", 100*time.Second)
	}

	var wg sync.WaitGroup
	goros := runtime.NumCPU() * 32
	readIters := 200
	writeIters := 100
	for i := 0; i < goros; i++ {
		wg.Add(2)
		// start a writing goro
		go func() {
			for j := 0; j < writeIters; j++ {
				for _, k := range keys {
					setup.Set(k, fmt.Sprintf("abc:%d", j), 100*time.Second)
				}
			}
			wg.Done()
		}()
		// start a reading goro
		go func() {
			for j := 0; j < readIters; j++ {
				for _, k := range keys {
					setup.Get(k)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()

	ls.PullLatencyStats()
	time.Sleep(time.Second)
	ls.ReadLatencyStats()
	setup.Shutdown()

}
