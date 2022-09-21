package kv

import (
	"context"
	"math/rand"
	"sync"
	"time"

	pb "cs426.yale.edu/lab4/kv/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Kv is the client data structure for finding, writing to, and
// querying shards on the server side.
type Kv struct {
	shardMap   *ShardMap
	clientPool ClientPool

	// Add any client-side state you want here
}

func MakeKv(shardMap *ShardMap, clientPool ClientPool) *Kv {
	kv := &Kv{
		shardMap:   shardMap,
		clientPool: clientPool,
	}
	// Add any initialization logic
	return kv
}

func (kv *Kv) Get(ctx context.Context, key string) (string, bool, error) {
	// Trace-level logging -- you can remove or use to help debug in your tests
	// with `-log-level=trace`. See the logging section of the spec.
	logrus.WithFields(
		logrus.Fields{"key": key},
	).Trace("client sending Get() request")

	shard := GetShardForKey(key, kv.shardMap.NumShards())
	nodes := kv.shardMap.GetState().ShardsToNodes[shard]
	if len(nodes) == 0 {
		return "", false, status.Errorf(
			codes.NotFound,
			"node not found for shard %v",
			shard,
		)
	}

	rand_index := rand.Intn(len(nodes))
	var lasterr error

	for i := 0; i < len(nodes); i++ {
		selected_node := nodes[(rand_index+i)%len(nodes)] // B2 approach 1: random load. Try next node until all tried if err.
		client, err := kv.clientPool.GetClient(selected_node)
		if err != nil {
			logrus.Debugf("Unable to create a KvClient for node %v!", selected_node)
			lasterr = err
			continue
		}

		// Get the key from the correct client.
		response, err := client.Get(ctx, &pb.GetRequest{Key: key})
		if err != nil {
			logrus.Debugf("Client GetRequest failed for node %v, key %v", selected_node, key)
			lasterr = err
			continue
		}

		// Not in cache
		if !response.WasFound {
			return "", false, nil
		}

		return response.Value, true, nil
		//panic("TODO: Part B")
	}

	return "", false, lasterr

}

func (kv *Kv) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	logrus.WithFields(
		logrus.Fields{"key": key},
	).Trace("client sending Set() request")

	shard := GetShardForKey(key, kv.shardMap.NumShards())
	nodes := kv.shardMap.GetState().ShardsToNodes[shard]
	if len(nodes) == 0 {
		return status.Errorf(
			codes.NotFound,
			"node not found for shard %v",
			shard,
		)
	}

	errchan := make(chan error, 2*len(nodes)) // potentially 2 * # nodes errors
	rand_index := rand.Intn(len(nodes))
	var wg sync.WaitGroup

	for i := 0; i < len(nodes); i++ {
		wg.Add(1)
		go func(i int, errchan chan error) {
			defer wg.Done()
			selected_node := nodes[(rand_index+i)%len(nodes)] // Randomize to spread load more evenly
			client, err := kv.clientPool.GetClient(selected_node)
			if err != nil || client == nil {
				errchan <- status.Errorf(codes.NotFound,
					"Unable to create a KvClient for node %v!",
					selected_node)
				return
			}

			// Get the key from the correct client.
			_, err = client.Set(ctx, &pb.SetRequest{Key: key, Value: value, TtlMs: ttl.Milliseconds()})
			if err != nil {
				errchan <- status.Errorf(codes.Unavailable,
					"Unable to set key %v and value %v for node %v!",
					key,
					value,
					selected_node)
			}
		}(i, errchan)
	}

	wg.Wait() // wait for requests to finish

	// check if channel non-empty, if not return the first error message
	if len(errchan) > 0 {
		return <-errchan
	}

	return nil
	//panic("TODO: Part B")
}

func (kv *Kv) Delete(ctx context.Context, key string) error {
	logrus.WithFields(
		logrus.Fields{"key": key},
	).Trace("client sending Delete() request")
	shard := GetShardForKey(key, kv.shardMap.NumShards())
	nodes := kv.shardMap.GetState().ShardsToNodes[shard]
	if len(nodes) == 0 {
		return status.Errorf(
			codes.NotFound,
			"node not found for shard %v",
			shard,
		)
	}

	errchan := make(chan error, 2*len(nodes)) // potentially 2 * # nodes errors
	rand_index := rand.Intn(len(nodes))
	var wg sync.WaitGroup

	for i := 0; i < len(nodes); i++ {
		wg.Add(1)
		go func(i int, errchan chan error) {
			defer wg.Done()
			selected_node := nodes[(rand_index+i)%len(nodes)] // Randomize to spread load more evenly
			client, err := kv.clientPool.GetClient(selected_node)
			if err != nil || client == nil {
				errchan <- status.Errorf(codes.NotFound,
					"Unable to create a KvClient for node %v!",
					selected_node)
				return
			}

			// Get the key from the correct client.
			_, err = client.Delete(ctx, &pb.DeleteRequest{Key: key})
			if err != nil {
				errchan <- status.Errorf(codes.Unavailable,
					"Unable to delete key %v for node %v!",
					key,
					selected_node)
			}
		}(i, errchan)
	}

	wg.Wait() // wait for requests to finish

	// check if channel non-empty, if not return the first error message
	if len(errchan) > 0 {
		return <-errchan
	}

	return nil

	//panic("TODO: Part B")
}
