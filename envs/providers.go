package envs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

type SecretRef struct {
	Original string
	Scheme   string
	URL      *url.URL
	Optional bool
}

type Provider interface {
	Scheme() string
	Resolve(ctx context.Context, ref SecretRef) (string, error)
}

type ProviderRegistry map[string]Provider

func DefaultProviders() ProviderRegistry {
	providers := []Provider{
		OPProvider{},
		AWSSecretsManagerProvider{},
		BitwardenProvider{},
	}
	registry := make(ProviderRegistry, len(providers))
	for _, provider := range providers {
		registry[provider.Scheme()] = provider
	}
	return registry
}

func (r ProviderRegistry) Resolve(ctx context.Context, raw string, optional bool) (Value, bool, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return Value{}, false, err
	}
	provider, ok := r[u.Scheme]
	if !ok {
		return Value{}, false, fmt.Errorf("unsupported secret provider %q", u.Scheme)
	}
	resolved, err := provider.Resolve(ctx, SecretRef{Original: raw, Scheme: u.Scheme, URL: u, Optional: optional})
	if err != nil {
		if optional {
			return Value{}, false, nil
		}
		return Value{}, false, err
	}
	return Value{Value: resolved, Raw: "secret:" + raw, Secret: true, Resolved: true, Provider: u.Scheme}, true, nil
}

type OPProvider struct{}

func (OPProvider) Scheme() string { return "op" }

func (OPProvider) Resolve(ctx context.Context, ref SecretRef) (string, error) {
	cmd := exec.CommandContext(ctx, "op", "read", ref.Original)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

type AWSSecretsManagerProvider struct{}

func (AWSSecretsManagerProvider) Scheme() string { return "aws-secretsmanager" }

func (AWSSecretsManagerProvider) Resolve(ctx context.Context, ref SecretRef) (string, error) {
	secretID := strings.TrimPrefix(ref.URL.Path, "/")
	if ref.URL.Host != "" {
		if secretID == "" {
			secretID = ref.URL.Host
		} else {
			secretID = ref.URL.Host + "/" + secretID
		}
	}
	if secretID == "" {
		return "", errors.New("aws-secretsmanager secret id is required")
	}

	args := []string{"secretsmanager", "get-secret-value", "--secret-id", secretID, "--query", "SecretString", "--output", "text"}
	q := ref.URL.Query()
	if profile := q.Get("profile"); profile != "" {
		args = append(args, "--profile", profile)
	}
	if region := q.Get("region"); region != "" {
		args = append(args, "--region", region)
	}

	cmd := exec.CommandContext(ctx, "aws", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	secretString := strings.TrimRight(string(out), "\r\n")
	if ref.URL.Fragment == "" {
		return secretString, nil
	}
	return selectJSONPath(secretString, ref.URL.Fragment)
}

type BitwardenProvider struct{}

func (BitwardenProvider) Scheme() string { return "bitwarden" }

func (BitwardenProvider) Resolve(ctx context.Context, ref SecretRef) (string, error) {
	item := strings.TrimPrefix(ref.URL.Path, "/")
	if ref.URL.Host != "" {
		if item == "" {
			item = ref.URL.Host
		} else {
			item = ref.URL.Host + "/" + item
		}
	}
	if item == "" {
		return "", errors.New("bitwarden item is required")
	}
	cmd := exec.CommandContext(ctx, "bw", "get", "item", item)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return selectBitwardenField(out, ref.URL.Fragment)
}

func selectJSONPath(input string, path string) (string, error) {
	var cur interface{}
	if err := json.Unmarshal([]byte(input), &cur); err != nil {
		return "", fmt.Errorf("secret value is not valid JSON: %w", err)
	}
	for _, part := range strings.Split(path, ".") {
		m, ok := cur.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("JSON path %q not found", path)
		}
		cur, ok = m[part]
		if !ok {
			return "", fmt.Errorf("JSON path %q not found", path)
		}
	}
	switch v := cur.(type) {
	case string:
		return v, nil
	case float64, bool:
		return fmt.Sprintf("%v", v), nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}

func selectBitwardenField(input []byte, field string) (string, error) {
	var item map[string]interface{}
	if err := json.Unmarshal(input, &item); err != nil {
		return "", err
	}
	switch field {
	case "username", "password":
		login, ok := item["login"].(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("bitwarden login field %q not found", field)
		}
		v, ok := login[field].(string)
		if !ok {
			return "", fmt.Errorf("bitwarden field %q not found", field)
		}
		return v, nil
	case "notes":
		v, ok := item["notes"].(string)
		if !ok {
			return "", errors.New("bitwarden notes field not found")
		}
		return v, nil
	default:
		if strings.HasPrefix(field, "custom.") {
			name := strings.TrimPrefix(field, "custom.")
			fields, _ := item["fields"].([]interface{})
			for _, rawField := range fields {
				m, ok := rawField.(map[string]interface{})
				if !ok {
					continue
				}
				if m["name"] == name {
					if v, ok := m["value"].(string); ok {
						return v, nil
					}
				}
			}
		}
	}
	return "", fmt.Errorf("bitwarden field %q not found", field)
}
