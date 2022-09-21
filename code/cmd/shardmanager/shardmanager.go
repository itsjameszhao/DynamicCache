package main

import (
	"flag"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"cs426.yale.edu/lab4/kv"
	kvtest "cs426.yale.edu/lab4/kv/test"
	"github.com/sirupsen/logrus"
)

/********************************
This file contains the logic for the Shard Manager in our paper.
Using the functions written in p99.go it is responsible
for querying each node for its latency and sending shard
move commands via shard state updates.

The complete system is set up here (all in one process).
In main() below, the nodes, shard manager, and load testing client
are all initialized and the client starts sending requests according
to a time-varying pattern.
********************************/

var (
	num_goros    = flag.Int("num-goros", 16, "Number of goroutines")
	enabled      = flag.Bool("enabled", false, "Whether dynamic shard management is enabled; false degenerates to lab 4")
	numshards    = flag.Int("numshards", 5, "Number of shards for our system")
	numnodes     = flag.Int("numnodes", 5, "Number of nodes for our system")
	max_replicas = flag.Int("max-replicas", 5, "Maximum number of replicas per shard")
	latency_max  = flag.Float64("latency-max", 15000, "Maximum latency (nanoseconds) before another shard replica is created")
	latency_min  = flag.Float64("latency-min", 10000, "Minimum latency (nanoseconds) before a shard replica is removed")
)

// Add a shard replica to our system, allocated randomly to
// nodes which do not yet have that shard.
func add_shard_replica(shard int, setup *kvtest.TestSetup) {

	state := setup.ShardMap.GetState()
	var candidates []string

	// Find the nodes which do not have the shard replica
	for node := range state.Nodes {
		for _, node2 := range state.ShardsToNodes[shard] {
			if node == node2 {
				break
			}
		}
		candidates = append(candidates, node)
	}

	// Randomly choose a node to add the shard replica to
	candidate := candidates[rand.Intn(len(candidates))]
	state.ShardsToNodes[shard] = append(state.ShardsToNodes[shard], candidate)

	// Update the ShardMap and propagate changes over to the listener
	setup.ShardMap.Update(state)

	// logrus.Infof("Added shard %v to node %s", shard, candidate)
	// logrus.Infof("Shardmap state is %v")
}

// Drop a shard replica from our system, chosen in reverse order
func drop_shard_replica(shard int, setup *kvtest.TestSetup) {
	state := setup.ShardMap.GetState()
	curr_nodes := state.ShardsToNodes[shard]
	if len(curr_nodes) <= 1 {
		logrus.Fatalf("Can't remove last shard or no shard replicas left!")
	}

	// Remove the last shard
	// logrus.Infof("Dropping shard %v from node %s", shard, curr_nodes[len(curr_nodes)-1])

	new_nodes := curr_nodes[0 : len(curr_nodes)-1]
	state.ShardsToNodes[shard] = new_nodes

	// Update the ShardMap and propagate changes over to the listener
	setup.ShardMap.Update(state)

}

// the heart of the shard manager: it polls the latencies,
// determines what needs to be updated, and actually sends
// out the signals to update everything.
func update_shards(ls *kv.LatencyStore, setup *kvtest.TestSetup) {
	// get latencies
	ls.PullLatencyStats()
	time.Sleep(time.Second)
	read_latencies, write_latencies := ls.ReadLatencyStats()
	logrus.Infof("Read latencies are %v Write latencies are %v", read_latencies, write_latencies)

	// determine what to do with the shards, if anything, if enabled

	if *enabled {
		for i := 0; i < *numshards; i++ {
			shard_latency := float64(read_latencies[i])
			num_replicas := len(setup.ShardMap.GetState().ShardsToNodes[i+1]) // Shards are 1-indexed
			if shard_latency > *latency_max && num_replicas < *max_replicas {
				add_shard_replica(i+1, setup) // Shards are 1-indexed
			} else if shard_latency < *latency_min && num_replicas > 1 {
				drop_shard_replica(i+1, setup)
			}
			// logrus.Infof("i: %d, num_replicas: %d\n", i, len(setup.ShardMap.GetState().ShardsToNodes[i+1]))
		}
	}
	// logrus.Infof("Adding shard replicas: ")
}

// run the shard manager to rebalance.
func main() {
	flag.Parse()
	rand.Seed(0)
	// Check to make sure number of replicas is not too high
	if *max_replicas > *numnodes {
		*max_replicas = *numnodes
		logrus.Warnf("Warning: number of replicas exceeds number of nodes. (Fixed)")
	}

	// Create and initialize the nodes.
	state := kvtest.MakeManyNodesWithManyShardsSimple(*numshards, *numnodes)
	setup := kvtest.MakeTestSetup(state)

	// Create our latency monitor
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

	// If dynamic shard management is enabled,
	// start the shard management algorithm
	go func() {
		time.Sleep(time.Second)
		for {
			// logrus.Infof("Updating shards")
			update_shards(&ls, setup)
			// logrus.Infof("Sleeping")
		}
	}()

	// LOAD TEST #1: Uniform distribution, no ShardManager
	keys := kvtest.RandomKeys(100, 10)
	vals := kvtest.RandomKeys(100, 1024)
	for i, k := range keys {
		val := vals[i]
		setup.Set(k, val, 3*time.Minute)
	}

	goros := *num_goros

	var wg sync.WaitGroup
	readIters := 2000
	writeIters := 20

	start_time := time.Now()
	num_requests := (readIters + writeIters) * 100 * goros

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
	logrus.Info("Time elapsed %v seconds, Number of requests %v, throughput (RPS) %v", time.Since(start_time).Seconds(), num_requests, float64(num_requests)/time.Since(start_time).Seconds())
	setup.Shutdown()

}
