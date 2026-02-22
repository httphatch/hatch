package caddy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/httphatch/hatch/internal/config"
)

// fullConfig returns a multi-service example config used in golden file tests.
func fullConfig() config.Config {
	return config.Config{
		Version: 1,
		Settings: config.Settings{
			TLD:       "test",
			HTTPPort:  80,
			HTTPSPort: 443,
			AutoStart: true,
			LogLevel:  "info",
		},
		Projects: map[string]config.Project{
			"acme": {
				Domain:  "acme.test",
				Path:    "/home/user/projects/acme",
				Enabled: true,
				Services: map[string]config.Service{
					"web": {Proxy: "http://localhost:3000"},
					"api": {Proxy: "http://localhost:8000", Route: "/api/*"},
					"ws":  {Proxy: "http://localhost:6001", Subdomain: "ws", WebSocket: true},
				},
			},
		},
	}
}

func TestTranslate_SingleService(t *testing.T) {
	cfg := config.Config{
		Version: 1,
		Settings: config.Settings{
			HTTPPort:  80,
			HTTPSPort: 443,
		},
		Projects: map[string]config.Project{
			"myapp": {
				Domain:  "myapp.test",
				Path:    "/path/to/myapp",
				Enabled: true,
				Services: map[string]config.Service{
					"web": {Proxy: "http://localhost:3000"},
				},
			},
		},
	}

	result := Translate(cfg, PKIPaths{}, "/test/data/caddy")

	servers := result["apps"].(map[string]any)["http"].(map[string]any)["servers"].(map[string]any)
	httpsServer := servers["hatch_https"].(map[string]any)
	routes := httpsServer["routes"].([]map[string]any)

	// 1 project route + 1 fallback route.
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routes))
	}

	route := routes[0]
	match := route["match"].([]map[string]any)[0]
	hosts := match["host"].([]string)
	if hosts[0] != "myapp.test" {
		t.Errorf("expected host myapp.test, got %s", hosts[0])
	}

	subroute := route["handle"].([]map[string]any)[0]
	if subroute["handler"] != "subroute" {
		t.Fatalf("expected handler subroute, got %s", subroute["handler"])
	}

	subRoutes := subroute["routes"].([]map[string]any)
	if len(subRoutes) != 2 {
		t.Fatalf("expected 2 subroutes (preflight + proxy), got %d", len(subRoutes))
	}

	// First subroute: OPTIONS preflight.
	preflightMatch := subRoutes[0]["match"].([]map[string]any)[0]
	if methods := preflightMatch["method"].([]string); methods[0] != "OPTIONS" {
		t.Errorf("expected preflight method OPTIONS, got %s", methods[0])
	}
	preflightHandler := subRoutes[0]["handle"].([]map[string]any)[0]
	if preflightHandler["status_code"] != "204" {
		t.Errorf("expected preflight status 204, got %v", preflightHandler["status_code"])
	}

	// Second subroute: headers + reverse_proxy.
	proxyHandlers := subRoutes[1]["handle"].([]map[string]any)
	if len(proxyHandlers) != 2 {
		t.Fatalf("expected 2 handlers (headers + reverse_proxy), got %d", len(proxyHandlers))
	}
	if proxyHandlers[0]["handler"] != "headers" {
		t.Errorf("expected headers handler, got %s", proxyHandlers[0]["handler"])
	}
	if proxyHandlers[1]["handler"] != "reverse_proxy" {
		t.Errorf("expected reverse_proxy handler, got %s", proxyHandlers[1]["handler"])
	}
	upstreams := proxyHandlers[1]["upstreams"].([]map[string]any)
	if upstreams[0]["dial"] != "localhost:3000" {
		t.Errorf("expected dial localhost:3000, got %s", upstreams[0]["dial"])
	}

	if route["terminal"] != true {
		t.Error("expected terminal: true")
	}
}

