package node

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"sync"
)

type Node struct {
	Url      *url.URL
	Proxy    *httputil.ReverseProxy
	Metadata map[string]string

	mutex sync.RWMutex
	alive bool
}

func (n *Node) Forward(w http.ResponseWriter, r *http.Request) {
	n.Proxy.ServeHTTP(w, r)
}

// GetMetadata returns the value associated with the given key in the metadata, or returns the default
func (n *Node) GetMetadata(key, defaultVal string) string {
	if n.Metadata == nil {
		return defaultVal
	}
	if val, ok := n.Metadata[key]; ok {
		return val
	}
	return defaultVal
}

// GetMetadataInt returns the int value associated with the given key in the metadata, or returns the default
func (n *Node) GetMetadataInt(key string, defaultVal int) int {
	if n.Metadata == nil {
		return defaultVal
	}
	val := n.GetMetadata(key, fmt.Sprintf("%d", defaultVal))
	finalVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return finalVal
}

// SetLiveness will change the current alive field value, and return the old value.
func (n *Node) SetLiveness(value bool) bool {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	old := n.alive
	n.alive = value
	return old
}

// IsAlive reports the liveness state of the node
func (n *Node) IsAlive() bool {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	return n.alive
}
