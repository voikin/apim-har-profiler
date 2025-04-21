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

func parseHARtoGraph(harJSON string) (*shared.APIGraph, error) {
	var har harEntry
	if err := json.Unmarshal([]byte(harJSON), &har); err != nil {
		return nil, fmt.Errorf("failed to parse HAR: %w", err)
	}

	segmentMap := map[string]string{}
	paramMap := map[string]string{}
	pathToSegmentID := map[string]string{}
	edgeSet := map[string]bool{} // for deduplication

	idCounter := 1
	var segments []*shared.PathSegment
	var edges []*shared.Edge
	statusMap := map[opKey]map[int]struct{}{}
	pathMap := map[opKey]string{} // full path for debug

	for _, entry := range har.Log.Entries {
		method := entry.Request.Method
		pathSegs, err := extractPathSegments(entry.Request.URL)
		if err != nil {
			return nil, err
		}

		var prevSegmentID string
		var segmentIDs []string

		for i, seg := range pathSegs {
			isParam := isUUID(seg) || isInteger(seg)
			var segmentID string

			if isParam {
				paramType := guessParamType(seg)
				paramKey := fmt.Sprintf("param-%s-%d", normalizePath(pathSegs[:i]), paramType)

				if existingID, ok := paramMap[paramKey]; ok {
					segmentID = existingID
				} else {
					segmentID = fmt.Sprintf("param-%d", idCounter)
					paramName := "{param}"
					if paramType == shared.ParameterType_PARAMETER_TYPE_UUID {
						paramName = "{uuid}"
					} else if paramType == shared.ParameterType_PARAMETER_TYPE_INTEGER {
						paramName = "{int}"
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
					paramMap[paramKey] = segmentID
					idCounter++
				}
			} else {
				staticKey := fmt.Sprintf("static-%s-%d", seg, i)
				if existingID, ok := segmentMap[staticKey]; ok {
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
					segmentMap[staticKey] = segmentID
					idCounter++
				}
			}

			if i > 0 {
				edgeKey := fmt.Sprintf("%s-%s", prevSegmentID, segmentID)
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

		// сохраняем только если еще нет
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
		pathMap[key] = strings.Join(pathSegs, "/")
	}

	var operations []*shared.Operation
	for key, statuses := range statusMap {
		var codes []int32
		for code := range statuses {
			codes = append(codes, int32(code))
		}
		segmentID := pathToSegmentID[key.Path]
		opID := fmt.Sprintf("op-%s-%s", strings.ToLower(key.Method), strings.ReplaceAll(key.Path, "/", "-"))

		operations = append(operations, &shared.Operation{
			Id:            opID,
			Method:        key.Method,
			PathSegmentId: segmentID,
			StatusCodes:   codes,
			// fullPath можно добавить как кастомное поле в protobuf при необходимости
		})
	}

	return &shared.APIGraph{
		Segments:   segments,
		Edges:      edges,
		Operations: operations,
	}, nil
}
