package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	privateIPBlocks         []*net.IPNet
	AllowLoopbackForTesting = false
)

func init() {
	cidrs := []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918 Private
		"172.16.0.0/12",  // RFC1918 Private
		"192.168.0.0/16", // RFC1918 Private
		"169.254.0.0/16", // Link-local
		"0.0.0.0/8",      // Current network
		"::1/128",        // IPv6 loopback
		"fc00::/7",       // Unique Local Address
		"fe80::/10",      // IPv6 link-local
	}

	for _, cidr := range cidrs {
		_, block, err := net.ParseCIDR(cidr)
		if err == nil {
			privateIPBlocks = append(privateIPBlocks, block)
		}
	}
}

// IsPrivateIP checks whether a net.IP is loopback, unspecified, link-local, or private.
func IsPrivateIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() {
		return !AllowLoopbackForTesting
	}
	if ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			if AllowLoopbackForTesting && (block.String() == "127.0.0.0/8" || block.String() == "::1/128") {
				continue
			}
			return true
		}
	}
	return false
}

// ValidateSafeURL checks if a target URL string uses HTTP/HTTPS, contains no credentials, and resolves only to public IPs.
func ValidateSafeURL(targetURL string) error {
	if strings.TrimSpace(targetURL) == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	u, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("unsupported URL scheme '%s'; only http and https are allowed", u.Scheme)
	}

	if u.User != nil {
		return fmt.Errorf("embedded user credentials in URL are not allowed")
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("URL missing host specification")
	}

	if strings.EqualFold(host, "localhost") {
		if AllowLoopbackForTesting {
			return nil
		}
		return fmt.Errorf("destination host 'localhost' is blocked for security (SSRF protection)")
	}

	// Check if host is already an IP address
	ip := net.ParseIP(host)
	if ip != nil {
		if IsPrivateIP(ip) {
			return fmt.Errorf("destination IP %s is a private or loopback address (SSRF protection)", ip.String())
		}
		return nil
	}

	// Perform DNS resolution for hostname
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	addrs, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return fmt.Errorf("failed to resolve host '%s': %w", host, err)
	}

	if len(addrs) == 0 {
		return fmt.Errorf("host '%s' resolved to no IP addresses", host)
	}

	for _, resolvedIP := range addrs {
		if IsPrivateIP(resolvedIP) {
			return fmt.Errorf("destination host '%s' resolved to private IP %s (SSRF protection)", host, resolvedIP.String())
		}
	}

	return nil
}

// NewSafeHTTPClient creates an http.Client with strict dial timeouts and redirect validation.
func NewSafeHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	dialer := &net.Dialer{
		Timeout:   3 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   3 * time.Second,
		ResponseHeaderTimeout: timeout,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects (max 5)")
			}
			if err := ValidateSafeURL(req.URL.String()); err != nil {
				return fmt.Errorf("redirect destination blocked by SSRF protection: %w", err)
			}
			return nil
		},
	}

	return client
}

// RedactURL removes sensitive userinfo or query string credentials from a URL string for safe logging.
func RedactURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "[malformed-url]"
	}

	if u.User != nil {
		u.User = url.User("redacted")
	}

	q := u.Query()
	for key := range q {
		lowerKey := strings.ToLower(key)
		if strings.Contains(lowerKey, "token") ||
			strings.Contains(lowerKey, "key") ||
			strings.Contains(lowerKey, "secret") ||
			strings.Contains(lowerKey, "auth") ||
			strings.Contains(lowerKey, "pass") {
			q.Set(key, "[REDACTED]")
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// SanitizeErrorMessage redacts credentials, routing keys, tokens, and authorization headers from error strings.
func SanitizeErrorMessage(msg string) string {
	if msg == "" {
		return ""
	}

	// Redact bearer tokens
	reBearer := regexp.MustCompile(`(?i)(bearer\s+)[a-zA-Z0-9_\-\.=]+`)
	msg = reBearer.ReplaceAllString(msg, "$1[REDACTED]")

	// Redact routing keys & tokens in key=value or json format
	reSecret := regexp.MustCompile(`(?i)("(?:routing_key|integration_key|service_key|token|secret|password|auth_header)":\s*")[^"]+(")`)
	msg = reSecret.ReplaceAllString(msg, "$1[REDACTED]$2")

	// Redact basic auth in URLs http://user:pass@host
	reURLCreds := regexp.MustCompile(`(https?://)[^:@\s]+:[^@\s]+@`)
	msg = reURLCreds.ReplaceAllString(msg, "$1[REDACTED]:[REDACTED]@")

	return msg
}
