package server

import (
	"fmt"
	"github.com/realHoangHai/golb/internal/config"
	"github.com/realHoangHai/golb/internal/health"
	"github.com/realHoangHai/golb/pkg/log"
	"github.com/realHoangHai/golb/pkg/node"
	"github.com/realHoangHai/golb/pkg/strategy"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type Server struct {
	// Config is the configuration loaded from a config file
	Config *config.Config

	// NodeList will contain a mapping between matcher and replicas
	NodeList map[string]*NodeList
}

type NodeList struct {
	// Nodes are the replicas
	Nodes []*node.Node

	// Name of the service
	Name string

	// Strategy defines how the node list is load balanced.
	// It can never be 'nil', it should always default to a 'RoundRobin' version.
	Strategy strategy.BalancingStrategy

	// Health checker for the servers
	HC *health.Checker
}

func NewServer(conf *config.Config) *Server {
	nodeMap := make(map[string]*NodeList, 0)

	for _, service := range conf.Services {
		servers := make([]*node.Node, 0)
		for _, replica := range service.Replicas {
			ur, err := url.Parse(replica.Url)
			if err != nil {
				log.Fatalf("cannot parse url: %s", err)
			}
			proxy := httputil.NewSingleHostReverseProxy(ur)
			servers = append(servers, &node.Node{
				Url:      ur,
				Proxy:    proxy,
				Metadata: replica.Metadata,
			})
		}
		checker, err := health.NewChecker(servers)
		if err != nil {
			log.Fatalf("cannot create health checker: %s", err)
		}
		nodeMap[service.Matcher] = &NodeList{
			Nodes:    servers,
			Name:     service.Name,
			Strategy: strategy.LoadStrategy(service.Strategy),
			HC:       checker,
		}
	}
	// start all the health checkers for all provided matchers
	for _, sl := range nodeMap {
		go sl.HC.Start()
	}
	return &Server{
		Config:   conf,
		NodeList: nodeMap,
	}
}

// Looks for the first node list that matches the reqPath (i.e matcher)
// Will return an error if no matcher have been found.
func (s *Server) findServiceList(reqPath string) (*NodeList, error) {
	log.Infof("Trying to find matcher for request '%s'", reqPath)
	for matcher, s := range s.NodeList {
		if strings.HasPrefix(reqPath, matcher) {
			log.Infof("Found service '%s' matching the request", s.Name)
			return s, nil
		}
	}
	return nil, fmt.Errorf("could not find a matcher for url: '%s'", reqPath)
}

func (s *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	log.Infof("Received new request: url='%s'", req.Host)
	sl, err := s.findServiceList(req.URL.Path)
	if err != nil {
		log.Errorf("Could not find a service matching the request: %s", err)
		res.WriteHeader(http.StatusNotFound)
		return
	}

	next, err := sl.Strategy.Next(sl.Nodes)

	if err != nil {
		log.Errorf("Could not find a node to serve the request: %s", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Infof("Forwarding to the node='%s'", next.Url.Host)
	next.Forward(res, req)
}

func (s *Server) Run() {
	srv := http.Server{
		Addr:    fmt.Sprintf(":%d", s.Config.Port),
		Handler: s,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("cannot start the node: %s", err)
	}
}
