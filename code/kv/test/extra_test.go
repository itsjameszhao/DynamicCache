package kvtest

// import (
// 	"fmt"
// 	"math/rand"
// 	"testing"
// 	"time"

// 	"cs426.yale.edu/lab4/kv"
// 	"github.com/sirupsen/logrus"
// 	"github.com/stretchr/testify/assert"
// )

// // Like previous labs, you must write some tests on your own.
// // Add your test cases in this file and submit them as extra_test.go.
// // You must add at least 5 test cases, though you can add as many as you like.
// //
// // You can use any type of test already used in this lab: server
// // tests, client tests, or integration tests.
// //
// // You can also write unit tests of any utility functions you have in utils.go
// //
// // Tests are run from an external package, so you are testing the public API
// // only. You can make methods public (e.g. utils) by making them Capitalized.

// //This test whether the get shards RPC is working sucessfully
// func TestGetContentsRPC(t *testing.T) {
// 	setup := MakeTestSetup(
// 		kv.ShardMapState{
// 			NumShards: 1,
// 			Nodes:     makeNodeInfos(2),
// 			ShardsToNodes: map[int][]string{
// 				1: {"n1"},
// 			},
// 		},
// 	)
// 	fmt.Printf("Setup is %v", setup)
// 	keys := RandomKeys(100, 10)
// 	vals := RandomKeys(100, 1024*1024)
// 	logrus.Debugf("created random data")
// 	for i, k := range keys {
// 		val := vals[i]
// 		err := setup.NodeSet("n1", k, val, 10*time.Second)
// 		assert.Nil(t, err)
// 	}

// 	for _, k := range keys {
// 		_, wasFound, err := setup.NodeGet("n1", k)
// 		assert.Nil(t, err)
// 		assert.True(t, wasFound)
// 	}

// 	setup.UpdateShardMapping(map[int][]string{
// 		1: {"n1", "n2"},
// 	})
// 	// Remove shard 1 from n1
// 	setup.UpdateShardMapping(map[int][]string{
// 		1: {"n2"},
// 	})
// 	cnt := 0
// 	for _, k := range keys {
// 		val, wasFound, err := setup.NodeGet("n2", k)
// 		assert.Nil(t, err)
// 		assert.True(t, wasFound)
// 		assert.Equal(t, val, vals[cnt])
// 		cnt++
// 	}
// 	setup.Shutdown()
// }

// func TestTTL(t *testing.T) {

// 	setup := MakeTestSetup(MakeBasicOneShard())

// 	for i := 0; i < 100; i++ {

// 		original_ttl := rand.Intn(200)
// 		sleep_ttl := rand.Intn(200)

// 		err := setup.NodeSet("n1", "abc", "123", time.Duration(original_ttl)*time.Millisecond)
// 		assert.Nil(t, err)
// 		time.Sleep(time.Duration(sleep_ttl) * time.Millisecond)

// 		if sleep_ttl > original_ttl {
// 			_, wasFound, err := setup.NodeGet("n1", "abc")
// 			assert.Nil(t, err)
// 			assert.False(t, wasFound)
// 		} else {
// 			// val, wasFound, err := setup.NodeGet("n1", "abc")
// 			// assert.Nil(t, err)
// 			// assert.True(t, wasFound)
// 			// assert.Equal(t, "123", val)
// 		}
// 	}
// 	assert.True(t, true)
// }

// func MakeThreeNodeBothAssignedSingleShard() kv.ShardMapState {
// 	return kv.ShardMapState{
// 		NumShards: 1,
// 		Nodes:     makeNodeInfos(3),
// 		ShardsToNodes: map[int][]string{
// 			1: {"n1", "n2", "n3"},
// 		},
// 	}

// }

// //Test Client Load Balancing with three nodes
// func TestClientGetLoadBalance(t *testing.T) {
// 	// Tests load balancing of Get() calls on three nodes

// 	setup := MakeTestSetupWithoutServers(MakeThreeNodeBothAssignedSingleShard())

// 	setup.clientPool.OverrideGetResponse("n1", "val", true)
// 	setup.clientPool.OverrideGetResponse("n2", "val", true)
// 	setup.clientPool.OverrideGetResponse("n3", "val", true)

