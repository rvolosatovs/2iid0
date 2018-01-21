package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/multi"
	"gonum.org/v1/gonum/graph/network"
	"gonum.org/v1/gonum/graph/simple"
)

func init() {
	log.SetOutput(os.Stderr)
}

const graphCount = 5

var _ graph.Graph = (*TemporalGraph)(nil)
var _ graph.Directed = (*TemporalGraph)(nil)

type TemporalEdge struct {
	graph.Edge
	Start time.Time
	End   time.Time
}

type TemporalGraph struct {
	nodes map[int64]graph.Node
	from  map[int64]map[int64]*TemporalEdge
	to    map[int64]map[int64]*TemporalEdge
	Start time.Time
	End   time.Time
}

func (g *TemporalGraph) Has(n graph.Node) bool {
	_, ok := g.nodes[n.ID()]
	return ok
}

func (g *TemporalGraph) Nodes() []graph.Node {
	nodes := make([]graph.Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		nodes = append(nodes, n)
	}
	return nodes
}

func (g *TemporalGraph) From(n graph.Node) []graph.Node {
	ns, ok := g.from[n.ID()]
	if !ok {
		return nil
	}
	from := make([]graph.Node, 0, len(ns))
	for id := range ns {
		from = append(from, g.nodes[id])
	}
	return from
}

func (g *TemporalGraph) HasEdgeBetween(x graph.Node, y graph.Node) bool {
	xid := x.ID()
	yid := y.ID()
	if to, ok := g.from[xid]; ok {
		if _, ok = to[yid]; ok {
			return true
		}
	}
	if to, ok := g.from[yid]; ok {
		if _, ok = to[xid]; ok {
			return true
		}
	}
	return false
}

func (g *TemporalGraph) Edge(u graph.Node, v graph.Node) graph.Edge {
	to, ok := g.from[u.ID()]
	if !ok {
		return nil
	}
	return to[v.ID()]
}

func (g *TemporalGraph) HasEdgeFromTo(u graph.Node, v graph.Node) bool {
	to, ok := g.from[u.ID()]
	if !ok {
		return false
	}
	_, ok = to[v.ID()]
	return ok
}

func (g *TemporalGraph) To(n graph.Node) []graph.Node {
	ns, ok := g.to[n.ID()]
	if !ok {
		return nil
	}
	to := make([]graph.Node, 0, len(ns))
	for id := range ns {
		to = append(to, g.nodes[id])
	}
	return to
}

func (g *TemporalGraph) AddNode(n graph.Node) {
	if _, ok := g.nodes[n.ID()]; ok {
		panic(fmt.Errorf("node %d already exists", n.ID()))
	}
	g.nodes[n.ID()] = n
	g.from[n.ID()] = make(map[int64]*TemporalEdge)
	g.to[n.ID()] = make(map[int64]*TemporalEdge)
}

func (g *TemporalGraph) Edges() []graph.Edge {
	es := make([]graph.Edge, 0, len(g.from))
	for _, to := range g.from {
		for _, e := range to {
			es = append(es, e)
		}
	}
	return es
}

func (g *TemporalGraph) SetEdge(e *TemporalEdge) {
	from := e.From()
	if !g.Has(from) {
		g.AddNode(from)
	}

	to := e.To()
	if !g.Has(to) {
		g.AddNode(to)
	}

	fid, tid := from.ID(), to.ID()

	if _, ok := g.from[fid][tid]; ok {
		panic(fmt.Sprintf("Edge %d -> %d already exists", fid, tid))
	}
	g.from[fid][tid] = e

	if _, ok := g.to[tid][fid]; ok {
		panic(fmt.Sprintf("Edge %d -> %d already exists", fid, tid))
	}
	g.to[tid][fid] = e

	if e.Start.Before(g.Start) || g.Start.IsZero() {
		g.Start = e.Start
	}

	if e.End.After(g.End) || g.End.IsZero() {
		g.End = e.End
	}
}

func NewTemporalGraph(nodeHint, edgeHint int) *TemporalGraph {
	return &TemporalGraph{
		nodes: make(map[int64]graph.Node, nodeHint),
		from:  make(map[int64]map[int64]*TemporalEdge, edgeHint),
		to:    make(map[int64]map[int64]*TemporalEdge, edgeHint),
	}
}

type TemporalMultiGraph struct {
	*multi.DirectedGraph
	temporal map[graph.Line]*TemporalEdge
	Start    time.Time
	End      time.Time
}

func (g *TemporalMultiGraph) SetEdge(e *TemporalEdge) {
	l := g.DirectedGraph.NewLine(e.From(), e.To())
	_, ok := g.temporal[l]
	if ok {
		panic(fmt.Errorf("Temporal edge for line ID %d already exists", l.ID()))
	}
	g.temporal[l] = e
	g.DirectedGraph.SetLine(l)

	if e.Start.Before(g.Start) || g.Start.IsZero() {
		g.Start = e.Start
	}

	if e.End.After(g.End) || g.End.IsZero() {
		g.End = e.End
	}
}

func (g *TemporalMultiGraph) Snapshot(t time.Time) *TemporalGraph {
	ns := g.Nodes()
	out := NewTemporalGraph(len(ns), len(g.Edges()))
	for _, u := range ns {
		for _, v := range g.From(u) {
			for _, l := range g.Lines(u, v) {
				e, ok := g.temporal[l]
				if !ok {
					panic(fmt.Errorf("Temporal edge for line ID %d does not exist", l.ID()))
				}
				if e.Start.After(t) || e.End.Before(t) {
					continue
				}
				out.SetEdge(e)
			}
		}
	}
	return out
}