func TestTranslate_PathRouting(t *testing.T) {
	cfg := config.Config{
		Version: 1,
		Settings: config.Settings{
			HTTPPort:  80,
			HTTPSPort: 443,
		},
		Projects: map[string]config.Project{
			"myapp": {
				Domain:  "myapp.test",
				Path:    "/path/to/myapp",
				Enabled: true,
				Services: map[string]config.Service{
					"api": {Proxy: "http://localhost:8000", Route: "/api/*"},
				},
			},
		},
	}

	result := Translate(cfg, PKIPaths{}, "/test/data/caddy")

	servers := result["apps"].(map[string]any)["http"].(map[string]any)["servers"].(map[string]any)
	routes := servers["hatch_https"].(map[string]any)["routes"].([]map[string]any)

	// 1 project route + 1 fallback route.
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routes))
	}

	match := routes[0]["match"].([]map[string]any)[0]
	paths, ok := match["path"].([]string)
	if !ok {
		t.Fatal("expected path matcher to be present")
	}
	if paths[0] != "/api/*" {
		t.Errorf("expected path /api/*, got %s", paths[0])
	}
}

func TestTranslate_SubdomainRouting(t *testing.T) {
	cfg := config.Config{
		Version: 1,
		Settings: config.Settings{
			HTTPPort:  80,
			HTTPSPort: 443,
		},
		Projects: map[string]config.Project{
			"myapp": {
				Domain:  "myapp.test",
				Path:    "/path/to/myapp",
				Enabled: true,
				Services: map[string]config.Service{
					"docs": {Proxy: "http://localhost:4000", Subdomain: "docs"},
				},
			},
		},
	}

	result := Translate(cfg, PKIPaths{}, "/test/data/caddy")

	servers := result["apps"].(map[string]any)["http"].(map[string]any)["servers"].(map[string]any)
	routes := servers["hatch_https"].(map[string]any)["routes"].([]map[string]any)

	// 1 project route + 1 fallback route.
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routes))
	}

	match := routes[0]["match"].([]map[string]any)[0]
	hosts := match["host"].([]string)
	if hosts[0] != "docs.myapp.test" {
		t.Errorf("expected host docs.myapp.test, got %s", hosts[0])
	}
}

func TestTranslate_WebSocket(t *testing.T) {
	cfg := config.Config{
		Version: 1,
		Settings: config.Settings{
			HTTPPort:  80,
			HTTPSPort: 443,
		},
		Projects: map[string]config.Project{
			"myapp": {
				Domain:  "myapp.test",
				Path:    "/path/to/myapp",
				Enabled: true,
				Services: map[string]config.Service{
					"ws": {Proxy: "http://localhost:6001", Subdomain: "ws", WebSocket: true},
				},
			},
		},
	}

	result := Translate(cfg, PKIPaths{}, "/test/data/caddy")

	servers := result["apps"].(map[string]any)["http"].(map[string]any)["servers"].(map[string]any)
	routes := servers["hatch_https"].(map[string]any)["routes"].([]map[string]any)

	// 1 project route + 1 fallback route.
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routes))
	}

	subroute := routes[0]["handle"].([]map[string]any)[0]
	if subroute["handler"] != "subroute" {
		t.Fatalf("expected subroute handler, got %s", subroute["handler"])
	}

	// The reverse_proxy handler is the second handler in the second subroute.
	subRoutes := subroute["routes"].([]map[string]any)
	proxyHandlers := subRoutes[1]["handle"].([]map[string]any)
	handler := proxyHandlers[1]

	flush, ok := handler["flush_interval"]
	if !ok {
		t.Fatal("expected flush_interval to be set")
	}
	if flush != -1 {
		t.Errorf("expected flush_interval -1, got %v", flush)
	}

	headers := handler["headers"].(map[string]any)
	reqHeaders := headers["request"].(map[string]any)
	setHeaders := reqHeaders["set"].(map[string]any)

	connHeader := setHeaders["Connection"].([]string)
	if connHeader[0] != "{http.request.header.Connection}" {
		t.Errorf("unexpected Connection header: %s", connHeader[0])
	}

	upgradeHeader := setHeaders["Upgrade"].([]string)
	if upgradeHeader[0] != "{http.request.header.Upgrade}" {
		t.Errorf("unexpected Upgrade header: %s", upgradeHeader[0])
	}
}

