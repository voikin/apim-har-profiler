package controller

import (
	"context"

	harprofilerpb "github.com/voikin/apim-proto/gen/go/apim_har_profiler/v1"
	shared "github.com/voikin/apim-proto/gen/go/shared/v1"
)

func (c *Controller) BuildAPIGraph(
	_ context.Context,
	_ *harprofilerpb.BuildAPIGraphRequest,
) (*harprofilerpb.BuildAPIGraphResponse, error) {
	return &harprofilerpb.BuildAPIGraphResponse{
		Graph: &shared.APIGraph{
			Segments: []*shared.PathSegment{
				{Segment: &shared.PathSegment_Static{Static: &shared.StaticSegment{Id: "api_1", Name: "api"}}},
				{Segment: &shared.PathSegment_Static{Static: &shared.StaticSegment{Id: "v1_1", Name: "v1"}}},
				{Segment: &shared.PathSegment_Static{Static: &shared.StaticSegment{Id: "users_1", Name: "users"}}},
				{Segment: &shared.PathSegment_Param{Param: &shared.Parameter{
					Id:      "id_1",
					Name:    "id",
					Type:    shared.ParameterType_PARAMETER_TYPE_UUID,
					Example: "123e4567-e89b-12d3-a456-426614174000",
				}}},
				{Segment: &shared.PathSegment_Static{Static: &shared.StaticSegment{Id: "roles_1", Name: "roles"}}},
			},
			Edges: []*shared.Edge{
				{From: "api_1", To: "v1_1"},
				{From: "v1_1", To: "users_1"},
				{From: "users_1", To: "id_1"},
				{From: "id_1", To: "roles_1"},
			},
			Operations: []*shared.Operation{
				{
					Id:            "op_roles_1",
					Method:        "GET",
					PathSegmentId: "roles_1",
				},
			},
		},
	}, nil
}
