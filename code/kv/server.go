package kv

import (
	"context"
	"math/rand"

	"sync"
	"time"

	"cs426.yale.edu/lab4/kv/proto"
	"github.com/influxdata/tdigest"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type value_TTL struct {
	value       string
	expire_time uint64
}

// Key-value server implementation
type KvServerImpl struct {
	proto.UnimplementedKvServer
	nodeName string

	// Server Constants
	shardMap   *ShardMap
	listener   *ShardMapListener
	clientPool ClientPool
	shutdown   chan struct{}

	// Key Value Store
	kv_locks []sync.RWMutex
	kv_store []map[string]value_TTL

	// Key Value Store Metadata
	data_mu     sync.RWMutex
	shardActive map[int]bool
	// keyShardIndex map[string]int

	// Server Metadata
	p99Read_mu      sync.RWMutex
	p99Write_mu     sync.RWMutex
	shardToP99Read  map[int]*tdigest.TDigest
	shardToP99Write map[int]*tdigest.TDigest
	p99_listener    *P99Listener
}

func (server *KvServerImpl) sendP99Info() {

}

func (server *KvServerImpl) handleShardMapUpdate() {

	server.data_mu.Lock()
	//1. Determine the new shards for this node
	newShards := server.listener.shardMap.ShardsForNode(server.nodeName)
	//2. Old Shards
	oldShardMap := server.shardActive

	//3. Determine what needs to be added and deleted
	newShardMap := make(map[int]bool, 0)
	for i := 0; i < len(newShards); i++ {
		newShardMap[newShards[i]] = true
	}

	//determine what is needed for add
	addCand := make([]int, 0)
	for k := range newShardMap {
		if !oldShardMap[k] {
			addCand = append(addCand, k)
		}
	}

	//4. Add New Key
	//Step 1. Determine the correct client using the client pool
	for i := 0; i < len(addCand); i++ {
		cand := addCand[i]
		shardIndex := cand - 1
		nodes := server.shardMap.GetState().ShardsToNodes[cand]

		// Load balancing step
		rand_index := rand.Intn(len(nodes))
		var lasterror error
		//Step 2. Attempt to connect to client, if not continue to connect through all nodes
		for k := 0; k < len(nodes); k++ {
			selected_node := nodes[(rand_index+k)%len(nodes)]
			if selected_node != server.nodeName { //make sure we don't connect to current server
				client, err := server.clientPool.GetClient(selected_node)
				if err != nil {
					logrus.Debugf("Unable to create a ServerClient for node %v!", selected_node)
					lasterror = err
					continue
				}
				server.data_mu.Unlock()
				new_values, err := client.GetShardContents(context.Background(), &proto.GetShardContentsRequest{Shard: int32(cand)})
				server.data_mu.Lock()
				if err != nil {
					logrus.Debugf("Unable to retrieve contents from shard, attempt another node")
					lasterror = err
					continue
				}
				server.kv_locks[shardIndex].Lock()
				toUpdate := new_values.Values
				for j := 0; j < len(toUpdate); j++ {
					new_key := toUpdate[j].Key
					new_value := toUpdate[j].Value
					time_remaining := toUpdate[j].TtlMsRemaining
					server.kv_store[cand-1][new_key] = value_TTL{value: new_value, expire_time: uint64(time_remaining)}
					// server.keyShardIndex[new_key] = cand - 1
				}

				server.kv_locks[shardIndex].Unlock()
			}
		}
		if lasterror != nil {
			logrus.Debugf("Unable to connect to a server client, intializing shard as empty")
		}
	}
	//6. Update new shard map
	server.shardActive = newShardMap
	server.data_mu.Unlock()

	//determine what to delete
	deleteCand := make([]int, 0)
	for k := range oldShardMap {
		if !newShardMap[k] {
			deleteCand = append(deleteCand, k)
		}
	}

	//5. Delete Values from associated with shard from KV Store
	for i := 0; i < len(deleteCand); i++ {
		cand := deleteCand[i]
		server.kv_locks[cand-1].Lock()
		for k := range server.kv_store[cand-1] {
			shard_idx := GetShardForKey(k, server.shardMap.NumShards())
			if shard_idx == cand {
				// log.Println("delete", cand-1, k)
				delete(server.kv_store[cand-1], k)
			}
		}
		server.kv_locks[cand-1].Unlock()
	}
}

func (server *KvServerImpl) shardMapListenLoop() {
	listener := server.listener.UpdateChannel()
	for {
		select {
		case <-server.shutdown:
			break
		case <-listener:
			server.handleShardMapUpdate()
		}
	}
}

func (server *KvServerImpl) P99ListenLoop() {
	listener := server.p99_listener.UpdateChannel()
	for {
		select {
		case <-server.shutdown:
			break
		case <-listener:

			latencyMap := make(map[int]ReadWriteLatency)
			server.p99Read_mu.RLock()
			server.p99Write_mu.RLock()
			for key, boolean := range server.shardActive {
				if boolean {
					latencyMap[key] = ReadWriteLatency{server.shardToP99Read[key].Quantile(0.99), server.shardToP99Write[key].Quantile(0.99)}
				}
			}
			server.p99Read_mu.RUnlock()
			server.p99Write_mu.RUnlock()
			server.p99_listener.UpdateP99(latencyMap)
		}
	}
}

// Checks if node has a shard
func (server *KvServerImpl) CheckCorrectShard(request_string string) bool {
	server.data_mu.RLock()
	defer server.data_mu.RUnlock()

	shardIndex := GetShardForKey(request_string, server.shardMap.NumShards())

	return server.shardActive[shardIndex]
}

func (server *KvServerImpl) Clean() {

	for {
		time.Sleep(1 * time.Second)
		server.data_mu.Lock()
		for i := 0; i < len(server.kv_store); i++ {
			server.kv_locks[i].Lock()
			for key, value := range server.kv_store[i] {
				if value.expire_time < uint64(time.Now().UnixMilli()) {
					delete(server.kv_store[i], key)
				}
			}
			server.kv_locks[i].Unlock()
		}
		server.data_mu.Unlock()
	}
}

func MakeKvServer(nodeName string, shardMap *ShardMap, clientPool ClientPool) *KvServerImpl {
	listener := shardMap.MakeListener()
	server := KvServerImpl{
		nodeName:    nodeName,
		shardMap:    shardMap,
		listener:    &listener,
		clientPool:  clientPool,
		shutdown:    make(chan struct{}),
		kv_locks:    make([]sync.RWMutex, shardMap.NumShards()),
		kv_store:    make([]map[string]value_TTL, shardMap.NumShards()),
		shardActive: make(map[int]bool),
		// keyShardIndex:   make(map[string]int),
		shardToP99Read:  make(map[int]*tdigest.TDigest),
		shardToP99Write: make(map[int]*tdigest.TDigest),
	}

	for i := 0; i < len(server.kv_store); i++ {
		server.kv_store[i] = make(map[string]value_TTL)
	}

	// Initialize shardToP99 (1-indexed)
	for i := 0; i < server.shardMap.NumShards(); i++ {
		server.shardToP99Read[i+1] = tdigest.NewWithCompression(1000)
		server.shardToP99Write[i+1] = tdigest.NewWithCompression(1000)
	}

	go server.shardMapListenLoop()
	server.handleShardMapUpdate()
	go server.Clean()

	return &server
}

// Add latency listener to this node
func (server *KvServerImpl) AddLatencyListener(ls *LatencyStore) {
	p99_ls := ls.MakeListener()
	server.p99_listener = &p99_ls
	go server.P99ListenLoop()
}

func (server *KvServerImpl) Shutdown() {
	server.shutdown <- struct{}{}
	server.listener.Close()
}

func (server *KvServerImpl) Get(
	ctx context.Context,
	request *proto.GetRequest,
) (*proto.GetResponse, error) {
	// Trace-level logging for node receiving this request (enable by running with -log-level=trace),
	// feel free to use Trace() or Debug() logging in your code to help debug tests later without
	// cluttering logs by default. See the logging section of the spec.
	start := time.Now().UnixNano()

	//1. Check if Key is Valid
	var response proto.GetResponse
	if request.Key == "" {
		response.Value = ""
		response.WasFound = false
		return &response, status.Error(
			codes.InvalidArgument,
			"empty key",
		)
	}
	//2. Check if current shard is active
	if !server.CheckCorrectShard(request.Key) {
		response.Value = ""
		response.WasFound = false
		return &response, status.Error(
			codes.NotFound,
			"wrong shard",
		)
	}

	//3. Determine Shard Index

	// log.Println("get", shardIndex, request.Key)
	shard_index := GetShardForKey(request.Key, server.shardMap.NumShards())

	//4. Check if the key is valid and return if true=
	server.kv_locks[shard_index-1].RLock()
	if val, ok := server.kv_store[shard_index-1][request.Key]; ok {
		if val.expire_time < uint64(time.Now().UnixMilli()) {
			response.Value = ""
			response.WasFound = false
		} else {
			response.Value = val.value
			response.WasFound = true
		}
	} else {
		response.Value = ""
		response.WasFound = false
	}
	server.kv_locks[shard_index-1].RUnlock()

	elapsed := time.Now().UnixNano() - start

	server.p99Read_mu.Lock()
	server.shardToP99Read[shard_index].Add(float64(elapsed), 1)
	server.p99Read_mu.Unlock()

	return &response, nil
}

func (server *KvServerImpl) Set(
	ctx context.Context,
	request *proto.SetRequest,
) (*proto.SetResponse, error) {

	start := time.Now().UnixNano()

	//1. Check Key requirements
	if request.Key == "" {

		return nil, status.Error(
			codes.InvalidArgument,
			"empty key",
		)
	}

	if !server.CheckCorrectShard(request.Key) {
		return nil, status.Error(
			codes.NotFound,
			"wrong shard",
		)
	}

	//2. Determine Shard Index

	shardIndex := GetShardForKey(request.Key, server.shardMap.NumShards())
	sIndex := shardIndex - 1
	server.kv_locks[sIndex].Lock()
	value_ttl_pair := server.kv_store[sIndex][request.Key]
	value_ttl_pair.value = request.Value
	value_ttl_pair.expire_time = uint64(time.Now().UnixMilli()) + uint64(request.TtlMs)
	server.kv_store[sIndex][request.Key] = value_ttl_pair
	server.kv_locks[sIndex].Unlock()

	//3. Update Latencies
	var response proto.SetResponse
	elapsed := time.Now().UnixNano() - start
	server.p99Write_mu.Lock()
	server.shardToP99Write[shardIndex].Add(float64(elapsed), 1)
	server.p99Write_mu.Unlock()

	return &response, nil
}

func (server *KvServerImpl) Delete(
	ctx context.Context,
	request *proto.DeleteRequest,
) (*proto.DeleteResponse, error) {

	start := time.Now().UnixNano()

	if request.Key == "" {
		return nil, status.Error(
			codes.InvalidArgument,
			"empty key",
		)
	}

	if !server.CheckCorrectShard(request.Key) {

		return nil, status.Error(
			codes.NotFound,
			"wrong shard",
		)
	}

	shardIndex := GetShardForKey(request.Key, server.shardMap.NumShards())

	server.kv_locks[shardIndex-1].Lock()
	delete(server.kv_store[shardIndex-1], request.Key)
	server.kv_locks[shardIndex-1].Unlock()

	var response proto.DeleteResponse

	elapsed := time.Now().UnixNano() - start
	shard_index := GetShardForKey(request.Key, server.shardMap.NumShards())

	server.p99Write_mu.Lock()
	server.shardToP99Write[shard_index].Add(float64(elapsed), 1)
	server.p99Write_mu.Unlock()

	return &response, nil
}

func (server *KvServerImpl) GetShardContents(
	ctx context.Context,
	request *proto.GetShardContentsRequest,
) (*proto.GetShardContentsResponse, error) {

	shard_indx := request.Shard - 1
	server.kv_locks[shard_indx].RLock()
	defer server.kv_locks[shard_indx].RUnlock()

	ret := make([]*proto.GetShardValue, 0)

	for k, v := range server.kv_store[shard_indx] {
		ret = append(ret, &proto.GetShardValue{Key: k, Value: v.value, TtlMsRemaining: int64(v.expire_time)})
	}

	return &proto.GetShardContentsResponse{Values: ret}, nil
}
