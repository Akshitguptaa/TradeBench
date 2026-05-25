// Package proxy provides lightweight reverse-proxy handlers for routing
// gateway requests to downstream TradeBench services.
package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// NewProxy returns an http.Handler that reverse-proxies requests to the given
// target URL. The Host header is rewritten to the target so downstream services
// see the correct host.
func NewProxy(target string) http.Handler {
	u, err := url.Parse(target)
	if err != nil {
		log.Fatalf("proxy: invalid target URL %q: %v", target, err)
	}

	rp := httputil.NewSingleHostReverseProxy(u)

	// Override the Director to set the scheme, host, and strip nothing.
	origDirector := rp.Director
	rp.Director = func(req *http.Request) {
		origDirector(req)
		req.Host = u.Host
	}

	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy error %s %s → %s: %v", r.Method, r.URL.Path, target, err)
		http.Error(w, `{"error":"service unavailable"}`, http.StatusBadGateway)
	}

	return rp
}