func TestTranslate_RouteOrdering(t *testing.T) {
	cfg := fullConfig()
	result := Translate(cfg, PKIPaths{}, "/test/data/caddy")

	servers := result["apps"].(map[string]any)["http"].(map[string]any)["servers"].(map[string]any)
	routes := servers["hatch_https"].(map[string]any)["routes"].([]map[string]any)

	// 3 project routes + 1 fallback route.
	if len(routes) != 4 {
		t.Fatalf("expected 4 routes, got %d", len(routes))
	}

	// Route 0: subdomain (ws.acme.test)
	host0 := routes[0]["match"].([]map[string]any)[0]["host"].([]string)[0]
	if host0 != "ws.acme.test" {
		t.Errorf("route 0: expected ws.acme.test, got %s", host0)
	}

	// Route 1: path (/api/*)
	match1 := routes[1]["match"].([]map[string]any)[0]
	host1 := match1["host"].([]string)[0]
	path1 := match1["path"].([]string)[0]
	if host1 != "acme.test" {
		t.Errorf("route 1: expected acme.test, got %s", host1)
	}
	if path1 != "/api/*" {
		t.Errorf("route 1: expected /api/*, got %s", path1)
	}

	// Route 2: catch-all (acme.test)
	match2 := routes[2]["match"].([]map[string]any)[0]
	host2 := match2["host"].([]string)[0]
	if host2 != "acme.test" {
		t.Errorf("route 2: expected acme.test, got %s", host2)
	}
	if _, hasPath := match2["path"]; hasPath {
		t.Error("route 2: catch-all should not have path matcher")
	}

	// Route 3: fallback (no match key).
	if _, hasMatch := routes[3]["match"]; hasMatch {
		t.Error("route 3: fallback should not have match key")
	}
}

func TestTranslate_DisabledSkipped(t *testing.T) {
	cfg := config.Config{
		Version: 1,
		Settings: config.Settings{
			HTTPPort:  80,
			HTTPSPort: 443,
		},
		Projects: map[string]config.Project{
			"disabled": {
				Domain:  "disabled.test",
				Path:    "/path/to/disabled",
				Enabled: false,
				Services: map[string]config.Service{
					"web": {Proxy: "http://localhost:3000"},
				},
			},
		},
	}

	result := Translate(cfg, PKIPaths{}, "/test/data/caddy")

	servers := result["apps"].(map[string]any)["http"].(map[string]any)["servers"].(map[string]any)
	routes := servers["hatch_https"].(map[string]any)["routes"].([]map[string]any)

	if len(routes) != 0 {
		t.Errorf("expected 0 routes for disabled project, got %d", len(routes))
	}
}

func TestTranslate_MultipleProjects(t *testing.T) {
	cfg := config.Config{
		Version: 1,
		Settings: config.Settings{
			HTTPPort:  80,
			HTTPSPort: 443,
		},
		Projects: map[string]config.Project{
			"alpha": {
				Domain:  "alpha.test",
				Path:    "/path/to/alpha",
				Enabled: true,
				Services: map[string]config.Service{
					"web": {Proxy: "http://localhost:3000"},
				},
			},
			"beta": {
				Domain:  "beta.test",
				Path:    "/path/to/beta",
				Enabled: true,
				Services: map[string]config.Service{
					"web": {Proxy: "http://localhost:4000"},
				},
			},
		},
	}

	result := Translate(cfg, PKIPaths{}, "/test/data/caddy")

	servers := result["apps"].(map[string]any)["http"].(map[string]any)["servers"].(map[string]any)
	routes := servers["hatch_https"].(map[string]any)["routes"].([]map[string]any)

	// 2 project routes + 1 fallback route.
	if len(routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(routes))
	}

	// Routes should be alphabetical (both are catch-all tier).
	host0 := routes[0]["match"].([]map[string]any)[0]["host"].([]string)[0]
	host1 := routes[1]["match"].([]map[string]any)[0]["host"].([]string)[0]
	if host0 != "alpha.test" {
		t.Errorf("route 0: expected alpha.test, got %s", host0)
	}
	if host1 != "beta.test" {
		t.Errorf("route 1: expected beta.test, got %s", host1)
	}
}

func TestTranslate_HTTPRedirects(t *testing.T) {
	cfg := fullConfig()
	result := Translate(cfg, PKIPaths{}, "/test/data/caddy")

	servers := result["apps"].(map[string]any)["http"].(map[string]any)["servers"].(map[string]any)
	httpServer := servers["hatch_http"].(map[string]any)
	routes := httpServer["routes"].([]map[string]any)

	if len(routes) != 1 {
		t.Fatalf("expected 1 redirect route, got %d", len(routes))
	}

	match := routes[0]["match"].([]map[string]any)[0]
	hosts := match["host"].([]string)

	// Should include base domain and subdomain (sorted).
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts in redirect, got %d", len(hosts))
	}
	if hosts[0] != "acme.test" {
		t.Errorf("expected acme.test, got %s", hosts[0])
	}
	if hosts[1] != "ws.acme.test" {
		t.Errorf("expected ws.acme.test, got %s", hosts[1])
	}

	handler := routes[0]["handle"].([]map[string]any)[0]
	if handler["handler"] != "static_response" {
		t.Errorf("expected static_response handler, got %s", handler["handler"])
	}
	if handler["status_code"] != "302" {
		t.Errorf("expected status 302, got %s", handler["status_code"])
	}
}

