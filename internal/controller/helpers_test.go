package controller

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	sharedpb "github.com/voikin/apim-proto/gen/go/shared/v1"
)

func TestParseHARtoGraph_SimpleRequest(t *testing.T) {
	har := map[string]interface{}{
		"log": map[string]interface{}{
			"entries": []interface{}{
				map[string]interface{}{
					"request": map[string]interface{}{
						"method": "GET",
						"url":    "http://example.com/api/v1/users/12345",
					},
					"response": map[string]interface{}{
						"status": 200,
					},
				},
			},
		},
	}
	data, err := json.Marshal(har)
	require.NoError(t, err)

	graph, err := parseHARtoGraph(string(data), false)
	require.NoError(t, err)
	require.NotNil(t, graph)

	require.Len(t, graph.Segments, 4) // /api/v1/users/{int}
	require.Len(t, graph.Operations, 1)
	require.Len(t, graph.Edges, 3)

	// Проверка операции
	op := graph.Operations[0]
	require.Equal(t, "GET", op.Method)
	require.Len(t, op.StatusCodes, 1)
	require.Equal(t, int32(200), op.StatusCodes[0])
	require.Contains(t, op.Id, "op::get::") // ID операции

	// Проверка параметров
	lastSegment := findSegmentByID(graph, op.PathSegmentId)
	param, ok := lastSegment.Segment.(*sharedpb.PathSegment_Param)
	require.True(t, ok)
	require.Equal(t, "int", param.Param.Name)
	require.Equal(t, sharedpb.ParameterType_PARAMETER_TYPE_INTEGER, param.Param.Type)
	require.Equal(t, "12345", param.Param.Example)
}

func findSegmentByID(graph *sharedpb.APIGraph, id string) *sharedpb.PathSegment {
	for _, seg := range graph.Segments {
		switch s := seg.Segment.(type) {
		case *sharedpb.PathSegment_Static:
			if s.Static.Id == id {
				return seg
			}
		case *sharedpb.PathSegment_Param:
			if s.Param.Id == id {
				return seg
			}
		}
	}
	return nil
}

func TestParseHARtoGraph_WithTransitions(t *testing.T) {
	har := map[string]interface{}{
		"log": map[string]interface{}{
			"entries": []interface{}{
				map[string]interface{}{
					"request": map[string]interface{}{
						"method": "GET",
						"url":    "http://example.com/cart",
					},
					"response": map[string]interface{}{
						"status": 200,
					},
				},
				map[string]interface{}{
					"request": map[string]interface{}{
						"method": "POST",
						"url":    "http://example.com/cart/checkout",
					},
					"response": map[string]interface{}{
						"status": 201,
					},
				},
			},
		},
	}
	data, err := json.Marshal(har)
	require.NoError(t, err)

	graph, err := parseHARtoGraph(string(data), true)
	require.NoError(t, err)
	require.Len(t, graph.Operations, 2)
	require.Len(t, graph.Transitions, 1)

	tr := graph.Transitions[0]
	require.NotEmpty(t, tr.From)
	require.NotEmpty(t, tr.To)
	require.NotEqual(t, tr.From, tr.To)
}
