// Package executor provides HTTP client and request execution for the Antigravity API.
package executor

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
)

// NewHTTPClient creates an HTTP client with optional proxy configuration.
func NewHTTPClient(proxyURL string, timeout time.Duration) *http.Client {
	client := &http.Client{}
	if timeout > 0 {
		client.Timeout = timeout
	}

	if proxyURL != "" {
		transport := buildProxyTransport(proxyURL)
		if transport != nil {
			client.Transport = transport
		}
	}

	return client
}

// buildProxyTransport creates an HTTP transport configured for the given proxy URL.
// It supports SOCKS5, HTTP, and HTTPS proxy protocols.
func buildProxyTransport(proxyURL string) *http.Transport {
	if proxyURL == "" {
		return nil
	}

	parsedURL, err := url.Parse(proxyURL)
	if err != nil {
		log.Errorf("parse proxy URL failed: %v", err)
		return nil
	}

	var transport *http.Transport

	switch parsedURL.Scheme {
	case "socks5":
		var proxyAuth *proxy.Auth
		if parsedURL.User != nil {
			username := parsedURL.User.Username()
			password, _ := parsedURL.User.Password()
			proxyAuth = &proxy.Auth{User: username, Password: password}
		}
		dialer, err := proxy.SOCKS5("tcp", parsedURL.Host, proxyAuth, proxy.Direct)
		if err != nil {
			log.Errorf("create SOCKS5 dialer failed: %v", err)
			return nil
		}
		transport = &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			},
		}
	case "http", "https":
		transport = &http.Transport{Proxy: http.ProxyURL(parsedURL)}
	default:
		log.Errorf("unsupported proxy scheme: %s", parsedURL.Scheme)
		return nil
	}

	return transport
}

// resolveHost extracts the host from a URL string.
func resolveHost(baseURL string) string {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	if parsed.Host != "" {
		return parsed.Host
	}
	return strings.TrimPrefix(strings.TrimPrefix(baseURL, "https://"), "http://")
}