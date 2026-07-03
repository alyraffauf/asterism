package backlink

import (
	"github.com/bluesky-social/indigo/atproto/syntax"
)

func isLinkTarget(s string) bool {
	if _, err := syntax.ParseATURI(s); err == nil {
		return true
	}
	if _, err := syntax.ParseDID(s); err == nil {
		return true
	}
	if _, err := syntax.ParseURI(s); err == nil {
		return true
	}
	return false
}


func tryStrongRef(obj map[string]any) (target string, targetCid string, ok bool) {
	if len(obj) != 2 {
		return "", "", false
	}

	uri, isURI := obj["uri"].(string)
	c, isCid := obj["cid"].(string)

	if !isURI || !isCid {
		return "", "", false
	}

	if !isLinkTarget(uri) {
		return "", "", false
	}

	return uri, c, true
}

func joinPath(base, key string) string {
	return base + "." + key
}

func arrayPathSuffix(val any) string {
	if obj, ok := val.(map[string]any); ok {
		if t, ok := obj["$type"].(string); ok {
			return "[" + t + "]"
		}
	}
	return "[]"
}


func walk(path string, value any, base Link) []Link {
	switch v := value.(type) {
	case map[string]any:
		if target, targetCid, ok := tryStrongRef(v); ok {
			link := base
			link.FieldPath = joinPath(path, "uri")
			link.Target = target
			link.TargetCid = targetCid
			return []Link{link}
		}

		var out []Link
		for key, val := range v {
			out = append(out, walk(joinPath(path, key), val, base)...)
		}
		return out

	case []any:
		var out []Link
		for _, val := range v {
			out = append(out, walk(path+arrayPathSuffix(val), val, base)...)
		}
		return out


	case string:
		if isLinkTarget(v) {
			link := base
			link.FieldPath = path
			link.Target = v
			return []Link{link}
		}
	}
	return nil
}

func Extract(record map[string]any, base Link) []Link {
	return walk("", record, base)
}
