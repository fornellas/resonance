// This build fails on 32 bit:
// GOMODCACHE/github.com/goccy/go-graphviz@v0.1.3/internal/ccall/cdt.go:131:11: cannot use *(*[wordSize]byte)(unsafe.Pointer(v.c)) (variable of type [8]byte) as [4]byte value in assignment (compile)
// GOMODCACHE/github.com/goccy/go-graphviz@v0.1.3/internal/ccall/cdt.go:241:11: cannot use *(*[wordSize]byte)(unsafe.Pointer(header.Data)) (variable of type [8]byte) as [4]byte value in assignment (compile)
// GOMODCACHE/github.com/goccy/go-graphviz@v0.1.3/internal/ccall/cdt.go:250:11: cannot use *(*[wordSize]byte)(unsafe.Pointer(v.c)) (variable of type [8]byte) as [4]byte value in assignment (compile)
//
//go:build !386 && !arm

package resources

import (
	"bytes"

	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
)

// Returns a Graphviz graph of all nodes.
func (ns Nodes) Graphviz() (string, error) {
	g := graphviz.New()
	defer g.Close()

	graph, err := g.Graph()
	if err != nil {
		return "", err
	}
	defer graph.Close()

	gNodeMap := map[string]*cgraph.Node{}
	for _, node := range ns {
		name := node.String()
		gNode, err := graph.CreateNode(name)
		if err != nil {
			return "", err
		}
		gNodeMap[name] = gNode
	}

	for _, node := range ns {
		gNode := gNodeMap[node.String()]
		for _, toNode := range node.requiredBy {
			toGNode := gNodeMap[toNode.String()]
			_, err := graph.CreateEdge("required_by", gNode, toGNode)
			if err != nil {
				return "", err
			}
		}
	}

	var buf bytes.Buffer
	if err := g.Render(graph, "dot", &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}
