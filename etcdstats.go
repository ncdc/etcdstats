package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"k8s.io/kubernetes/pkg/client/restclient"
	knet "k8s.io/kubernetes/pkg/util/net"

	"golang.org/x/net/context"

	"github.com/coreos/etcd/client"
)

// borrowed from
// https://github.com/openshift/origin/blob/bfa9bb91c15d3c3f1b671b98939cf8f1e911d8a7/pkg/cmd/server/etcd/etcd.go#L68-L99
func makeEtcdClient(server, cacert, cert, key string) (client.Client, error) {
	tlsConfig, err := restclient.TLSConfigFor(&restclient.Config{
		TLSClientConfig: restclient.TLSClientConfig{
			CertFile: cert,
			KeyFile:  key,
			CAFile:   cacert,
		},
	})
	if err != nil {
		return nil, err
	}

	transport := knet.SetTransportDefaults(&http.Transport{
		TLSClientConfig: tlsConfig,
		Dial: (&net.Dialer{
			// default from http.DefaultTransport
			Timeout: 30 * time.Second,
			// Lower the keep alive for connections.
			KeepAlive: 1 * time.Second,
		}).Dial,
		// Because watches are very bursty, defends against long delays in watch reconnections.
		MaxIdleConnsPerHost: 500,
	})

	cfg := client.Config{
		Endpoints: []string{server},
		Transport: transport,
	}
	return client.New(cfg)

}

type multiValueFlag []string

func (m *multiValueFlag) String() string {
	return fmt.Sprint(*m)
}

func (m *multiValueFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func main() {
	var (
		server    string
		cacert    string
		cert      string
		key       string
		n         int
		summarize multiValueFlag
		prefix    string
	)

	flag.StringVar(&server, "server", "", "server url, e.g. https://127.0.0.1:2379 (required)")
	flag.StringVar(&cacert, "cacert", "", "CA certificate file (optional)")
	flag.StringVar(&cert, "cert", "", "client certificate file (optional)")
	flag.StringVar(&key, "key", "", "client certificate key file (optional)")
	flag.IntVar(&n, "n", 20, "display top n highest nodes")
	flag.Var(&summarize, "summarize", "summarize descendent nodes for the directory prefixed by this value instead of displaying these nodes; may specify multiple times for multiple directories")
	flag.StringVar(&prefix, "prefix", "/", "directory prefix to summarize")

	flag.Parse()

	if len(server) == 0 {
		log.Fatal("--server is required")
	}

	c, err := makeEtcdClient(server, cacert, cert, key)
	if err != nil {
		log.Fatal(err)
	}

	kapi := client.NewKeysAPI(c)
	s := stats{
		client: c,
		keys:   kapi,
		seen:   make(map[string]*nodeinfo),
		list:   make([]*nodeinfo, 0),
	}
	if err := s.examineNode(prefix); err != nil {
		log.Fatal(err)
	}

	sum := 0
	sumWithoutExclusions := 0
	sort.Sort(bysize(s.list))
	printed := 0
	fmt.Printf("Top %d highest etcd nodes by value size (excluding summarized items):\n", n)

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintf(w, "NODE\tCHILDREN\tSIZE\n")

Outer:
	for i := range s.list {
		ni := s.list[i]
		if !ni.dir {
			sum += ni.size
			// skip all summary-only nodes
			for i := range summarize {
				if strings.HasPrefix(ni.key, string(summarize[i])) {
					continue Outer
				}
			}
			sumWithoutExclusions += ni.size
			if printed < n {
				fmt.Fprintf(w, "%s\tN/A\t%d\n", ni.key, ni.size)
				printed++
			}
		} else {
			if printed < n {
				fmt.Fprintf(w, "%s\t%d\t%d\n", ni.key, ni.children, ni.size)
				printed++
			}
		}
	}
	w.Flush()
	fmt.Println("\nTotal value size:", sum)
	fmt.Println("Value size excluding summarized items", sumWithoutExclusions)
}

type stats struct {
	client client.Client
	keys   client.KeysAPI
	seen   map[string]*nodeinfo
	list   []*nodeinfo
}

type nodeinfo struct {
	key      string
	dir      bool
	size     int
	children int
}

func (s *stats) examineNode(key string) error {
	resp, err := s.keys.Get(context.Background(), key, &client.GetOptions{})
	if err != nil {
		return err
	}

	node := resp.Node

	ni, ok := s.seen[key]
	if !ok {
		ni = &nodeinfo{key: key}
		s.seen[key] = ni
		s.list = append(s.list, ni)
	}

	if !node.Dir {
		ni.size = len(node.Value)
		li := strings.LastIndex(key, "/")
		if li > -1 {
			parentKey := key[0:li]
			parentNode, ok := s.seen[parentKey]
			if ok {
				parentNode.size += ni.size
				parentNode.children++
			} else {
				log.Printf("couldn't find parent %s", parentKey)
			}
		}
	} else {
		ni.dir = true
		for i := range node.Nodes {
			if err := s.examineNode(node.Nodes[i].Key); err != nil {
				return err
			}
		}
	}

	return nil
}

type bysize []*nodeinfo

func (b bysize) Len() int           { return len(b) }
func (b bysize) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b bysize) Less(i, j int) bool { return b[i].size > b[j].size }
