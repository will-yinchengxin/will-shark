package hash

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
)

/*
 * @package will_tools/hash/consistent_hash.go
 * @author：Will Yin <826895143@qq.com>
 * @copyright Copyright (C) 2023/4/12 Will


	简单 hash: 简单的哈希函数 m = hash(o) mod n, 其中，o为对象名称，n为机器的数量，m为机器编号
	三个机器: 1 2 3
	10 个数字: 1 ~ 10
	数据分布: 机器0: 3，6，9   机器1: 1，4，7，10  机器2: 2，5，8

	当新增一个机器
	数据分布: 机器0: 4，8   机器1: 1，5，9  机器2: 2，6，10   机器3: 3，7

	只有数据1和数据2没有移动，所以当集群中数据量很大时，采用一般的哈希函数，在节点数量动态变化的情况下会造成大量的数据迁移，
	导致网络通信压力的剧增，严重情况，还可能导致数据库宕机。

	一致性hash算法正是为了解决此类问题的方法，它可以保证当机器增加或者减少时，
	节点之间的数据迁移只限于两个节点之间，不会造成全局的网络问题

	当集群中的节点数量较少时，可能会出现节点在哈希空间中分布不平衡的问题。如果节点A、B、C分布较为集中，造成hash环的倾斜。
	数据1、2、3、4、6全部被存储到了节点A上，节点B上只存储了数据5，而节点C上什么数据都没有存储。A、B、C三台机器的负载极其不均衡
	在极端情况下，假如A节点出现故障，存储在A上的数据要全部转移到B上，大量的数据导可能会导致节点B的崩溃，之后A和B上所有的数据向节点C迁移，
	导致节点C也崩溃，由此导致整个集群宕机。这种情况被称为雪崩效应

	解决办法:

	- 虚拟节点法: 我们可以将现有的物理节通过虚拟的方法复制多个出来，这些由实际节点虚拟复制而来的节点被称为虚拟节点

	在分布式集群中，经常要寻找指定数据存储的物理节点，关于这个问题有三种比较典型的方法来解决
	  - Napster：
	    使用一个中心服务器接收所有的查询，中心服务器返回数据存储的节点位置信息。
	    存在的问题：中心服务器单点失效导致整个网络瘫痪

	  - Gnutella：
	    使用消息洪泛（message flooding）来定位数据。一个消息被发到系统内每一个节点，直到找到其需要的数据为止。使用生存时间（TTL）来限制网络内转发消息的数量。
	    存在的问题：消息数与节点数成线性关系，导致网络负载较重

	  - SN:
	    现在大多数采用所谓超级节点（Super Node），SN保存网络中节点的索引信息，这一点和中心服务器类型一样，但是网内有多个SN，
	    其索引信息会在这些SN中进行传播，所以整个系统的崩溃几率就会小很多。尽管如此，网络还是有崩溃的可能
*/

const (
	TopWeight   = 100
	minReplicas = 100
	prime       = 16777619
)

type (
	Func func(data []byte) uint64

	ConsistentHash struct {
		hashFunc Func
		replicas int
		keys     []uint64
		ring     map[uint64][]interface{}
		nodes    map[string]struct{}
		lock     sync.RWMutex
	}
)

func NewConsistentHash() *ConsistentHash {
	return NewCustomConsistentHash(minReplicas, Hash)
}

func NewCustomConsistentHash(replicas int, fn Func) *ConsistentHash {
	if replicas < minReplicas {
		replicas = minReplicas
	}

	if fn == nil {
		fn = Hash
	}

	return &ConsistentHash{
		hashFunc: fn,
		replicas: replicas,
		ring:     make(map[uint64][]interface{}),
		nodes:    make(map[string]struct{}),
	}
}

func (h *ConsistentHash) Add(node interface{}) {
	h.AddWithReplicas(node, h.replicas)
}

func (h *ConsistentHash) AddWithReplicas(node interface{}, replicas int) {
	h.Remove(node)
	if replicas > h.replicas {
		replicas = h.replicas
	}
	nodeRepr := repr(node)
	h.lock.Lock()
	defer h.lock.Unlock()
	h.addNode(nodeRepr)
	for i := 0; i < replicas; i++ {
		hash := h.hashFunc([]byte(nodeRepr + strconv.Itoa(i)))
		h.keys = append(h.keys, hash)
		h.ring[hash] = append(h.ring[hash], node)
	}
	sort.Slice(h.keys, func(i, j int) bool {
		return h.keys[i] < h.keys[j]
	})
}

func (h *ConsistentHash) AddWithWeight(node interface{}, weight int) {
	replicas := h.replicas * weight / TopWeight
	h.AddWithReplicas(node, replicas)
}

func (h *ConsistentHash) Get(v interface{}) (interface{}, bool) {
	h.lock.RLock()
	defer h.lock.RUnlock()
	if len(h.ring) == 0 {
		return nil, false
	}
	hash := h.hashFunc([]byte(repr(v)))
	index := sort.Search(len(h.keys), func(i int) bool {
		return h.keys[i] >= hash
	}) % len(h.keys)
	nodes := h.ring[h.keys[index]]
	switch len(nodes) {
	case 0:
		return nil, false
	case 1:
		return nodes[0], true
	default:
		innerIndex := h.hashFunc([]byte(innerRepr(v)))
		pos := int(innerIndex % uint64(len(nodes)))
		return nodes[pos], true
	}
}

func (h *ConsistentHash) Remove(node interface{}) {
	nodeRepr := repr(node)
	h.lock.Lock()
	defer h.lock.Unlock()
	if !h.containsNode(nodeRepr) {
		return
	}
	for i := 0; i < h.replicas; i++ {
		hash := h.hashFunc([]byte(nodeRepr + strconv.Itoa(i)))
		index := sort.Search(len(h.keys), func(i int) bool {
			return h.keys[i] >= hash
		})
		if index < len(h.keys) && h.keys[index] == hash {
			h.keys = append(h.keys[:index], h.keys[index+1:]...)
		}
		h.removeRingNode(hash, nodeRepr)
	}
	//删除真实节点
	h.removeNode(nodeRepr)
}

func (h *ConsistentHash) removeRingNode(hash uint64, nodeRepr string) {
	if nodes, ok := h.ring[hash]; ok {
		newNodes := nodes[:0]
		for _, x := range nodes {
			if repr(x) != nodeRepr {
				newNodes = append(newNodes, x)
			}
		}
		if len(newNodes) > 0 {
			h.ring[hash] = newNodes
		} else {
			delete(h.ring, hash)
		}
	}
}

func (h *ConsistentHash) addNode(nodeRepr string) {
	h.nodes[nodeRepr] = struct{}{}
}

func (h *ConsistentHash) containsNode(nodeRepr string) bool {
	_, ok := h.nodes[nodeRepr]
	return ok
}

func (h *ConsistentHash) removeNode(nodeRepr string) {
	delete(h.nodes, nodeRepr)
}

func innerRepr(v interface{}) string {
	return fmt.Sprintf("%d:%v", prime, v)
}

func repr(node interface{}) string {
	if node == nil {
		return ""
	}

	return ReprOfValue(node)
}
