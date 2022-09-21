# DynamicCache

DynamicCache is a system that is dynamic system that is able to use per-shard latency to manage the ShardManager for optimal latencies and througput. This system can be divided into three main sections: Shard Manager, P99Latency, Base Server.  

## Shard Manager
This section conisists of the Shard Manager which based on performance metrics is able to modify the configuration of the ShardMap for optimal results. 

### Usage

Be in `/cmd/shardmanager` working directory. Then execute the following command with customizable flags. 

`go run shardmanager.go -num-goros <int> -enabled <bool> -numshards <int> -numnodes <int> -max-replicas <int> -latency-max <float> -latency-min <float>`

`-num-goros <int>`: this flag is to modify the number of go routines per system to increase the load on the current system. Default: 16

`-enabled <bool>`: this flag enables the ShardManager. If set to false it runs without ShardManager. Default: false

`-numshards <int>`: this modifies the number of shards present in the system created. Default: 5 

`-numnodes <int>`: this modifies the number of nodes present in the system created. Default: 5 

`-max-replicas <int>`: this is the maximum number of replicas per shard that can be created with the Shard Manager. Default: 5 

`-latency-max <int>`: latency threshold before another shard is created. Default: 15000 

`-latency-min <int>`: latency threshold before a shard is removed. Default: 10000 

### Walkthrough 

This will be a detailed step-by-step walkthrough for how to use DynamicCACHE for an example, utilizing different flags. 

#### ShardManager Active

This section will have an active shard manager flag and will use 64 go routines. The numshards, numnodes, max-replicas and latency thresholds are set to the default amount. 

First we must be in the right directory: `cd /cmd/shardmanager`

Run this command: `go run shardmanager.go -enabled=true -num-goros=64`


There may be a case in which the program fails, we've found that sometimes we overload the  constraints of our local machine, so we ask that you either reduce the number of go-routines or re-run the program. You should be able to obtain the proper result then. 

The results should resemble this:

<pre>
INFO[0002] Read latencies are [144871 490072 314785 288654 262332] Write latencies are [2468307 5951554 4372755 4142116 4341976] 
INFO[0003] Read latencies are [70836 52569 30124 2000 161190] Write latencies are [4372755 4142116 4341976 2468307 5951554] 
INFO[0004] Read latencies are [2000 4212 1295 1000 80544] Write latencies are [4372755 4142116 4341976 2468307 5951554] 
INFO[0005] Read latencies are [1000 1000 42767 1000 2000] Write latencies are [4341976 2468307 5951554 4372755 4142116] 
INFO[0006] Read latencies are [1000 1000 8007 1000 1318] Write latencies are [4341976 2468307 5951554 4372755 4142116] 
INFO[0006] Time elapsed %v seconds, Number of requests %v, throughput (RPS) %v6.068778238 12928000 2.1302474989894163e+06 
</pre>

However, the results may be different due to the your systems computer architecture. 

#### ShardManager Not Active

Thi section will not have an active ShardManager, but will maintain the other flags as the same. 

Run this command: `go run shardmanager.go -enabled=false -num-goros=64`

The results should resemble this:

<pre>
INFO[0002] Read latencies are [317786 246388 452511 398244 350288] Write latencies are [3191144 3021419 5072443 4026545 3564017] 
INFO[0003] Read latencies are [95008 60634 167789 161789 122484] Write latencies are [3191144 3021419 5072443 4026545 3564017] 
INFO[0004] Read latencies are [18740 3513 2000 51887 49096] Write latencies are [3564017 3191144 3021419 5072443 4026545] 
INFO[0005] Read latencies are [2000 1379 1000 3747 2346] Write latencies are [3564017 3191144 3021419 5072443 4026545] 
INFO[0006] Read latencies are [1000 1000 1937 1692 1000] Write latencies are [3191144 3021419 5072443 4026545 3564017] 
INFO[0006] Time elapsed %v seconds, Number of requests %v, throughput (RPS) %v6.047678447 12928000 2.1376797321988116e+06 
</pre>

However, the results may be different due to the your systems computer architecture. 

## P99Latency

While the ShardManager is the main deliverable for this project, there were many other sections that we needed to develop in order for the ShardManager to work effectively. P99Latency is where we developed our per-shard latency calculation. For this section, we will just explain how to  test our implementation. 

For this section we must be in: `/kv/test` directory. In here there is a file named: p99_test.go which contains our implementation tests.

### Testing

To test our implementation, run the following command:

`go test -v -run='TestP99'`

The results should resemble this: 

<pre>
=== RUN   TestP99Basic
--- PASS: TestP99Basic (1.00s)
=== RUN   TestP99MultiShard
--- PASS: TestP99MultiShard (1.66s)
=== RUN   TestP99TwoNodeMultiShard
--- PASS: TestP99TwoNodeMultiShard (1.65s)
=== RUN   TestP99FourNodesFiveShard
--- PASS: TestP99FourNodesFiveShard (1.98s)
=== RUN   TestP99ConcurrentGetsAndSets
--- PASS: TestP99ConcurrentGetsAndSets (2.74s)
=== RUN   TestP99ConcurrentGetsAndSets2
--- PASS: TestP99ConcurrentGetsAndSets2 (26.04s)
PASS
</pre>

## Overall System Testing

Aside from our specific implementation testing, building upon the tests for Lab4, we attempted to test our system through all of Lab4 tests. Our system should also pass the following command, which runs all of the tests. 

`go test -v`

#Modified Files

For refrence, this section will describe the files that are modified for this project if you would like ot take a further look. 

`/cmd/shardmanager/shardmanager.go` This file has the implementation the dynamic shard manager

`/kv/p99.go` This file maintains the new p99 listener that we developed. Figure 1 in the paper. 

`/kv/test/p99_test.go` Maintains our implementation tests for our new latency listener.

`/kv/server.go` This file has the server implementation that we modified to ensure minimal bottlenecks. 

Additionally, we have a script that we used for generating images. There are present under `/scripts`

#Group Contributions

We divided our project into three main sections which each of us worked on with help from eacother. Rohan handled updating the server implemenation to ensure new concurrent locking, Ziming worked on the P99Latency, and James developed the ShardManager.go. However, we all debugged issues together and worked relative equal hours on the project. We all contributed to the write-up as well. 
