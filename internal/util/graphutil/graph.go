package graphutil

import "slices"

// Graph is a directed graph whose nodes preserve insertion order.
type Graph[N comparable] struct {
	nodes    []N
	edges    map[N][]N
	selfRefs map[N]struct{}
}

// New creates an empty directed graph.
func New[N comparable]() *Graph[N] {
	return &Graph[N]{
		nodes:    []N{},
		edges:    map[N][]N{},
		selfRefs: map[N]struct{}{},
	}
}

// AddEdge adds a directed edge. Missing nodes are added automatically and
// duplicate edges are ignored.
func (g *Graph[N]) AddEdge(from, to N) {
	g.addNode(from)
	g.addNode(to)
	if slices.Contains(g.edges[from], to) {
		return
	}
	g.edges[from] = append(g.edges[from], to)
	if from == to {
		g.selfRefs[from] = struct{}{}
	}
}

// FindCycles returns strongly connected components that contain a cycle.
func (g *Graph[N]) FindCycles() [][]N {
	op := &_TarjanOp[N]{
		graph: g.edges,
		nodes: make([]_TarjanNode, 0, len(g.edges)),
		index: make(map[N]int, len(g.edges)),
	}
	for _, node := range g.nodes {
		if _, ok := op.index[node]; !ok {
			op.strongConnect(node)
		}
	}

	cycles := make([][]N, 0, len(op.output))
	for _, component := range op.output {
		if len(component) == 1 {
			if _, ok := g.selfRefs[component[0]]; !ok {
				continue
			}
		}
		cycles = append(cycles, component)
	}
	return cycles
}

func (g *Graph[N]) addNode(node N) {
	if _, ok := g.edges[node]; ok {
		return
	}
	g.nodes = append(g.nodes, node)
	g.edges[node] = []N{}
}

type _TarjanOp[N comparable] struct {
	graph  map[N][]N
	nodes  []_TarjanNode
	stack  []N
	index  map[N]int
	output [][]N
}

type _TarjanNode struct {
	lowLink int
	stacked bool
}

func (op *_TarjanOp[N]) strongConnect(nodeValue N) *_TarjanNode {
	index := len(op.nodes)
	op.index[nodeValue] = index
	op.stack = append(op.stack, nodeValue)
	op.nodes = append(op.nodes, _TarjanNode{lowLink: index, stacked: true})
	node := &op.nodes[index]

	for _, target := range op.graph[nodeValue] {
		targetIndex, seen := op.index[target]
		if !seen {
			targetNode := op.strongConnect(target)
			if targetNode.lowLink < node.lowLink {
				node.lowLink = targetNode.lowLink
			}
		} else if op.nodes[targetIndex].stacked && targetIndex < node.lowLink {
			node.lowLink = targetIndex
		}
	}

	if node.lowLink == index {
		var component []N
		stackIndex := len(op.stack) - 1
		for {
			target := op.stack[stackIndex]
			targetIndex := op.index[target]
			op.nodes[targetIndex].stacked = false
			component = append(component, target)
			if targetIndex == index {
				break
			}
			stackIndex--
		}
		op.stack = op.stack[:stackIndex]
		op.output = append(op.output, component)
	}

	return node
}
