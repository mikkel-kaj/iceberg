package compose

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGenerateComposeFile(t *testing.T) {
	spec := DeploySpec{Services: []ServiceSpec{{Name: "app", Image: "nginx:latest", Port: 80, Env: map[string]string{"A": "1"}, Volumes: []string{"./data:/data"}}}}
	body, err := GenerateComposeFile(spec)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "nginx:latest") || !strings.Contains(string(body), "80:80") {
		t.Fatalf("unexpected body: %s", string(body))
	}
	if !strings.Contains(string(body), "unless-stopped") || !strings.Contains(string(body), "driver: local") {
		t.Fatal("missing defaults")
	}
	var parsed any
	if err := yaml.Unmarshal(body, &parsed); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateComposeFileMultiService(t *testing.T) {
	spec := DeploySpec{Services: []ServiceSpec{{Name: "app", Image: "app", DependsOn: []string{"db"}}, {Name: "db", Image: "postgres:16"}}}
	body, err := GenerateComposeFile(spec)
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	if !strings.Contains(s, "depends_on") || !strings.Contains(s, "postgres:16") {
		t.Fatalf("unexpected body: %s", s)
	}
}