// 	for i := 0; i < 100; i++ {
// 		val, wasFound, err := setup.Get("abc")
// 		assert.Nil(t, err)
// 		assert.True(t, wasFound)
// 		assert.Equal(t, "val", val)
// 	}
// 	n1Rpc := setup.clientPool.GetRequestsSent("n1")
// 	assert.Less(t, 20, n1Rpc)
// 	assert.Greater(t, 50, n1Rpc)

// 	n2Rpc := setup.clientPool.GetRequestsSent("n2")
// 	assert.Less(t, 20, n2Rpc)
// 	assert.Greater(t, 50, n2Rpc)

// 	n3Rpc := setup.clientPool.GetRequestsSent("n2")
// 	assert.Less(t, 20, n3Rpc)
// 	assert.Greater(t, 50, n3Rpc)
// }

// // This test should check if we maintain the same expire time after we update it
// // by calling GetShardContents.
// func TestServerExpireTimeUpdate(t *testing.T) {
// 	setup := MakeTestSetup(MakeTwoNodeMultiShard())

// 	keys := RandomKeys(100, 10)
// 	keysToOriginalNode := make(map[string]string)
// 	for _, key := range keys {
// 		err := setup.NodeSet("n1", key, "123", 2*time.Second)
// 		if err == nil {
// 			keysToOriginalNode[key] = "n1"
// 		} else {
// 			assertShardNotAssigned(t, err)
// 			err = setup.NodeSet("n2", key, "123", 10*time.Second)
// 			assert.Nil(t, err)
// 			keysToOriginalNode[key] = "n2"
// 		}
// 	}

// 	setup.UpdateShardMapping(map[int][]string{
// 		1: {"n1", "n2"},
// 		2: {"n1", "n2"},
// 	})

// 	setup.UpdateShardMapping(map[int][]string{
// 		1: {"n2"},
// 		2: {"n1"},
// 	})

// 	time.Sleep(2 * time.Second)

// 	for _, key := range keys {
// 		var newNode string
// 		if keysToOriginalNode[key] == "n1" {
// 			newNode = "n2"
// 		} else {
// 			newNode = "n1"
// 		}
// 		// But it should not exist
// 		_, wasFound, _ := setup.NodeGet(newNode, key)
// 		assert.False(t, wasFound)
// 	}

// 	setup.Shutdown()

// }

// func TestSingleShardThousandCopy(t *testing.T) {
// 	// Similar to above, but actually tests shard copying from n1 to n2.
// 	// A single shard is assigned to n1, one key written, then we add the
// 	// shard to n2. At this point n2 should copy data from its peer n1.
// 	//
// 	// We then delete the shard from n1, so n2 is the sole owner, and ensure
// 	// that n2 has the data originally written to n1.
// 	setup := MakeTestSetup(
// 		kv.ShardMapState{
// 			NumShards: 1,
// 			Nodes:     makeNodeInfos(2),
// 			ShardsToNodes: map[int][]string{
// 				1: {"n1"},
// 			},
// 		},
// 	)

// 	// n1 hosts the shard, so we should be able to set data
// 	keys := RandomKeys(1000, 10)
// 	for _, key := range keys {
// 		err := setup.NodeSet("n1", key, "123", 10*time.Second)
// 		assert.Nil(t, err)
// 	}
// 	// Add shard 1 to n2
// 	setup.UpdateShardMapping(map[int][]string{
// 		1: {"n1", "n2"},
// 	})
// 	// Remove shard 1 from n1
// 	setup.UpdateShardMapping(map[int][]string{
// 		1: {"n2"},
// 	})

// 	for _, key := range keys {
// 		_, _, err := setup.NodeGet("n1", key)
// 		assertShardNotAssigned(t, err)
// 	}

// 	for _, key := range keys {
// 		val, wasFound, err := setup.NodeGet("n2", key)
// 		assert.Nil(t, err)
// 		assert.True(t, wasFound)
// 		assert.Equal(t, val, "123")
// 	}

// 	setup.Shutdown()
// }