func TestTranslate_TLSAutomation(t *testing.T) {
	cfg := fullConfig()
	result := Translate(cfg, PKIPaths{}, "/test/data/caddy")

	tls := result["apps"].(map[string]any)["tls"].(map[string]any)
	automation := tls["automation"].(map[string]any)
	policies := automation["policies"].([]map[string]any)

	if len(policies) != 1 {
		t.Fatalf("expected 1 TLS policy, got %d", len(policies))
	}

	subjects := policies[0]["subjects"].([]string)
	// Includes *.test wildcard for fallback route TLS handshake.
	want := []string{"*.test", "acme.test", "ws.acme.test"}
	if len(subjects) != len(want) {
		t.Fatalf("expected %d TLS subjects, got %d: %v", len(want), len(subjects), subjects)
	}
	for i, w := range want {
		if subjects[i] != w {
			t.Errorf("subject %d: expected %s, got %s", i, w, subjects[i])
		}
	}

	issuers := policies[0]["issuers"].([]map[string]any)
	if issuers[0]["module"] != "internal" {
		t.Errorf("expected internal issuer, got %s", issuers[0]["module"])
	}
}

func TestTranslate_CustomPorts(t *testing.T) {
	cfg := config.Config{
		Version: 1,
		Settings: config.Settings{
			HTTPPort:  8080,
			HTTPSPort: 8443,
		},
		Projects: map[string]config.Project{},
	}

	result := Translate(cfg, PKIPaths{}, "/test/data/caddy")

	servers := result["apps"].(map[string]any)["http"].(map[string]any)["servers"].(map[string]any)

	httpsListen := servers["hatch_https"].(map[string]any)["listen"].([]string)
	if httpsListen[0] != ":8443" {
		t.Errorf("expected :8443, got %s", httpsListen[0])
	}

	httpListen := servers["hatch_http"].(map[string]any)["listen"].([]string)
	if httpListen[0] != ":8080" {
		t.Errorf("expected :8080, got %s", httpListen[0])
	}
}

func TestTranslate_EmptyConfig(t *testing.T) {
	cfg := config.Config{
		Version: 1,
		Settings: config.Settings{
			HTTPPort:  80,
			HTTPSPort: 443,
		},
		Projects: map[string]config.Project{},
	}

	result := Translate(cfg, PKIPaths{}, "/test/data/caddy")

	// Should still produce valid structure.
	servers := result["apps"].(map[string]any)["http"].(map[string]any)["servers"].(map[string]any)

	httpsRoutes := servers["hatch_https"].(map[string]any)["routes"].([]map[string]any)
	if len(httpsRoutes) != 0 {
		t.Errorf("expected 0 HTTPS routes, got %d", len(httpsRoutes))
	}

	httpRoutes := servers["hatch_http"].(map[string]any)["routes"].([]map[string]any)
	if len(httpRoutes) != 0 {
		t.Errorf("expected 0 HTTP routes, got %d", len(httpRoutes))
	}

	tls := result["apps"].(map[string]any)["tls"].(map[string]any)
	policies := tls["automation"].(map[string]any)["policies"].([]map[string]any)
	if len(policies) != 0 {
		t.Errorf("expected 0 TLS policies, got %d", len(policies))
	}
}

