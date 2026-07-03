package api

import (
	"fmt"
	"strings"
)

func parseSource(raw string) (collection string, path string, err error) {
	collection, rawPath, ok := strings.Cut(raw, ":")
	if !ok {
		return "", "", fmt.Errorf("source must contain a ':' separator")
	}
	if collection == "" {
		return "", "", fmt.Errorf("source is missing a collection before ':'")
	}

	path, err = normalizePath(rawPath)
	if err != nil {
		return "", "", fmt.Errorf("source path: %w", err)
	}

	return collection, path, nil
}

func normalizePath(rawPath string) (string, error) {
	switch {
	case rawPath == "":
		return "", fmt.Errorf("path is empty")
	case rawPath == ".":
		return ".", nil
	case strings.HasPrefix(rawPath, "."):
		return "", fmt.Errorf("path must not start with '.'")
	default:
		return "." + rawPath, nil
	}
}
