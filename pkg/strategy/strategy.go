package strategy

import (
	"fmt"
	"github.com/realHoangHai/golb/pkg/log"
	"github.com/realHoangHai/golb/pkg/node"
	"sync"
)

// Known load balancing strategies, each entry in this block should correspond
// to a load balancing strategy with a concrete implementation.
const (
	sRoundRobin         = "RoundRobin"
	sWeightedRoundRobin = "WeightedRoundRobin"
	sUnknown            = "Unknown"
)

// BalancingStrategy is the load balancing abstraction that every algorithm should implement.
type BalancingStrategy interface {
	Next([]*node.Node) (*node.Node, error)
}

// Map of BalancingStrategy factories
var strategies map[string]func() BalancingStrategy

func init() {
	strategies = make(map[string]func() BalancingStrategy, 0)
	strategies[sRoundRobin] = func() BalancingStrategy {
		return &RoundRobin{
			mutex:   sync.Mutex{},
			current: 0,
		}
	}
	strategies[sWeightedRoundRobin] = func() BalancingStrategy {
		return &WeightedRoundRobin{mutex: sync.Mutex{}}
	}
	// Add other load balancing strategies here
}

type RoundRobin struct {
	// The current node to forward the request to.
	// the next node should be (current + 1) % len(Nodes)
	mutex   sync.Mutex
	current int
}

func (r *RoundRobin) Next(servers []*node.Node) (*node.Node, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	index := 0
	var choosen *node.Node
	for index < len(servers) {
		choosen = servers[r.current]
		r.current = (r.current + 1) % len(servers)
		if choosen.IsAlive() {
			break
		}
		index++
	}
	if choosen == nil || index == len(servers) {
		log.Errorf("No alive servers found")
		return nil, fmt.Errorf("no available servers")
	}
	log.Infof("Choosen node: %s", choosen.Url.Host)
	return choosen, nil
}

// WeightedRoundRobin is a strategy that is similar to the RoundRobin strategy,
// the only difference is that it takes node compute power into consideration.
// The compute power of a node is given as an integer, it represents the
// fraction of requests that one node can handle over another.
//
// A RoundRobin strategy is equivalent to a WeightedRoundRobin strategy with all
// weights = 1
type WeightedRoundRobin struct {
	// Any changes to the below field should only be done while holding the `mu`
	// lock.
	mutex sync.Mutex
	// Note: This is making the assumption that the node list coming through the
	// Next function won't change between succesive calls.
	// Changing the node list would cause this strategy to break, panic, or not
	// route properly.
	//
	// count will keep track of the number of request node `i` processed.
	count []int
	// cur is the index of the last node that executed a request.
	cursor int
}

func (w *WeightedRoundRobin) Next(servers []*node.Node) (*node.Node, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.count == nil {
		// First time using the strategy
		w.count = make([]int, len(servers))
		w.cursor = 0
	}

	index := 0
	var choosen *node.Node
	for index < len(servers) {
		choosen = servers[w.cursor]
		capacity := choosen.GetMetadataInt("weight", 1)
		if !choosen.IsAlive() {
			index += 1
			// Current node is not alive, so we reset the node's bucket count
			// and we try the next node in the next loop iteration
			w.count[w.cursor] = 0
			w.cursor = (w.cursor + 1) % len(servers)
			continue
		}

		if w.count[w.cursor] <= capacity {
			w.count[w.cursor] += 1
			log.Infof("Choosen node: %s", choosen.Url.Host)
			return choosen, nil
		}
		// node is at it's limit, reset the current one
		// and move on to the next node
		w.count[w.cursor] = 0
		w.cursor = (w.cursor + 1) % len(servers)
	}

	if choosen == nil || index == len(servers) {
		log.Errorf("No alive servers found")
		return nil, fmt.Errorf("no available servers")
	}

	return choosen, nil
}

// LoadStrategy will try and resolve the balancing strategy based on the name,
// and will default to a 'RoundRobin' one if no strategy matched.
func LoadStrategy(name string) BalancingStrategy {
	s, ok := strategies[name]
	if !ok {
		log.Warnf("Strategy with name '%s' not found, falling back to a RoundRobin strategy", name)
		return strategies[sRoundRobin]()
	}
	log.Infof("Choosen strategy '%s'", name)
	return s()
}