func TestTranslate_PKIConfig(t *testing.T) {
	cfg := config.Config{
		Version: 1,
		Settings: config.Settings{
			HTTPPort:  80,
			HTTPSPort: 443,
		},
		Projects: map[string]config.Project{
			"myapp": {
				Domain:  "myapp.test",
				Path:    "/path/to/myapp",
				Enabled: true,
				Services: map[string]config.Service{
					"web": {Proxy: "http://localhost:3000"},
				},
			},
		},
	}

	pkiPaths := PKIPaths{
		RootCert:         "/home/user/.hatch/certs/rootCA.pem",
		RootKey:          "/home/user/.hatch/certs/rootCA-key.pem",
		IntermediateCert: "/home/user/.hatch/certs/intermediateCA.pem",
		IntermediateKey:  "/home/user/.hatch/certs/intermediateCA-key.pem",
	}

	result := Translate(cfg, pkiPaths, "/test/data/caddy")

	apps := result["apps"].(map[string]any)

	// PKI app should be present.
	pki, ok := apps["pki"].(map[string]any)
	if !ok {
		t.Fatal("expected pki app to be present")
	}
	cas := pki["certificate_authorities"].(map[string]any)
	hatchCA := cas["hatch"].(map[string]any)
	if hatchCA["name"] != "Hatch Local CA" {
		t.Errorf("expected CA name 'Hatch Local CA', got %s", hatchCA["name"])
	}
	root := hatchCA["root"].(map[string]any)
	if root["certificate"] != pkiPaths.RootCert {
		t.Errorf("expected certificate %s, got %s", pkiPaths.RootCert, root["certificate"])
	}
	if root["private_key"] != pkiPaths.RootKey {
		t.Errorf("expected private_key %s, got %s", pkiPaths.RootKey, root["private_key"])
	}

	// Intermediate should be present.
	intermediate := hatchCA["intermediate"].(map[string]any)
	if intermediate["certificate"] != pkiPaths.IntermediateCert {
		t.Errorf("expected intermediate certificate %s, got %s", pkiPaths.IntermediateCert, intermediate["certificate"])
	}
	if intermediate["private_key"] != pkiPaths.IntermediateKey {
		t.Errorf("expected intermediate private_key %s, got %s", pkiPaths.IntermediateKey, intermediate["private_key"])
	}

	// TLS issuer should reference "hatch" CA.
	tls := apps["tls"].(map[string]any)
	policies := tls["automation"].(map[string]any)["policies"].([]map[string]any)
	issuers := policies[0]["issuers"].([]map[string]any)
	if issuers[0]["module"] != "internal" {
		t.Errorf("expected internal issuer module, got %s", issuers[0]["module"])
	}
	if issuers[0]["ca"] != "hatch" {
		t.Errorf("expected ca 'hatch', got %v", issuers[0]["ca"])
	}
}

func TestTranslate_PKIConfig_RootOnlyNoIntermediate(t *testing.T) {
	cfg := config.Config{
		Version: 1,
		Settings: config.Settings{
			HTTPPort:  80,
			HTTPSPort: 443,
		},
		Projects: map[string]config.Project{
			"myapp": {
				Domain:  "myapp.test",
				Path:    "/path/to/myapp",
				Enabled: true,
				Services: map[string]config.Service{
					"web": {Proxy: "http://localhost:3000"},
				},
			},
		},
	}

	pkiPaths := PKIPaths{
		RootCert: "/home/user/.hatch/certs/rootCA.pem",
		RootKey:  "/home/user/.hatch/certs/rootCA-key.pem",
	}

	result := Translate(cfg, pkiPaths, "/test/data/caddy")

	apps := result["apps"].(map[string]any)
	pki := apps["pki"].(map[string]any)
	cas := pki["certificate_authorities"].(map[string]any)
	hatchCA := cas["hatch"].(map[string]any)

	// Root should be present.
	if _, ok := hatchCA["root"]; !ok {
		t.Error("expected root to be present")
	}

	// Intermediate should NOT be present.
	if _, ok := hatchCA["intermediate"]; ok {
		t.Error("expected no intermediate when paths are empty")
	}
}

func TestTranslate_PKIConfig_NoPKIWhenEmpty(t *testing.T) {
	cfg := config.Config{
		Version: 1,
		Settings: config.Settings{
			HTTPPort:  80,
			HTTPSPort: 443,
		},
		Projects: map[string]config.Project{
			"myapp": {
				Domain:  "myapp.test",
				Path:    "/path/to/myapp",
				Enabled: true,
				Services: map[string]config.Service{
					"web": {Proxy: "http://localhost:3000"},
				},
			},
		},
	}

	result := Translate(cfg, PKIPaths{}, "/test/data/caddy")

	apps := result["apps"].(map[string]any)

	// PKI app should NOT be present.
	if _, ok := apps["pki"]; ok {
		t.Error("expected no pki app when CA paths are empty")
	}

	// TLS issuer should NOT have "ca" field.
	tls := apps["tls"].(map[string]any)
	policies := tls["automation"].(map[string]any)["policies"].([]map[string]any)
	issuers := policies[0]["issuers"].([]map[string]any)
	if _, ok := issuers[0]["ca"]; ok {
		t.Error("expected no 'ca' field in issuer when CA paths are empty")
	}
}

