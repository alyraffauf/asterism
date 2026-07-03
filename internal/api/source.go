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

	switch {
	case rawPath == "":
		return "", "", fmt.Errorf("source is missing a path after ':'")
	case rawPath == ".":
		return collection, ".", nil
	case strings.HasPrefix(rawPath, "."):
		return "", "", fmt.Errorf("source path must not start with '.'")
	default:
		return collection, "." + rawPath, nil
	}
}
