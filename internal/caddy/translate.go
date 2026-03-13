package caddy

import (
	"fmt"
	"html"
	"net/url"
	"sort"
	"strings"

	"github.com/httphatch/hatch/internal/config"
)

// PKIPaths holds the file paths for root and intermediate CA certificates
// and keys used to configure Caddy's PKI app.
type PKIPaths struct {
	RootCert         string
	RootKey          string
	IntermediateCert string
	IntermediateKey  string
}

// Translate converts a Hatch config into a full Caddy JSON configuration.
// It skips disabled projects and returns a map suitable for JSON marshaling.
// When pki.RootCert is non-empty, a PKI app is added so Caddy uses the
// provided CA for issuing leaf certificates. dataDir controls where Caddy
// stores certificates and PKI data. tunnelDomains maps "project/service"
// keys to external hostnames resolved from Cloudflare tunnel ingress rules.
func Translate(cfg config.Config, pki PKIPaths, dataDir string, tunnelDomains map[string][]string, tunnelProxies map[string]string) map[string]any {
	infos := collectRouteInfos(cfg, tunnelDomains, tunnelProxies)
	httpsRoutes := buildRoutes(infos)
	httpRedirectRoutes := buildHTTPRedirectRoutes(cfg, tunnelDomains)
	tlsConfig := buildTLSConfig(cfg, pki.RootCert, tunnelDomains)

	httpsPort := fmt.Sprintf(":%d", cfg.Settings.HTTPSPort)
	httpPort := fmt.Sprintf(":%d", cfg.Settings.HTTPPort)

	httpsServer := map[string]any{
		"listen":                 []string{httpsPort},
		"routes":                 httpsRoutes,
		"tls_connection_policies": []map[string]any{{}},
		"automatic_https": map[string]any{
			"disable_redirects": true,
		},
		"logs": map[string]any{
			"default_logger_name": "access",
		},
	}

	if len(infos) > 0 {
		httpsServer["errors"] = buildServerErrorRoutes(infos)
	}

	apps := map[string]any{
		"http": map[string]any{
			"servers": map[string]any{
				"hatch_https": httpsServer,
				"hatch_http": map[string]any{
					"listen": []string{httpPort},
					"routes": httpRedirectRoutes,
				},
			},
		},
		"tls": tlsConfig,
	}

	if pki.RootCert != "" {
		apps["pki"] = buildPKIConfig(pki)
	}

	return map[string]any{
		"admin": map[string]any{
			"listen": DefaultAdminAddr,
		},
		"apps":    apps,
		"logging": buildLoggingConfig(),
		"storage": map[string]any{
			"module": "file_system",
			"root":   dataDir,
		},
	}
}

// routeInfo holds metadata for sorting routes by specificity.
type routeInfo struct {
	domain        string
	service       config.Service
	proxyOverride string // rewrite proxy address for tunnel domain routes
}

// collectRouteInfos gathers all enabled proxy services from the config.
// When tunnelDomains is non-nil, additional routeInfo entries are created
// for each external hostname attached to a service's named tunnel.
// When tunnelProxies is non-nil, tunnel domain routes use the rewrite proxy
// address instead of the original upstream.
func collectRouteInfos(cfg config.Config, tunnelDomains map[string][]string, tunnelProxies map[string]string) []routeInfo {
	var infos []routeInfo

	for projName, proj := range cfg.Projects {
		if !proj.Enabled {
			continue
		}
		for svcName, svc := range proj.Services {
			if svc.Proxy == "" {
				continue
			}
			domain := proj.Domain
			if svc.Subdomain != "" {
				domain = svc.Subdomain + "." + proj.Domain
			}
			infos = append(infos, routeInfo{
				domain:  domain,
				service: svc,
			})

			key := projName + "/" + svcName
			proxyAddr := tunnelProxies[key]
			for _, td := range tunnelDomains[key] {
				infos = append(infos, routeInfo{
					domain:        td,
					service:       svc,
					proxyOverride: proxyAddr,
				})
			}
		}
	}

	return infos
}