func TestTranslate_GoldenFile(t *testing.T) {
	cfg := fullConfig()
	result := Translate(cfg, PKIPaths{}, "/test/data/caddy")

	got, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("marshaling result: %v", err)
	}

	goldenPath := filepath.Join("testdata", "full.json")

	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(goldenPath, append(got, '\n'), 0o644); err != nil {
			t.Fatalf("updating golden file: %v", err)
		}
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("reading golden file: %v", err)
	}

	if string(got)+"\n" != string(want) {
		t.Errorf("output does not match golden file %s\nRun with UPDATE_GOLDEN=1 to update\n\ngot:\n%s", goldenPath, string(got))
	}
}

func TestTranslate_CommandOnlyServiceSkipped(t *testing.T) {
	cfg := config.Config{
		Version: 1,
		Settings: config.Settings{
			HTTPPort:  80,
			HTTPSPort: 443,
		},
		Projects: map[string]config.Project{
			"myapp": {
				Domain:  "myapp.test",
				Path:    "/path/to/myapp",
				Enabled: true,
				Services: map[string]config.Service{
					"web":    {Proxy: "http://localhost:3000", Command: "npm start"},
					"worker": {Command: "php artisan queue:work"},
				},
			},
		},
	}

	result := Translate(cfg, PKIPaths{}, "/test/data/caddy")

	servers := result["apps"].(map[string]any)["http"].(map[string]any)["servers"].(map[string]any)
	routes := servers["hatch_https"].(map[string]any)["routes"].([]map[string]any)

	// 1 project route + 1 fallback route.
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes (command-only skipped), got %d", len(routes))
	}

	host := routes[0]["match"].([]map[string]any)[0]["host"].([]string)[0]
	if host != "myapp.test" {
		t.Errorf("expected host myapp.test, got %s", host)
	}

	// Domains should only include myapp.test (not a domain for the worker).
	tls := result["apps"].(map[string]any)["tls"].(map[string]any)
	policies := tls["automation"].(map[string]any)["policies"].([]map[string]any)
	if len(policies) != 1 {
		t.Fatalf("expected 1 TLS policy, got %d", len(policies))
	}
	subjects := policies[0]["subjects"].([]string)
	if len(subjects) != 1 || subjects[0] != "myapp.test" {
		t.Errorf("expected [myapp.test], got %v", subjects)
	}
}

func TestTranslate_AllCommandOnlyProject(t *testing.T) {
	cfg := config.Config{
		Version: 1,
		Settings: config.Settings{
			HTTPPort:  80,
			HTTPSPort: 443,
		},
		Projects: map[string]config.Project{
			"myapp": {
				Domain:  "myapp.test",
				Path:    "/path/to/myapp",
				Enabled: true,
				Services: map[string]config.Service{
					"worker": {Command: "php artisan queue:work"},
				},
			},
		},
	}

	result := Translate(cfg, PKIPaths{}, "/test/data/caddy")

	servers := result["apps"].(map[string]any)["http"].(map[string]any)["servers"].(map[string]any)
	routes := servers["hatch_https"].(map[string]any)["routes"].([]map[string]any)

	if len(routes) != 0 {
		t.Errorf("expected 0 routes for command-only project, got %d", len(routes))
	}

	tls := result["apps"].(map[string]any)["tls"].(map[string]any)
	policies := tls["automation"].(map[string]any)["policies"].([]map[string]any)
	if len(policies) != 0 {
		t.Errorf("expected 0 TLS policies, got %d", len(policies))
	}
}