func NewTemporalMultiGraph(edgeHint int) *TemporalMultiGraph {
	return &TemporalMultiGraph{
		DirectedGraph: multi.NewDirectedGraph(),
		temporal:      make(map[graph.Line]*TemporalEdge, edgeHint),
	}
}

type jfloat64 float64

func (f jfloat64) MarshalJSON() ([]byte, error) {
	v := float64(f)
	if math.IsInf(v, 1) {
		return []byte("\"+Inf\""), nil
	}
	if math.IsInf(v, -1) {
		return []byte("\"-Inf\""), nil
	}
	return json.Marshal(v)
}

func main() {
	n := flag.Int("n", 5, "temporal graphs to evaluate")
	edges := flag.Int("edges", 4729035, "edge count hint")
	flag.Usage = func() {
		name := os.Args[0]
		fmt.Fprintf(os.Stderr, "Usage: %s file\n", name)
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		log.Fatalf("Invalid number of arguments, expected %d, got %d", 1, flag.NArg())
	}

	var fpath string
	if fpath = flag.Arg(0); fpath == "" {
		log.Fatal("Empty filepath specified")
	}

	f, err := os.OpenFile(fpath, os.O_RDONLY, 0)
	if err != nil {
		log.Fatalf("Failed to open file to read at %s: %s", os.Args[1], err)
	}
	defer f.Close()

	g := NewTemporalMultiGraph(*edges)

	log.Printf("Parsing file at %s...", fpath)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var vals [4]int64
		for i, s := range strings.Fields(sc.Text()) {
			v, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				log.Fatalf("Failed to parse %s as int64: %s", s, err)
			}
			vals[i] = v
		}
		g.SetEdge(&TemporalEdge{
			Edge: simple.Edge{
				F: simple.Node(vals[0]),
				T: simple.Node(vals[1]),
			},
			Start: time.Unix(vals[2], 0),
			End:   time.Unix(vals[3], 0),
		})
	}
	if sc.Err() != nil {
		log.Fatalf("Failed to read input: %s", err)
	}

	ts := make([]time.Time, 0, *n+2)
	delta := g.End.Sub(g.Start) / time.Duration(*n)
	for t := g.Start; t.Before(g.End); t = t.Add(delta) {
		ts = append(ts, t)
	}
	ts = append(ts, g.End)

	type Measures struct {
		InDegree       int      `json:"in_degree"`
		OutDegree      int      `json:"out_degree"`
		Closeness      jfloat64 `json:"closeness"`
		Farness        jfloat64 `json:"farness"`
		Betweenness    jfloat64 `json:"betweenness"`
		HITS_Authority jfloat64 `json:"hits_authority"`
		HITS_Hub       jfloat64 `json:"hits_hub"`
		PageRank       jfloat64 `json:"pagerank"`
	}

	measure := func(g graph.Directed) map[int64]Measures {
		var pr map[int64]float64
		var cls map[int64]float64
		var far map[int64]float64
		var btw map[int64]float64
		if len(g.Nodes()) < 678907 {
			//log.Println("Dijkstra...")
			//as := path.DijkstraAllPaths(g)

			//log.Println("Closeness...")
			//cls = network.Closeness(g, as)

			//log.Println("Farness...")
			//far = network.Closeness(g, as)

			//log.Println("PageRank...")
			//pr = network.PageRank(g, 0.85, 1e-06)

			log.Println("Betweenness...")
			btw = network.Betweenness(g)
		}

		log.Println("HITS...")
		hits := network.HITS(g, 1e-08)

		m := make(map[int64]Measures)
		for _, n := range g.Nodes() {
			id := n.ID()
			m[id] = Measures{
				InDegree:       len(g.To(n)),
				OutDegree:      len(g.From(n)),
				Closeness:      jfloat64(cls[id]),
				Farness:        jfloat64(far[id]),
				Betweenness:    jfloat64(btw[id]),
				HITS_Authority: jfloat64(hits[id].Authority),
				HITS_Hub:       jfloat64(hits[id].Hub),
				PageRank:       jfloat64(pr[id]),
			}
		}
		return m
	}

	edgeList := func(edges []graph.Edge) [][2]int64 {
		out := make([][2]int64, len(edges))
		for i, e := range edges {
			out[i] = [2]int64{e.From().ID(), e.To().ID()}
		}
		return out
	}

	type Snapshot struct {
		Time     time.Time          `json:"time"`
		Measures map[int64]Measures `json:"measures"`
		Edges    [][2]int64         `json:"edges"`
	}

	os.Stdout.Write([]byte("["))

	enc := json.NewEncoder(os.Stdout)

	log.Println("Measuring original graph...")
	if err := enc.Encode(Snapshot{
		Measures: measure(g),
		Edges:    edgeList(g.Edges()),
	}); err != nil {
		log.Fatal(err)
	}

	for _, t := range ts {
		log.Printf("Taking snapshot at %s...", t)
		tg := g.Snapshot(t)
		log.Println("Measuring snapshot ...")
		if err := enc.Encode(Snapshot{
			Time:     t,
			Measures: measure(tg),
			Edges:    edgeList(tg.Edges()),
		}); err != nil {
			log.Fatal(err)
		}
	}

	os.Stdout.Write([]byte("]"))
}