// buildRoutes builds HTTPS routes for all enabled projects, sorted by specificity.
func buildRoutes(infos []routeInfo) []map[string]any {
	sorted := make([]routeInfo, len(infos))
	copy(sorted, infos)

	sort.SliceStable(sorted, func(i, j int) bool {
		ti := routeTier(sorted[i])
		tj := routeTier(sorted[j])
		if ti != tj {
			return ti < tj
		}
		// Within same tier, longer paths first.
		if len(sorted[i].service.Route) != len(sorted[j].service.Route) {
			return len(sorted[i].service.Route) > len(sorted[j].service.Route)
		}
		// Alphabetical path tiebreaker.
		if sorted[i].service.Route != sorted[j].service.Route {
			return sorted[i].service.Route < sorted[j].service.Route
		}
		// Alphabetical domain tiebreaker.
		return sorted[i].domain < sorted[j].domain
	})

	routes := make([]map[string]any, 0, len(sorted)+1)
	for _, info := range sorted {
		proxyURL := info.service.Proxy
		if info.proxyOverride != "" {
			proxyURL = info.proxyOverride
		}
		routes = append(routes, buildRoute(info.domain, info.service, proxyURL))
	}

	if len(routes) > 0 {
		routes = append(routes, buildFallbackRoute(infos))
	}

	return routes
}

// routeTier returns a sorting priority: 0 = subdomain, 1 = path, 2 = catch-all.
func routeTier(info routeInfo) int {
	if info.service.Subdomain != "" {
		return 0
	}
	if info.service.Route != "" {
		return 1
	}
	return 2
}

// corsHeaders returns the CORS response headers added to every proxied response.
// The Origin is reflected from the request so that credentialed cross-origin
// requests work between .test subdomains during local development.
func corsHeaders() map[string][]string {
	return map[string][]string{
		"Access-Control-Allow-Origin":          {"{http.request.header.Origin}"},
		"Access-Control-Allow-Methods":         {"GET, POST, PUT, PATCH, DELETE, OPTIONS"},
		"Access-Control-Allow-Headers":         {"Authorization, Content-Type, X-Requested-With, Accept, Origin"},
		"Access-Control-Allow-Credentials":     {"true"},
		"Access-Control-Allow-Private-Network": {"true"},
		"Vary":                                 {"Origin"},
	}
}

// buildRoute builds a single HTTPS route with host matcher, optional path matcher,
// and a subroute that handles CORS preflight and adds CORS headers to responses.
// proxyURL is the upstream address to proxy to (may differ from svc.Proxy when
// a rewrite proxy is in use for tunnel domain routes).
func buildRoute(domain string, svc config.Service, proxyURL string) map[string]any {
	match := map[string]any{
		"host": []string{domain},
	}
	if svc.Route != "" {
		match["path"] = []string{svc.Route}
	}

	proxyHandler := buildReverseProxyHandler(proxyURL, svc.WebSocket, domain, svc.Proxy)

	cors := corsHeaders()

	// Preflight headers include Max-Age to let browsers cache the result.
	preflightHeaders := make(map[string][]string, len(cors)+1)
	for k, v := range cors {
		preflightHeaders[k] = v
	}
	preflightHeaders["Access-Control-Max-Age"] = []string{"3600"}

	subroute := map[string]any{
		"handler": "subroute",
		"routes": []map[string]any{
			{
				"match": []map[string]any{
					{"method": []string{"OPTIONS"}},
				},
				"handle": []map[string]any{
					{
						"handler":     "static_response",
						"status_code": "204",
						"headers":     preflightHeaders,
					},
				},
			},
			{
				"handle": []map[string]any{
					{
						"handler": "headers",
						"response": map[string]any{
							"set": cors,
						},
					},
					proxyHandler,
				},
			},
		},
	}

	return map[string]any{
		"match":    []map[string]any{match},
		"handle":   []map[string]any{subroute},
		"terminal": true,
	}
}