func TestTranslate_FallbackRoute(t *testing.T) {
	cfg := fullConfig()
	result := Translate(cfg, PKIPaths{}, "/test/data/caddy")

	servers := result["apps"].(map[string]any)["http"].(map[string]any)["servers"].(map[string]any)
	routes := servers["hatch_https"].(map[string]any)["routes"].([]map[string]any)

	// Last route should be the fallback with no match key.
	fallback := routes[len(routes)-1]
	if _, hasMatch := fallback["match"]; hasMatch {
		t.Error("fallback route should not have a match key")
	}

	handlers := fallback["handle"].([]map[string]any)
	if len(handlers) != 1 {
		t.Fatalf("expected 1 handler, got %d", len(handlers))
	}

	handler := handlers[0]
	if handler["handler"] != "static_response" {
		t.Errorf("expected static_response handler, got %s", handler["handler"])
	}
	if handler["status_code"] != "404" {
		t.Errorf("expected status 404, got %v", handler["status_code"])
	}

	body := handler["body"].(string)
	if !strings.Contains(body, "acme.test") {
		t.Error("fallback body should contain configured domain acme.test")
	}
	if !strings.Contains(body, "ws.acme.test") {
		t.Error("fallback body should contain configured domain ws.acme.test")
	}
	if !strings.Contains(body, "http://localhost:3000") {
		t.Error("fallback body should contain upstream http://localhost:3000")
	}
	if !strings.Contains(body, "location.host") {
		t.Error("fallback body should use JS location.host for safe display")
	}
	if !strings.Contains(body, "location.pathname") {
		t.Error("fallback body should use JS location.pathname for safe display")
	}

	headers := handler["headers"].(map[string]any)
	ct := headers["Content-Type"].([]string)
	if ct[0] != "text/html; charset=utf-8" {
		t.Errorf("expected Content-Type text/html; charset=utf-8, got %s", ct[0])
	}
}

func TestTranslate_FallbackRoute_EmptyConfig(t *testing.T) {
	cfg := config.Config{
		Version: 1,
		Settings: config.Settings{
			HTTPPort:  80,
			HTTPSPort: 443,
		},
		Projects: map[string]config.Project{},
	}

	result := Translate(cfg, PKIPaths{}, "/test/data/caddy")

	servers := result["apps"].(map[string]any)["http"].(map[string]any)["servers"].(map[string]any)
	routes := servers["hatch_https"].(map[string]any)["routes"].([]map[string]any)

	// No routes at all when config is empty (no fallback either).
	if len(routes) != 0 {
		t.Errorf("expected 0 routes for empty config, got %d", len(routes))
	}
}

func TestTranslate_ErrorResponse(t *testing.T) {
	cfg := config.Config{
		Version: 1,
		Settings: config.Settings{
			HTTPPort:  80,
			HTTPSPort: 443,
		},
		Projects: map[string]config.Project{
			"myapp": {
				Domain:  "myapp.test",
				Path:    "/path/to/myapp",
				Enabled: true,
				Services: map[string]config.Service{
					"web": {Proxy: "http://localhost:3000"},
				},
			},
		},
	}

	result := Translate(cfg, PKIPaths{}, "/test/data/caddy")

	servers := result["apps"].(map[string]any)["http"].(map[string]any)["servers"].(map[string]any)
	routes := servers["hatch_https"].(map[string]any)["routes"].([]map[string]any)

	// First route is the project route.
	subroute := routes[0]["handle"].([]map[string]any)[0]
	subRoutes := subroute["routes"].([]map[string]any)
	proxyHandler := subRoutes[1]["handle"].([]map[string]any)[1]

	if proxyHandler["handler"] != "reverse_proxy" {
		t.Fatalf("expected reverse_proxy handler, got %s", proxyHandler["handler"])
	}

	handleResponse, ok := proxyHandler["handle_response"].([]map[string]any)
	if !ok {
		t.Fatal("expected handle_response on reverse_proxy handler")
	}
	if len(handleResponse) != 2 {
		t.Fatalf("expected 2 handle_response entries (502 + 503), got %d", len(handleResponse))
	}

	// Check each entry has the correct status code.
	for i, wantCode := range []int{502, 503} {
		entry := handleResponse[i]
		matcher := entry["match"].(map[string]any)
		statusCodes := matcher["status_code"].([]int)
		if len(statusCodes) != 1 || statusCodes[0] != wantCode {
			t.Errorf("entry %d: expected status_code [%d], got %v", i, wantCode, statusCodes)
		}

		respRoutes := entry["routes"].([]map[string]any)
		if len(respRoutes) != 1 {
			t.Fatalf("entry %d: expected 1 response route, got %d", i, len(respRoutes))
		}

		respHandler := respRoutes[0]["handle"].([]map[string]any)[0]
		if respHandler["handler"] != "static_response" {
			t.Errorf("entry %d: expected static_response handler, got %s", i, respHandler["handler"])
		}
		wantStatus := fmt.Sprintf("%d", wantCode)
		if respHandler["status_code"] != wantStatus {
			t.Errorf("entry %d: expected status %s, got %v", i, wantStatus, respHandler["status_code"])
		}

		body := respHandler["body"].(string)
		if !strings.Contains(body, "myapp.test") {
			t.Errorf("entry %d: error response body should contain domain myapp.test", i)
		}
		if !strings.Contains(body, "http://localhost:3000") {
			t.Errorf("entry %d: error response body should contain upstream http://localhost:3000", i)
		}
		if !strings.Contains(body, "location.pathname") {
			t.Errorf("entry %d: error response body should use JS location.pathname for safe display", i)
		}
	}
}

