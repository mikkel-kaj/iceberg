package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

var runCommand = func(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

type bwItem struct {
	Login struct {
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"login"`
	Fields []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"fields"`
}

func ResolveBitwardenReference(ctx context.Context, ref string) (string, error) {
	itemID, selector, err := parseBitwardenReference(ref)
	if err != nil {
		return "", err
	}
	item, err := readBitwardenItem(ctx, itemID)
	if err != nil {
		return "", err
	}
	switch {
	case selector == "" || selector == "password":
		if item.Login.Password == "" {
			return "", fmt.Errorf("bitwarden item %q has no password value", itemID)
		}
		return item.Login.Password, nil
	case selector == "username":
		if item.Login.Username == "" {
			return "", fmt.Errorf("bitwarden item %q has no username value", itemID)
		}
		return item.Login.Username, nil
	case strings.HasPrefix(selector, "field:"):
		fieldName := strings.TrimPrefix(selector, "field:")
		for _, field := range item.Fields {
			if strings.EqualFold(field.Name, fieldName) {
				return field.Value, nil
			}
		}
		return "", fmt.Errorf("bitwarden item %q missing custom field %q", itemID, fieldName)
	default:
		return "", fmt.Errorf("unsupported bitwarden selector %q", selector)
	}
}

func parseBitwardenReference(ref string) (itemID, selector string, err error) {
	raw := ref
	raw = strings.TrimPrefix(raw, "bw://")
	raw = strings.TrimPrefix(raw, "bitwarden://")
	if raw == ref {
		return "", "", fmt.Errorf("not a bitwarden reference: %q", ref)
	}
	parts := strings.SplitN(raw, "#", 2)
	itemID = strings.TrimSpace(parts[0])
	if itemID == "" {
		return "", "", fmt.Errorf("bitwarden reference missing item id")
	}
	if len(parts) == 2 {
		selector = strings.TrimSpace(parts[1])
	}
	return itemID, selector, nil
}

func readBitwardenItem(ctx context.Context, itemID string) (*bwItem, error) {
	out, err := runBW(ctx, "get", "item", itemID, "--raw")
	if err != nil {
		return nil, err
	}
	var item bwItem
	if err := json.Unmarshal(out, &item); err != nil {
		return nil, fmt.Errorf("parse bitwarden item %q: %w", itemID, err)
	}
	return &item, nil
}

func runBW(ctx context.Context, args ...string) ([]byte, error) {
	out, err := runCommand(ctx, "bw", args...)
	if err == nil {
		return out, nil
	}
	if len(args) > 0 && args[len(args)-1] == "--raw" {
		fallback := append([]string{}, args[:len(args)-1]...)
		fallbackOut, fallbackErr := runCommand(ctx, "bw", fallback...)
		if fallbackErr == nil {
			return fallbackOut, nil
		}
		return nil, fmt.Errorf("bitwarden CLI command failed: %v (%s); fallback failed: %v (%s)", err, trimOutput(out), fallbackErr, trimOutput(fallbackOut))
	}
	return nil, fmt.Errorf("bitwarden CLI command failed: %w (%s)", err, trimOutput(out))
}

func trimOutput(out []byte) string {
	s := strings.TrimSpace(string(out))
	if s == "" {
		return "no output"
	}
	const max = 180
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
