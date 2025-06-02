package controller

import (
	"context"
	"fmt"

	harprofilerpb "github.com/voikin/apim-proto/gen/go/apim_har_profiler/v1"
	"github.com/voikin/apim-proto/gen/go/shared/v1"
)

func (c *Controller) BuildAPIGraph(
	_ context.Context,
	req *harprofilerpb.BuildAPIGraphRequest,
) (*harprofilerpb.BuildAPIGraphResponse, error) {
	resultGraph := &shared.APIGraph{
		Segments:    []*shared.PathSegment{},
		Edges:       []*shared.Edge{},
		Operations:  []*shared.Operation{},
		Transitions: []*shared.Transition{},
	}

	segmentSet := map[string]bool{}
	edgeSet := map[string]bool{}
	opSet := map[string]bool{}
	transitionSet := map[string]bool{}

	for _, har := range req.GetHarFiles() {
		graph, err := parseHARtoGraph(har.GetContent(), har.GetIsSequence())
		if err != nil {
			return nil, fmt.Errorf("parseHARtoGraph: %w", err)
		}

		for _, seg := range graph.GetSegments() {
			segID := getSegmentID(seg)
			if !segmentSet[segID] {
				resultGraph.Segments = append(resultGraph.Segments, seg)
				segmentSet[segID] = true
			}
		}

		for _, edge := range graph.GetEdges() {
			edgeKey := edge.GetFrom() + "->" + edge.GetTo()
			if !edgeSet[edgeKey] {
				resultGraph.Edges = append(resultGraph.Edges, edge)
				edgeSet[edgeKey] = true
			}
		}

		for _, op := range graph.GetOperations() {
			if !opSet[op.GetId()] {
				resultGraph.Operations = append(resultGraph.Operations, op)
				opSet[op.GetId()] = true
			}
		}

		for _, tr := range graph.GetTransitions() {
			transitionKey := tr.GetFrom() + "->" + tr.GetTo()
			if !transitionSet[transitionKey] {
				resultGraph.Transitions = append(resultGraph.Transitions, tr)
				transitionSet[transitionKey] = true
			}
		}
	}

	return &harprofilerpb.BuildAPIGraphResponse{
		Graph: resultGraph,
	}, nil
}

func getSegmentID(seg *shared.PathSegment) string {
	if seg.GetStatic() != nil {
		return seg.GetStatic().Id
	}
	if seg.GetParam() != nil {
		return seg.GetParam().GetId()
	}

	return "unknown"
}
