package ui

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

type jsonTreeLine struct {
	path      string
	text      string
	value     interface{}
	canExpand bool
	expanded  bool
}

func buildJSONTreeLines(root interface{}, expanded map[string]bool) []jsonTreeLine {
	if expanded == nil {
		expanded = map[string]bool{}
	}
	if _, ok := expanded["$"]; !ok {
		expanded["$"] = true
	}
	lines := make([]jsonTreeLine, 0, 64)
	appendJSONTreeNode(&lines, root, "$", "$", nil, true, expanded)
	return lines
}

func appendJSONTreeNode(lines *[]jsonTreeLine, value interface{}, path, label string, ancestorsHasNext []bool, isLast bool, expanded map[string]bool) {
	canExpand := isExpandableJSONValue(value)
	isExpanded := canExpand && expanded[path]
	marker := "•"
	if canExpand {
		if isExpanded {
			marker = "▾"
		} else {
			marker = "▸"
		}
	}
	prefix := buildTreePrefix(ancestorsHasNext, isLast)
	typeLabel := jsonTypeLabel(value)
	text := fmt.Sprintf("%s%s %s %s", prefix, marker, label, summarizeJSONValue(value))
	if typeLabel != "" {
		text += "  [" + typeLabel + "]"
	}

	*lines = append(*lines, jsonTreeLine{
		path:      path,
		value:     value,
		canExpand: canExpand,
		expanded:  isExpanded,
		text:      text,
	})

	if !isExpanded {
		return
	}

	nextAncestors := append([]bool{}, ancestorsHasNext...)
	nextAncestors = append(nextAncestors, !isLast)

	if typed, ok := toMap(value); ok {
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for i, key := range keys {
			childPath := path + "." + key
			appendJSONTreeNode(lines, typed[key], childPath, key+":", nextAncestors, i == len(keys)-1, expanded)
		}
		return
	}
	if typed, ok := toSlice(value); ok {
		for i, child := range typed {
			childPath := fmt.Sprintf("%s[%d]", path, i)
			appendJSONTreeNode(lines, child, childPath, "["+strconv.Itoa(i)+"]:", nextAncestors, i == len(typed)-1, expanded)
		}
	}
}

func buildTreePrefix(ancestorsHasNext []bool, isLast bool) string {
	if len(ancestorsHasNext) == 0 {
		return ""
	}
	var sb strings.Builder
	for i := 0; i < len(ancestorsHasNext)-1; i++ {
		if ancestorsHasNext[i] {
			sb.WriteString("│  ")
		} else {
			sb.WriteString("   ")
		}
	}
	if isLast {
		sb.WriteString("└─ ")
	} else {
		sb.WriteString("├─ ")
	}
	return sb.String()
}

func isExpandableJSONValue(value interface{}) bool {
	if m, ok := toMap(value); ok {
		return len(m) > 0
	}
	if s, ok := toSlice(value); ok {
		return len(s) > 0
	}
	return false
}

func jsonTypeLabel(value interface{}) string {
	if _, ok := toMap(value); ok {
		return "object"
	}
	if _, ok := toSlice(value); ok {
		return "array"
	}
	switch value.(type) {
	case string:
		return "string"
	case bool:
		return "bool"
	case nil:
		return "null"
	case float64, float32, int, int64, uint64:
		return "number"
	default:
		return "value"
	}
}

func summarizeJSONValue(value interface{}) string {
	if typed, ok := toMap(value); ok {
		return fmt.Sprintf("{...} (%d keys)", len(typed))
	}
	if typed, ok := toSlice(value); ok {
		return fmt.Sprintf("[...] (%d items)", len(typed))
	}
	switch typed := value.(type) {
	case string:
		if len(typed) > 72 {
			return strconv.Quote(typed[:69] + "...")
		}
		return strconv.Quote(typed)
	case nil:
		return "null"
	default:
		return fmt.Sprintf("= %v", typed)
	}
}

func toMap(value interface{}) (map[string]interface{}, bool) {
	if value == nil {
		return nil, false
	}
	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Map {
		return nil, false
	}
	out := make(map[string]interface{}, v.Len())
	for _, key := range v.MapKeys() {
		out[fmt.Sprint(key.Interface())] = v.MapIndex(key).Interface()
	}
	return out, true
}

func toSlice(value interface{}) ([]interface{}, bool) {
	if value == nil {
		return nil, false
	}
	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil, false
	}
	out := make([]interface{}, v.Len())
	for i := 0; i < v.Len(); i++ {
		out[i] = v.Index(i).Interface()
	}
	return out, true
}

func formatJSONValueForCopy(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return typed
	case map[string]interface{}, []interface{}:
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return fmt.Sprintf("%v", value)
		}
		return string(data)
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%v", typed)
	}
}
