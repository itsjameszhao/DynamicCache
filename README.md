# DynamicCache
[DynamicCache: A Sharded and Replicated Key-Value Cache with Dynamic Shard Replication](DynamicCache.pdf) Final project for Building Distributed Systems (CS 426) class at Yale. 

# Abstract
Caching is widely used to increase application throughput and latency. However, changing load conditions (read/write access patterns,  query distribution) often hinder a cache from achieving optimal performance. Modifying the cache such that it dynamically alters its sharding and replication in response to varying workload conditions can largely improve cache performance. We present DynamicCache, a dynamic key-value cache that adapts sharding and replication to optimize for throughput and latency under dynamic access patterns. For evaluations, we run simulated queries with changing load distributions, demonstrating that DynamicCache is able to adapt shard replication to query patterns and achieves superior performance in query latency and throughput (up to 66\% latency and 18\% throughput improvements) compared to the baselines.

# Design
<p align="center">
  <img align="center" width="389" alt="image" src="https://user-images.githubusercontent.com/18220413/191397895-255a1eb6-7202-4b6b-8be6-1d8cdf1de1ae.png">
</p>
Our distributed cache system. A User Client sends Get and Set requests for a certain key k and the requests are automatically sent to the relevant node(s). Each node contains instrumentation monitoring per-shard-replica latency $T$ and $RPS$. The Shard Manager periodically pulls latency $T$ and $RPS$ from each of the nodes, and uses this information to issue add / drop shard replicas commands to the distributed cache.

# Algorithm
<p align="center">
  <img align="center" width="600" alt="image" src="https://user-images.githubusercontent.com/18220413/191398108-1a9c8eb1-79f3-4e63-950d-e6d5ad77db81.png">
</p>

Algorithm 1 summarizes the process using per-shard latency $T$ as an example. For implementation, we considered both $RPS$ and $T$ as indicators of shard load. $C_{max}$ is an array which indicates the maximum number of shard replicas per shard. $[C_1, C_2, ...]$. $C_{replicas}$ is an array that stores the current number of shard replicas for each shard. Shard number is used as the index of the two arrays. $get\_shard\_latency(i)$ obtains per-shard-replica from each node and calculates per-shard latency by averaging per-shard-replica latency over all replicas of the shard. Thus, when a shard experiences high per-shard read or write latency, shard manager will add a shard replica to a node where that particular shard has not been placed. Data from old shard replicas will be copied to the new shard replica to ensure that all shard replicas share the same content. Similarly, when a shard experiences low write latency but still have redundant shard replicas, shard manager will remove a shard replica from that shard. This is to reduce the resource usage if the shard does not need to handle high workload and do not need many shard replicas. The freed shard replicas are returned back to a \textit{pool} of available shard replicas and can then be used for other hot shards.

# Results and Performance Data
Evaluations are done on Apple Macbook with 10 CPU cores, 16 GB of memory.

We compare DynamicCache with Baseline-1 (shard manager disabled with shard replication set to 1) and Baseline-5 with shard replica count set to 5. We also set the maximum number of shard replicas DynamicCache can have to 5, meaning that the shard manager will not allocate more replicas than this amount.

We compare DynamicCache with both Baseline-1 and Baseline-5 as we want to investigate whether the potential improvement is brought by dynamically adjusting shard replicas or adding more shards. 

In this evaluation, we control the number of goroutines to vary the amount of contention on our system. We measure the average per-shard read latency, average per-shard write latency, and throughput of the compared systems.

<p align="center">
  <img align="center" width="600" alt="image" src="https://user-images.githubusercontent.com/18220413/191398671-c2200ddc-3ee3-41cf-a5be-43bd482c950c.png">
</p>
 
Average per-shard read latency comparison for DynamicCache, Baseline-1, Baseline-5. The Y-axis records the average per-shard read latency, measured in microseconds.

<p align="center">
  <img align="center" width="600" alt="image" src="https://user-images.githubusercontent.com/18220413/191398710-d1134775-fc66-44cc-8c74-39fe2a977f30.png">
</p>

Average shard write latency comparison for DynamicCache, Baseline-1, and Baseline-5. The Y-axis records the average per-shard read latency, measured in microseconds. 

<p align="center">
  <img align="center" width="600" alt="image" src="https://user-images.githubusercontent.com/18220413/191398745-8258e000-fc0a-44d4-8d81-ff0345024ca7.png">
</p>

System throughput comparison for DynamicCache, Baseline-1, and Baseline-5.

We first evaluate how dynamically adjusting shard replication performs in terms of read latency. The read comparison figure compares the average per-shard read latency between the three setups (DynamicCache, Baseline-1, and Baseline-5). We observe that using shard manager improves the average shard read latency. This is because adding more shard replica spreads out the read requests, alleviating lock contention under high load scenario. DynamicCache consistently outperforms Baseline-1 and Baseline-5 across all number of goroutines, and the improvement brought by the shard manager increases as the number of goroutines increases. With 256 number of goroutines, Baseline-5 with the maximum number of shard replicas (5) suffers from up to 66% times compared to DynamicCache. This suggests that in the high-contention scenario (larger number of goroutines) always having a high number of replicas may not be a good strategy due to replicated write requests for multiple shard replicas. On the flip side, we observe that always using a low shard replication is not optimal either, Baseline-1 is up to 26% slower than DynamicCache in read latency. This evaluation illustrates the importance of dynamically adjusting shard replication factor.

We observe similar trend with average shard write latency, as shown in the write comparison figure, showing that DynamicCache improves the write performance even with more write requests. We note that the write latency for baseline with 5 shard replicas is much higher than DynamicCache and Baseline-1, indicating that over-provisioning the shard replicas (by picking the maximum replication factor) degrades the write performance, as expected. Notably, DynamicCache achieves 39% latency improvement compared to Baseline-5. By monitoring when the shard becomes hotter or cooler, DynamicCache is able to remove unnecessary replicas from a less-requested shard thereby minimizing the unnecessary read requests.

In the throughput comparison figure, we see that DynamicCache outperforms both the Baseline-1 and Baseline-5, indicating that the dynamic balancing of shard replicas improves throughput performance, up to 18%. We believe that by dynamically adjusting the shard replication, DynamicCache is able to adapt to changing workloads and ensure that there is no hot key or unnecessarily replicated shards, thereby achieving high throughput. We conduct experiments with both uniform and hot key query patterns. For uniform query patterns we generate 100 keys evenly spread out across the five shards, while for hot key we generate keys that will be hashed to the same shard. 

# Conclusion

We have presented DynamicCache, a sharded and replicated key-value cache that dynamically allocates shard replicas. Our evaluation shows that DynamicCache is able to outperforms the baseline with one replica and the baseline with 5 replica (upper bound of number of shard replica) in terms of per-shard read and write latency, while achieving comparable throughput compared to the baselines.



