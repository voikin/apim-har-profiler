package controller

import (
	"context"
	"fmt"

	harprofilerpb "github.com/voikin/apim-proto/gen/go/apim_har_profiler/v1"
)

func (c *Controller) BuildAPIGraph(
	_ context.Context,
	req *harprofilerpb.BuildAPIGraphRequest,
) (*harprofilerpb.BuildAPIGraphResponse, error) {
	graph, err := parseHARtoGraph(req.HarJson)
	if err != nil {
		return nil, fmt.Errorf("parseHARtoGraph: %w", err)
	}

	return &harprofilerpb.BuildAPIGraphResponse{
		Graph: graph,
	}, nil
}
