package main

import (
	"fmt"
	"runtime/debug"
)

const serviceName = "apim_har_profiler"

func getSwaggerVersion() string {
	const moduleName = "github.com/voikin/apim-proto/gen/go"

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "latest"
	}

	for _, dep := range info.Deps {
		if dep.Path == moduleName {
			return dep.Version
		}
	}

	return "unknown"
}

func getSwaggerURL() string {
	version := getSwaggerVersion()
	return fmt.Sprintf(
		"https://raw.githubusercontent.com/voikin/apim-proto/gen/go/%s/gen/openapi/%s/v1/%s.swagger.json", //nolint:lll // single string
		version,
		serviceName,
		serviceName,
	)
}
