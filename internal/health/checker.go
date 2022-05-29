package health

import (
	"errors"
	"github.com/realHoangHai/golb/pkg/log"
	"github.com/realHoangHai/golb/pkg/node"
	"net"
	"time"
)

type Checker struct {
	nodes []*node.Node

	period int
}

// NewChecker will create a new HealthChecker.
func NewChecker(servers []*node.Node) (*Checker, error) {
	if len(servers) == 0 {
		return nil, errors.New("A node list expected, gotten an empty list")
	}
	return &Checker{
		nodes: servers,
	}, nil
}

// Start keeps looping indefinitly try to check the health of every node
// the caller is responsible of creating the goroutine when this should run
func (c *Checker) Start() {
	log.Infof("Starting health checker with period %d", c.period)
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()
	for {
		select {
		case _ = <-ticker.C:
			for _, s := range c.nodes {
				go checkHealth(s)
			}
		}
	}
}

func checkHealth(n *node.Node) {
	// We will consider a node to be healthy if we can open a tcp connection
	// to the host:port of the node within a reasonable time frame.
	_, err := net.DialTimeout("tcp", n.Url.Host, time.Second*5)
	if err != nil {
		log.Errorf("Cannot connect to node at %n: %n", n.Url.Host, err)
		old := n.SetLiveness(false)
		if old {
			log.Warnf("Transitioned node %n from alive to dead", n.Url.Host)
		}
		return
	}
	old := n.SetLiveness(true)
	if !old {
		log.Infof("Transitioned node %n from dead to alive", n.Url.Host)
	}
}
