package mutations

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/jmrGrav/hugo-mcp-go/internal/hugo/frontmatter"
)

const (
	maxFrontmatterBytes = 10 * 1024
	maxFrontmatterDepth = 3
)

var forbiddenFrontmatterFields = map[string]struct{}{
	"aliases":  {},
	"cascade":  {},
	"build":    {},
	"outputs":  {},
	"headless": {},
	"_target":  {},
}

func normalizeFrontmatterInput(raw any) (map[string]any, error) {
	if raw == nil {
		return map[string]any{}, nil
	}
	switch x := raw.(type) {
	case map[string]any:
		if x == nil {
			return map[string]any{}, nil
		}
		return x, nil
	case string:
		if x == "" {
			return map[string]any{}, nil
		}
		var out map[string]any
		if err := json.Unmarshal([]byte(x), &out); err != nil {
			return nil, fmt.Errorf("frontmatter must be a dict (or null)")
		}
		if out == nil {
			out = map[string]any{}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("frontmatter must be a dict (or null)")
	}
}

func validateFrontmatter(raw any, isUpdate bool, dedicated map[string]any) (map[string]any, error) {
	fm, err := normalizeFrontmatterInput(raw)
	if err != nil {
		return nil, err
	}
	if size := len(mustJSON(fm)); size > maxFrontmatterBytes {
		return nil, fmt.Errorf("frontmatter too large: %d bytes (max %d)", size, maxFrontmatterBytes)
	}
	forbidden := make([]string, 0)
	for key := range fm {
		if _, ok := forbiddenFrontmatterFields[key]; ok {
			forbidden = append(forbidden, key)
		}
	}
	if len(forbidden) > 0 {
		sort.Strings(forbidden)
		return nil, fmt.Errorf("Forbidden frontmatter fields (security): %s", joinComma(forbidden))
	}
	if !isUpdate {
		for key, value := range dedicated {
			if value == nil || isZeroLike(value) {
				continue
			}
			if _, ok := fm[key]; ok {
				return nil, fmt.Errorf("Conflict: field(s) provided both as dedicated param and in frontmatter: %s. Use only one.", key)
			}
		}
	}
	if isUpdate {
		if _, ok := fm["date"]; ok {
			return nil, fmt.Errorf("Field(s) cannot be modified via update_page: date")
		}
		for key, value := range dedicated {
			if value == nil || isZeroLike(value) {
				continue
			}
			if _, ok := fm[key]; ok {
				return nil, fmt.Errorf("Conflict: field(s) provided both as dedicated param and in frontmatter: %s. Use only one.", key)
			}
		}
	}
	if err := validateFrontmatterValueTree(fm, isUpdate, 1, ""); err != nil {
		return nil, err
	}
	return fm, nil
}

func validateFrontmatterValueTree(value any, isUpdate bool, depth int, path string) error {
	if depth > maxFrontmatterDepth {
		return fmt.Errorf("frontmatter too deep at '%s' (max depth %d)", path, maxFrontmatterDepth)
	}
	if value == nil {
		if !isUpdate {
			return fmt.Errorf("null not allowed at '%s' on create_page (only valid on update_page for field deletion)", path)
		}
		return nil
	}
	switch v := value.(type) {
	case string, int, int64, float64, bool:
		return nil
	case map[string]any:
		for k, child := range v {
			if err := validateFrontmatterValueTree(child, isUpdate, depth+1, joinPath(path, k)); err != nil {
				return err
			}
		}
		return nil
	case []any:
		for i, child := range v {
			if err := validateFrontmatterValueTree(child, isUpdate, depth+1, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
		return nil
	case []string:
		return nil
	default:
		return fmt.Errorf("Invalid type at '%s': %T (allowed: str, int, float, bool, list, dict, null)", path, value)
	}
}

func deepMerge(existing, updates map[string]any) map[string]any {
	out := frontmatter.CloneMap(existing)
	for k, v := range updates {
		if v == nil {
			delete(out, k)
			continue
		}
		next, ok := v.(map[string]any)
		if ok {
			if cur, ok := out[k].(map[string]any); ok {
				out[k] = deepMerge(cur, next)
				continue
			}
		}
		out[k] = v
	}
	return out
}

func mustJSON(v any) []byte {
	raw, _ := json.Marshal(v)
	return raw
}

func joinComma(items []string) string {
	if len(items) == 0 {
		return ""
	}
	out := items[0]
	for i := 1; i < len(items); i++ {
		out += ", " + items[i]
	}
	return out
}

func joinPath(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

func isZeroLike(v any) bool {
	switch x := v.(type) {
	case string:
		return x == ""
	case []string:
		return len(x) == 0
	case []any:
		return len(x) == 0
	case nil:
		return true
	default:
		return false
	}
}
