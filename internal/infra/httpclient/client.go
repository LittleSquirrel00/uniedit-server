package httpclient

import (
	"net"
	"net/http"

	"github.com/uniedit/server/internal/infra/config"
)

// New creates a new HTTP client with the given configuration.
func New(cfg config.HTTPClientConfig) *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   cfg.DialTimeout,
			KeepAlive: cfg.KeepAlive,
		}).DialContext,
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
		MaxConnsPerHost:     cfg.MaxConnsPerHost,
		IdleConnTimeout:     cfg.IdleConnTimeout,
		TLSHandshakeTimeout: cfg.TLSHandshakeTimeout,
		ForceAttemptHTTP2:   true,
		DisableCompression:  false,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   cfg.ResponseTimeout,
	}
}
