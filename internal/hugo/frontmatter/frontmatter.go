package frontmatter

import (
	"bytes"
	"errors"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func Split(raw []byte) (map[string]any, string, error) {
	if len(raw) == 0 {
		return map[string]any{}, "", nil
	}
	content := string(raw)
	if !strings.HasPrefix(content, "---") {
		return map[string]any{}, content, nil
	}
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return map[string]any{}, content, nil
	}
	fm, _ := ParseYAML(parts[1])
	return fm, strings.TrimSpace(parts[2]), nil
}

func ParseFile(path string) (map[string]any, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	return Split(raw)
}

func ParseYAML(raw string) (map[string]any, error) {
	fm := map[string]any{}
	if strings.TrimSpace(raw) == "" {
		return fm, nil
	}
	if err := yaml.NewDecoder(strings.NewReader(raw)).Decode(&fm); err != nil {
		return map[string]any{}, nil
	}
	if fm == nil {
		fm = map[string]any{}
	}
	return fm, nil
}

func MarshalYAML(fm map[string]any) ([]byte, error) {
	if fm == nil {
		fm = map[string]any{}
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	defer enc.Close()
	if err := enc.Encode(fm); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Render(fm map[string]any, content string) ([]byte, error) {
	body, err := MarshalYAML(fm)
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return []byte(content), nil
	}
	var out bytes.Buffer
	out.WriteString("---\n")
	out.Write(body)
	out.WriteString("---\n\n")
	out.WriteString(content)
	if !strings.HasSuffix(content, "\n") {
		out.WriteString("\n")
	}
	return out.Bytes(), nil
}

func CloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

var _ = errors.New
