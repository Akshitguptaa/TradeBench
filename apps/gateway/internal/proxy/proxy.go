package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func NewProxy(target string) http.Handler {
	u, err := url.Parse(target)
	if err != nil {
		log.Fatalf("proxy: invalid target URL %q: %v", target, err)
	}

	rp := httputil.NewSingleHostReverseProxy(u)

	// override Host header so the backend sees its own hostname, not the gateway's
	origDirector := rp.Director
	rp.Director = func(req *http.Request) {
		origDirector(req)
		req.Host = u.Host
	}

	// return a clean JSON error instead of the default stdlib HTML page
	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy error %s %s → %s: %v", r.Method, r.URL.Path, target, err)
		http.Error(w, `{"error":"service unavailable"}`, http.StatusBadGateway)
	}

	return rp
}