func TestTranslate_ServerErrorRoutes(t *testing.T) {
	cfg := fullConfig()
	result := Translate(cfg, PKIPaths{}, "/test/data/caddy")

	servers := result["apps"].(map[string]any)["http"].(map[string]any)["servers"].(map[string]any)
	httpsServer := servers["hatch_https"].(map[string]any)

	errors, ok := httpsServer["errors"].(map[string]any)
	if !ok {
		t.Fatal("expected errors config on HTTPS server")
	}

	routes := errors["routes"].([]map[string]any)
	if len(routes) != 1 {
		t.Fatalf("expected 1 error route, got %d", len(routes))
	}

	handler := routes[0]["handle"].([]map[string]any)[0]
	if handler["handler"] != "static_response" {
		t.Errorf("expected static_response handler, got %s", handler["handler"])
	}
	if handler["status_code"] != "502" {
		t.Errorf("expected status 502, got %v", handler["status_code"])
	}

	body := handler["body"].(string)
	if !strings.Contains(body, "acme.test") {
		t.Error("server error page should contain configured domain acme.test")
	}
	if !strings.Contains(body, "http://localhost:3000") {
		t.Error("server error page should contain upstream http://localhost:3000")
	}
	if !strings.Contains(body, "location.host") {
		t.Error("server error page should use JS location.host for safe display")
	}
}

func TestTranslate_ServerErrorRoutes_EmptyConfig(t *testing.T) {
	cfg := config.Config{
		Version: 1,
		Settings: config.Settings{
			HTTPPort:  80,
			HTTPSPort: 443,
		},
		Projects: map[string]config.Project{},
	}

	result := Translate(cfg, PKIPaths{}, "/test/data/caddy")

	servers := result["apps"].(map[string]any)["http"].(map[string]any)["servers"].(map[string]any)
	httpsServer := servers["hatch_https"].(map[string]any)

	if _, ok := httpsServer["errors"]; ok {
		t.Error("expected no errors config on HTTPS server when config is empty")
	}
}

func TestTranslate_TLSWildcard_NoTLD(t *testing.T) {
	cfg := config.Config{
		Version: 1,
		Settings: config.Settings{
			HTTPPort:  80,
			HTTPSPort: 443,
			// TLD not set.
		},
		Projects: map[string]config.Project{
			"myapp": {
				Domain:  "myapp.test",
				Path:    "/path/to/myapp",
				Enabled: true,
				Services: map[string]config.Service{
					"web": {Proxy: "http://localhost:3000"},
				},
			},
		},
	}

	result := Translate(cfg, PKIPaths{}, "/test/data/caddy")

	tls := result["apps"].(map[string]any)["tls"].(map[string]any)
	policies := tls["automation"].(map[string]any)["policies"].([]map[string]any)
	subjects := policies[0]["subjects"].([]string)

	// No wildcard when TLD is empty.
	for _, s := range subjects {
		if strings.HasPrefix(s, "*.") {
			t.Errorf("expected no wildcard subject when TLD is empty, got %s", s)
		}
	}
}

func TestExtractDialAddress(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"http://localhost:3000", "localhost:3000"},
		{"http://localhost:8000", "localhost:8000"},
		{"http://127.0.0.1:9000", "127.0.0.1:9000"},
		{"https://localhost:8443", "localhost:8443"},
		{"http://localhost", "localhost:80"},
		{"https://localhost", "localhost:443"},
		{"http://0.0.0.0:5000", "0.0.0.0:5000"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractDialAddress(tt.input)
			if got != tt.want {
				t.Errorf("extractDialAddress(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
