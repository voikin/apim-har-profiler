package controller

import (
	harprofilerpb "github.com/voikin/apim-proto/gen/go/apim_har_profiler/v1"
)

type Controller struct {
	harprofilerpb.UnimplementedHARProfilerServiceServer
}

func New() *Controller {
	return &Controller{}
}
