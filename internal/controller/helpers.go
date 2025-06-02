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
		return nil, fmt.Errorf("url.Parse: %w", err)
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

func normalizePath(segments []string) string {
	var parts []string
	for _, s := range segments {
		if isUUID(s) || isInteger(s) {
			parts = append(parts, "{param}")
		} else {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "/")
}

type opKey struct {
	Method string
	Path   string
}

func parseHARtoGraph(harJSON string, isSequence bool) (*shared.APIGraph, error) {
	var har harEntry
	if err := json.Unmarshal([]byte(harJSON), &har); err != nil {
		return nil, fmt.Errorf("failed to parse HAR: %w", err)
	}

	segmentMap := map[string]string{}
	paramMap := map[string]string{}
	pathToSegmentID := map[string]string{}
	edgeSet := map[string]bool{}

	var segments []*shared.PathSegment
	var edges []*shared.Edge
	statusMap := map[opKey]map[int]struct{}{}
	opKeyToOpID := map[opKey]string{}

	for _, entry := range har.Log.Entries {
		method := entry.Request.Method
		pathSegs, err := extractPathSegments(entry.Request.URL)
		if err != nil {
			return nil, err
		}

		var prevSegmentID string
		var segmentIDs []string

		for i, seg := range pathSegs {
			pathPrefix := pathSegs[:i+1]
			isParam := isUUID(seg) || isInteger(seg)
			var segmentID, segmentKey string

			if isParam {
				pathPrefix[len(pathPrefix)-1] = "{param}"
				paramType := guessParamType(seg)
				segmentKey = fmt.Sprintf("param::%s::%d", strings.Join(pathPrefix, "/"), paramType)
				if existingID, ok := paramMap[segmentKey]; ok {
					segmentID = existingID
				} else {
					segmentID = segmentKey
					paramName := "param"
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
					paramMap[segmentKey] = segmentID
				}
			} else {
				segmentKey = fmt.Sprintf("static::%s", strings.Join(pathPrefix, "/"))
				if existingID, ok := segmentMap[segmentKey]; ok {
					segmentID = existingID
				} else {
					segmentID = segmentKey
					segments = append(segments, &shared.PathSegment{
						Segment: &shared.PathSegment_Static{
							Static: &shared.StaticSegment{
								Id:   segmentID,
								Name: seg,
							},
						},
					})
					segmentMap[segmentKey] = segmentID
				}
			}

			if i > 0 {
				edgeKey := fmt.Sprintf("%s->%s", prevSegmentID, segmentID)
				if !edgeSet[edgeKey] {
					edges = append(edges, &shared.Edge{
						From: prevSegmentID,
						To:   segmentID,
					})
					edgeSet[edgeKey] = true
				}
			}
			prevSegmentID = segmentID
			segmentIDs = append(segmentIDs, segmentID)
		}

		normalized := normalizePath(pathSegs)
		lastSegmentID := segmentIDs[len(segmentIDs)-1]

		if _, exists := pathToSegmentID[normalized]; !exists {
			pathToSegmentID[normalized] = lastSegmentID
		}

		key := opKey{
			Method: method,
			Path:   normalized,
		}
		if _, ok := statusMap[key]; !ok {
			statusMap[key] = map[int]struct{}{}
		}
		statusMap[key][entry.Response.Status] = struct{}{}
	}

	var operations []*shared.Operation
	for key, statuses := range statusMap {
		var codes []int32
		for code := range statuses {
			codes = append(codes, int32(code))
		}
		segmentID := pathToSegmentID[key.Path]
		opID := fmt.Sprintf("op::%s::%s", strings.ToLower(key.Method), segmentID)

		operations = append(operations, &shared.Operation{
			Id:            opID,
			Method:        key.Method,
			PathSegmentId: segmentID,
			StatusCodes:   codes,
		})
		opKeyToOpID[key] = opID
	}

	var transitions []*shared.Transition
	if isSequence {
		transitions = buildTransitions(&har, opKeyToOpID)
	}

	return &shared.APIGraph{
		Segments:    segments,
		Edges:       edges,
		Operations:  operations,
		Transitions: transitions,
	}, nil
}


func buildTransitions(har *harEntry, opKeyToOpID map[opKey]string) []*shared.Transition {
	var transitions []*shared.Transition
	var prevOpID string

	for _, entry := range har.Log.Entries {
		method := entry.Request.Method
		urlPath, err := extractPathSegments(entry.Request.URL)
		if err != nil || len(urlPath) == 0 {
			continue
		}
		normalized := normalizePath(urlPath)
		key := opKey{Method: method, Path: normalized}

		opID, ok := opKeyToOpID[key]
		if !ok {
			continue
		}

		if prevOpID != "" {
			transitions = append(transitions, &shared.Transition{
				From: prevOpID,
				To:   opID,
			})
		}
		prevOpID = opID
	}

	return transitions
}
