package main

import (
	"net/url"
	"strings"
)

const (
	defaultWebDAVPath         = "/dav/"
	nextcloudExportPagePrefix = "/apps/betternascontrolplane/exports/"
)

func mountProfilePathForExport(mountPath string) string {
	if strings.TrimSpace(mountPath) == "" {
		return defaultWebDAVPath
	}

	return mountPath
}

func cloudProfilePathForExport(exportID string) string {
	return nextcloudExportPagePrefix + url.PathEscape(exportID)
}
