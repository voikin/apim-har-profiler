package controller

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/voikin/apim-proto/gen/go/shared/v1"
)

type harEntry struct {
	Log struct {
		Entries []struct {
			Request struct {
				Method string `json:"method"`
				URL    string `json:"url"`
			} `json:"request"`
			Response struct {
				Status int `json:"status"`
			} `json:"response"`
		} `json:"entries"`
	} `json:"log"`
}

func extractPathSegments(rawURL string) ([]string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	path := u.Path
	segments := strings.Split(strings.Trim(path, "/"), "/")
	return segments, nil
}

func guessParamType(s string) shared.ParameterType {
	switch {
	case isUUID(s):
		return shared.ParameterType_PARAMETER_TYPE_UUID
	case isInteger(s):
		return shared.ParameterType_PARAMETER_TYPE_INTEGER
	default:
		return shared.ParameterType_PARAMETER_TYPE_UNSPECIFIED
	}
}

func isUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

func isInteger(s string) bool {
	matched, _ := regexp.MatchString(`^\d+$`, s)
	return matched
}

type opKey struct {
	Method    string
	SegmentID string
}

func parseHARtoGraph(harJSON string) (*shared.APIGraph, error) {
	var har harEntry
	if err := json.Unmarshal([]byte(harJSON), &har); err != nil {
		return nil, fmt.Errorf("failed to parse HAR: %w", err)
	}

	segmentMap := map[string]string{} // key = name + position
	paramMap := map[string]string{}   // key = value + position
	idCounter := 1

	var segments []*shared.PathSegment
	var edges []*shared.Edge
	statusMap := map[opKey]map[int]struct{}{}

	for _, entry := range har.Log.Entries {
		method := entry.Request.Method
		pathSegs, err := extractPathSegments(entry.Request.URL)
		if err != nil {
			return nil, err
		}

		var prevSegmentID string
		var lastSegmentID string

		for i, seg := range pathSegs {
			isParam := isUUID(seg) || isInteger(seg)

			var segmentID string

			if isParam {
				key := fmt.Sprintf("param-%s-%d", seg, i)
				if existingID, ok := paramMap[key]; ok {
					segmentID = existingID
				} else {
					segmentID = fmt.Sprintf("param-%d", idCounter)
					paramType := guessParamType(seg)
					paramName := "id"
					if paramType == shared.ParameterType_PARAMETER_TYPE_UUID {
						paramName = "uuid"
					} else if paramType == shared.ParameterType_PARAMETER_TYPE_INTEGER {
						paramName = "int"
					}

					segments = append(segments, &shared.PathSegment{
						Segment: &shared.PathSegment_Param{
							Param: &shared.Parameter{
								Id:      segmentID,
								Name:    paramName,
								Type:    paramType,
								Example: seg,
							},
						},
					})
					paramMap[key] = segmentID
					idCounter++
				}
			} else {
				key := fmt.Sprintf("static-%s-%d", seg, i)
				if existingID, ok := segmentMap[key]; ok {
					segmentID = existingID
				} else {
					segmentID = fmt.Sprintf("static-%d", idCounter)

					segments = append(segments, &shared.PathSegment{
						Segment: &shared.PathSegment_Static{
							Static: &shared.StaticSegment{
								Id:   segmentID,
								Name: seg,
							},
						},
					})
					segmentMap[key] = segmentID
					idCounter++
				}
			}

			if i > 0 {
				edges = append(edges, &shared.Edge{
					From: prevSegmentID,
					To:   segmentID,
				})
			}
			prevSegmentID = segmentID
			lastSegmentID = segmentID
		}

		opK := opKey{Method: method, SegmentID: lastSegmentID}
		if _, ok := statusMap[opK]; !ok {
			statusMap[opK] = map[int]struct{}{}
		}
		statusMap[opK][entry.Response.Status] = struct{}{}
	}

	var operations []*shared.Operation
	for key, statuses := range statusMap {
		var codes []int32
		for code := range statuses {
			codes = append(codes, int32(code))
		}
		opID := fmt.Sprintf("op-%s-%s", strings.ToLower(key.Method), key.SegmentID)
		operations = append(operations, &shared.Operation{
			Id:            opID,
			Method:        key.Method,
			PathSegmentId: key.SegmentID,
			StatusCodes:   codes,
		})
	}

	return &shared.APIGraph{
		Segments:   segments,
		Edges:      edges,
		Operations: operations,
	}, nil
}