// buildReverseProxyHandler builds a reverse_proxy handler with the dial address
// extracted from proxyURL. If websocket is true, it adds flush_interval: -1 and
// Connection/Upgrade header forwarding. The domain is used for error response pages.
// displayUpstream is shown on error pages and may differ from proxyURL when a
// rewrite proxy sits between Caddy and the real upstream.
func buildReverseProxyHandler(proxyURL string, websocket bool, domain, displayUpstream string) map[string]any {
	handler := map[string]any{
		"handler":         "reverse_proxy",
		"upstreams":       []map[string]any{{"dial": extractDialAddress(proxyURL)}},
		"handle_response": buildErrorResponse(domain, displayUpstream),
	}

	requestHeaders := map[string]any{
		"X-Forwarded-Proto": []string{"https"},
	}

	if websocket {
		handler["flush_interval"] = -1
		requestHeaders["Connection"] = []string{"{http.request.header.Connection}"}
		requestHeaders["Upgrade"] = []string{"{http.request.header.Upgrade}"}
	}

	handler["headers"] = map[string]any{
		"request": map[string]any{
			"set": requestHeaders,
		},
	}

	return handler
}

// buildFallbackRoute builds a catch-all route with no host matcher that returns
// a branded 404 page listing all configured domains. Appended as the last HTTPS
// route so it only matches requests that fell through all project routes.
func buildFallbackRoute(infos []routeInfo) map[string]any {
	sorted := make([]routeInfo, len(infos))
	copy(sorted, infos)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].domain < sorted[j].domain
	})

	var rows []string
	for _, info := range sorted {
		safeDomain := html.EscapeString(info.domain)
		safeProxy := html.EscapeString(info.service.Proxy)
		rows = append(rows, fmt.Sprintf(
			`<tr><td>%s</td><td>%s</td></tr>`,
			safeDomain, safeProxy,
		))
	}

	tableHTML := strings.Join(rows, "\n              ")

	body := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>404 - Not Found</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #faf5ef; color: #2d2d2d; padding: 40px 20px; }
    .container { max-width: 640px; margin: 0 auto; }
    h1 { font-size: 24px; color: #5f8f8f; margin-bottom: 8px; }
    .subtitle { font-size: 14px; color: #666; margin-bottom: 32px; }
    .card { background: #fff; border: 1px solid #e0d8cf; border-radius: 8px; padding: 24px; margin-bottom: 24px; }
    .card h2 { font-size: 14px; text-transform: uppercase; letter-spacing: 0.5px; color: #999; margin-bottom: 12px; }
    .detail { font-size: 15px; margin-bottom: 4px; }
    .detail strong { color: #5f8f8f; }
    table { width: 100%%; border-collapse: collapse; margin-top: 8px; }
    th, td { text-align: left; padding: 8px 12px; font-size: 14px; }
    th { color: #999; font-weight: 500; border-bottom: 1px solid #e0d8cf; }
    td { border-bottom: 1px solid #f0ebe4; }
    ul { list-style: none; padding: 0; }
    ul li { font-size: 14px; padding: 4px 0; color: #666; }
    ul li::before { content: "\2022"; color: #5f8f8f; display: inline-block; width: 16px; }
  </style>
</head>
<body>
  <div class="container">
    <h1>Hatch</h1>
    <p class="subtitle">No route matched this request.</p>
    <div class="card">
      <h2>Request</h2>
      <p class="detail"><strong>Host:</strong> <span id="h"></span></p>
      <p class="detail"><strong>URL:</strong> <span id="u"></span></p>
    </div>
    <div class="card">
      <h2>Configured domains</h2>
      <table>
        <tr><th>Domain</th><th>Upstream</th></tr>
        %s
      </table>
    </div>
    <div class="card">
      <h2>Common causes</h2>
      <ul>
        <li>The domain is not configured in ~/.hatch/config.yml</li>
        <li>The project is disabled</li>
        <li>The Host header does not match any configured domain</li>
      </ul>
    </div>
  </div>
  <script>document.getElementById("h").textContent=location.host;document.getElementById("u").textContent=location.pathname+location.search;</script>
</body>
</html>`, tableHTML)

	return map[string]any{
		"handle": []map[string]any{
			{
				"handler":     "static_response",
				"status_code": "404",
				"headers": map[string]any{
					"Content-Type": []string{"text/html; charset=utf-8"},
				},
				"body": body,
			},
		},
		"terminal": true,
	}
}

// buildErrorResponse returns handle_response entries that intercept 502 and
// 503 responses from the upstream and replace them with branded error pages
// showing the failing domain and upstream.
func buildErrorResponse(domain, upstream string) []map[string]any {
	safeDomain := html.EscapeString(domain)
	safeUpstream := html.EscapeString(upstream)

	errorPage := func(statusCode int, title, subtitle string) map[string]any {
		body := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>%d - %s</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #faf5ef; color: #2d2d2d; padding: 40px 20px; }
    .container { max-width: 640px; margin: 0 auto; }
    h1 { font-size: 24px; color: #5f8f8f; margin-bottom: 8px; }
    .subtitle { font-size: 14px; color: #666; margin-bottom: 32px; }
    .card { background: #fff; border: 1px solid #e0d8cf; border-radius: 8px; padding: 24px; margin-bottom: 24px; }
    .card h2 { font-size: 14px; text-transform: uppercase; letter-spacing: 0.5px; color: #999; margin-bottom: 12px; }
    .detail { font-size: 15px; margin-bottom: 4px; }
    .detail strong { color: #5f8f8f; }
    ul { list-style: none; padding: 0; }
    ul li { font-size: 14px; padding: 4px 0; color: #666; }
    ul li::before { content: "\2022"; color: #5f8f8f; display: inline-block; width: 16px; }
  </style>
</head>
<body>
  <div class="container">
    <h1>Hatch</h1>
    <p class="subtitle">%s</p>
    <div class="card">
      <h2>Request</h2>
      <p class="detail"><strong>Domain:</strong> %s</p>
      <p class="detail"><strong>URL:</strong> <span id="u"></span></p>
      <p class="detail"><strong>Upstream:</strong> %s</p>
    </div>
    <div class="card">
      <h2>What to check</h2>
      <ul>
        <li>Verify the upstream service at %s is running</li>
        <li>Check the service logs for errors</li>
        <li>Confirm the port number in your Hatch configuration</li>
      </ul>
    </div>
  </div>
  <script>document.getElementById("u").textContent=location.pathname+location.search;</script>
</body>
</html>`, statusCode, title, subtitle, safeDomain, safeUpstream, safeUpstream)

		return map[string]any{
			"match": map[string]any{
				"status_code": []int{statusCode},
			},
			"routes": []map[string]any{
				{
					"handle": []map[string]any{
						{
							"handler":     "static_response",
							"status_code": fmt.Sprintf("%d", statusCode),
							"headers": map[string]any{
								"Content-Type": []string{"text/html; charset=utf-8"},
							},
							"body": body,
						},
					},
				},
			},
		}
	}

	return []map[string]any{
		errorPage(502, "Bad Gateway", "The upstream service is not reachable."),
		errorPage(503, "Service Unavailable", "The upstream service is temporarily unavailable."),
	}
}

// buildServerErrorRoutes returns the server-level error handling config that
// catches errors raised by handlers (e.g. upstream dial failures). This is
// separate from handle_response which only fires when the upstream actually
// responds. The error page includes a table of all configured domains so the
// user can look up their upstream.
func buildServerErrorRoutes(infos []routeInfo) map[string]any {
	sorted := make([]routeInfo, len(infos))
	copy(sorted, infos)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].domain < sorted[j].domain
	})

	var rows []string
	for _, info := range sorted {
		safeDomain := html.EscapeString(info.domain)
		safeProxy := html.EscapeString(info.service.Proxy)
		rows = append(rows, fmt.Sprintf(
			`<tr><td>%s</td><td>%s</td></tr>`,
			safeDomain, safeProxy,
		))
	}

	tableHTML := strings.Join(rows, "\n              ")

	body := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Error - Upstream Unreachable</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #faf5ef; color: #2d2d2d; padding: 40px 20px; }
    .container { max-width: 640px; margin: 0 auto; }
    h1 { font-size: 24px; color: #5f8f8f; margin-bottom: 8px; }
    .subtitle { font-size: 14px; color: #666; margin-bottom: 32px; }
    .card { background: #fff; border: 1px solid #e0d8cf; border-radius: 8px; padding: 24px; margin-bottom: 24px; }
    .card h2 { font-size: 14px; text-transform: uppercase; letter-spacing: 0.5px; color: #999; margin-bottom: 12px; }
    .detail { font-size: 15px; margin-bottom: 4px; }
    .detail strong { color: #5f8f8f; }
    table { width: 100%%; border-collapse: collapse; margin-top: 8px; }
    th, td { text-align: left; padding: 8px 12px; font-size: 14px; }
    th { color: #999; font-weight: 500; border-bottom: 1px solid #e0d8cf; }
    td { border-bottom: 1px solid #f0ebe4; }
    ul { list-style: none; padding: 0; }
    ul li { font-size: 14px; padding: 4px 0; color: #666; }
    ul li::before { content: "\2022"; color: #5f8f8f; display: inline-block; width: 16px; }
  </style>
</head>
<body>
  <div class="container">
    <h1>Hatch</h1>
    <p class="subtitle">The upstream service is not reachable.</p>
    <div class="card">
      <h2>Request</h2>
      <p class="detail"><strong>Host:</strong> <span id="h"></span></p>
      <p class="detail"><strong>URL:</strong> <span id="u"></span></p>
    </div>
    <div class="card">
      <h2>Configured domains</h2>
      <table>
        <tr><th>Domain</th><th>Upstream</th></tr>
        %s
      </table>
    </div>
    <div class="card">
      <h2>What to check</h2>
      <ul>
        <li>Verify the upstream service is running on the configured port</li>
        <li>Check the service logs for errors</li>
        <li>Confirm the port number in ~/.hatch/config.yml</li>
      </ul>
    </div>
  </div>
  <script>document.getElementById("h").textContent=location.host;document.getElementById("u").textContent=location.pathname+location.search;</script>
</body>
</html>`, tableHTML)

	return map[string]any{
		"routes": []map[string]any{
			{
				"handle": []map[string]any{
					{
						"handler":     "static_response",
						"status_code": "502",
						"headers": map[string]any{
							"Content-Type": []string{"text/html; charset=utf-8"},
						},
						"body": body,
					},
				},
			},
		},
	}
}

// buildHTTPRedirectRoutes builds HTTP→HTTPS redirect routes using a static_response
// handler with a 302 redirect. A catch-all redirect (no host matcher) is appended
// after the domain-specific routes so that unconfigured domains also redirect to
// HTTPS, where the on-demand TLS and fallback 404 route handle them.
func buildHTTPRedirectRoutes(cfg config.Config, tunnelDomains map[string][]string) []map[string]any {
	redirectHandler := []map[string]any{
		{
			"handler":     "static_response",
			"status_code": "302",
			"headers": map[string]any{
				"Location": []string{"https://{http.request.host}{http.request.uri}"},
			},
		},
	}

	domains := collectDomains(cfg, tunnelDomains)
	if len(domains) == 0 {
		return []map[string]any{
			{
				"handle": redirectHandler,
			},
		}
	}

	return []map[string]any{
		{
			"match": []map[string]any{
				{"host": domains},
			},
			"handle":   redirectHandler,
			"terminal": true,
		},
		{
			"handle": redirectHandler,
		},
	}
}

// buildTLSConfig builds the TLS automation config with internal issuer.
// When rootCACert is non-empty, the issuer references the "hatch" CA.
// A catch-all on-demand policy is appended so that requests to unconfigured
// domains still get a valid certificate during the TLS handshake, allowing
// the HTTP fallback route to serve the 404 debug page.
func buildTLSConfig(cfg config.Config, rootCACert string, tunnelDomains map[string][]string) map[string]any {
	domains := collectDomains(cfg, tunnelDomains)

	issuer := map[string]any{"module": "internal"}
	if rootCACert != "" {
		issuer["ca"] = "hatch"
	}

	policies := make([]map[string]any, 0)
	if len(domains) > 0 {
		policies = append(policies, map[string]any{
			"subjects": domains,
			"issuers":  []map[string]any{issuer},
		})
	}

	// On-demand catch-all policy: issues certs at TLS handshake time for
	// any domain not matched above. This lets unconfigured .test domains
	// complete the TLS handshake and reach the fallback 404 route.
	policies = append(policies, map[string]any{
		"on_demand":  true,
		"issuers":    []map[string]any{issuer},
	})

	return map[string]any{
		"automation": map[string]any{
			"policies": policies,
		},
	}
}

// buildPKIConfig returns the Caddy PKI app configuration that registers
// a "hatch" certificate authority backed by the given root and optional
// intermediate CA files.
func buildPKIConfig(pki PKIPaths) map[string]any {
	ca := map[string]any{
		"name": "Hatch Local CA",
		"root": map[string]any{
			"certificate": pki.RootCert,
			"private_key": pki.RootKey,
		},
	}
	if pki.IntermediateCert != "" {
		ca["intermediate"] = map[string]any{
			"certificate": pki.IntermediateCert,
			"private_key": pki.IntermediateKey,
		}
	}
	return map[string]any{
		"certificate_authorities": map[string]any{"hatch": ca},
	}
}

// collectDomains returns all unique domains across enabled projects, sorted.
// Each service's full domain is listed explicitly so that Caddy's automatic
// HTTPS exact-match check recognises them against the TLS automation policy.
// Tunnel domains are included so certificates are pre-generated.
func collectDomains(cfg config.Config, tunnelDomains map[string][]string) []string {
	domainSet := make(map[string]bool)

	for projName, proj := range cfg.Projects {
		if !proj.Enabled {
			continue
		}
		for svcName, svc := range proj.Services {
			if svc.Proxy == "" {
				continue
			}
			domainSet[proj.Domain] = true
			if svc.Subdomain != "" {
				domainSet[svc.Subdomain+"."+proj.Domain] = true
			}

			key := projName + "/" + svcName
			for _, td := range tunnelDomains[key] {
				domainSet[td] = true
			}
		}
	}

	domains := make([]string, 0, len(domainSet))
	for d := range domainSet {
		domains = append(domains, d)
	}
	sort.Strings(domains)
	return domains
}

// buildLoggingConfig returns the top-level Caddy logging configuration.
// Access logs are sent to stderr (which the daemon captures via pipe into
// the rotated log file). Noisy Caddy internals are silenced to WARN.
func buildLoggingConfig() map[string]any {
	return map[string]any{
		"logs": map[string]any{
			"default": map[string]any{
				"level":  "WARN",
				"writer": map[string]any{"output": "stderr"},
			},
			"access": map[string]any{
				"writer":  map[string]any{"output": "stderr"},
				"include": []string{"http.log.access"},
			},
		},
	}
}

// extractDialAddress parses a proxy URL and returns the host:port dial address.
// For URLs without an explicit port, it defaults to :80 for http and :443 for https.
func extractDialAddress(proxyURL string) string {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return proxyURL
	}
	host := u.Host
	if u.Port() == "" {
		switch u.Scheme {
		case "https":
			host += ":443"
		default:
			host += ":80"
		}
	}
	return host
}
