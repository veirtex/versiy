package security

import (
	"errors"
	"fmt"
	"html"
	"net"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

const (
	// MaxURLLength defines maximum allowed URL length (2KB)
	MaxURLLength = 2048
)

// Security errors
var (
	ErrURLTooLong        = errors.New("url exceeds maximum length of 2048 characters")
	ErrInvalidURLScheme  = errors.New("only http and https protocols are allowed")
	ErrInvalidURLFormat  = errors.New("invalid url format")
	ErrXSSDetected       = errors.New("url contains potentially malicious content (XSS)")
	ErrSSRFDetected      = errors.New("url points to internal network address (SSRF)")
	ErrDoubleProtocol    = errors.New("url contains double protocol sequence")
	ErrURLEncodedPayload = errors.New("url contains encoded characters which are not allowed")
	ErrSQLDetected       = errors.New("url contains SQL-like patterns")
	ErrIDNDetected       = errors.New("internationalized domain names not allowed for security")
	ErrInternalPath      = errors.New("redirect to internal application paths not allowed")
)

// Patterns for security validation
var (
	// Double protocol pattern (e.g., https://https://)
	doubleProtocolPattern = regexp.MustCompile(`:[a-zA-Z]+://.*[a-zA-Z]+://`)

	// URL encoding pattern
	urlEncodingPattern = regexp.MustCompile(`%[0-9a-fA-F]{2}`)

	// SQL injection pattern (case insensitive)
	sqlPattern = regexp.MustCompile(`(?i)['"]?\s*(OR|AND|UNION|SELECT|DROP|INSERT|UPDATE|DELETE|WHERE|EXEC|EXECUTE)`)

	// XSS patterns
	xssPatterns = []string{
		"<script",
		"</script>",
		"javascript:",
		"onerror=",
		"onload=",
		"onclick=",
		"onmouseover=",
		"onfocus=",
		"onblur=",
		"onchange=",
		"onsubmit=",
		"<img",
		"<iframe",
		"<object",
		"<embed",
		"<link",
		"<meta",
		"fromCharCode",
		"eval(",
		"alert(",
		"document.cookie",
		"document.write",
		"innerHTML",
		"vbscript:",
		"expression(",
	}

	// Private IP ranges for SSRF detection
	privateIPBlocks = []string{
		"10.0.0.0/8",      // Private Class A
		"172.16.0.0/12",   // Private Class B
		"192.168.0.0/16",  // Private Class C
		"127.0.0.0/8",     // Loopback
		"169.254.0.0/16",  // Link-local (cloud metadata)
		"100.64.0.0/10",   // CGNAT
		"192.0.0.0/24",    // IETF Protocol Assignments
		"192.0.2.0/24",    // TEST-NET-1
		"198.51.100.0/24", // TEST-NET-2
		"203.0.113.0/24",  // TEST-NET-3
		"::1/128",         // IPv6 loopback
		"fc00::/7",        // IPv6 private
		"fe80::/10",       // IPv6 link-local
	}

	// Blocked hostnames
	blockedHosts = []string{
		"localhost",
		"localhost.localdomain",
		"ip6-localhost",
		"ip6-loopback",
	}

	// Cloud metadata endpoints
	metadataPatterns = []string{
		"169.254.169.254",
		"metadata.google.internal",
		"metadata.azure.internal",
		"169.254.169.254/latest",
		"169.254.169.254/metadata",
	}

	// Internal application paths
	internalPaths = []string{
		"/admin",
		"/api",
		"/debug",
		"/metrics",
		"/status",
		"/.well-known",
		"/.env",
		"/config",
		"/internal",
		"/health", // Health endpoint should be accessed directly, not via short URL
	}
)

// ValidateURL performs comprehensive security validation of a URL
func ValidateURL(urlStr string, ownDomain string) (string, error) {
	// Check length
	if len(urlStr) > MaxURLLength {
		return "", ErrURLTooLong
	}

	// Decode URL encoding first
	decodedURL, err := url.QueryUnescape(urlStr)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidURLFormat, err)
	}

	// Check for URL encoding in decoded URL (double encoding attempt)
	if urlEncodingPattern.MatchString(decodedURL) {
		return "", ErrURLEncodedPayload
	}

	// Check for double protocols
	if doubleProtocolPattern.MatchString(decodedURL) {
		return "", ErrDoubleProtocol
	}

	// Check for SQL-like patterns
	if sqlPattern.MatchString(decodedURL) {
		return "", ErrSQLDetected
	}

	// Parse URL
	parsedURL, err := url.Parse(decodedURL)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidURLFormat, err)
	}

	// Validate scheme (only http and https allowed)
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", ErrInvalidURLScheme
	}

	// Validate host
	host := parsedURL.Hostname()
	if host == "" {
		return "", ErrInvalidURLFormat
	}

	// Check for XSS patterns
	if containsXSS(decodedURL) {
		return "", ErrXSSDetected
	}

	// Check for SSRF
	if isInternalHost(host) {
		return "", ErrSSRFDetected
	}

	// Check for IDN (Internationalized Domain Names)
	if isIDN(host) {
		return "", ErrIDNDetected
	}

	// Check for open redirect to internal paths
	if isOpenRedirect(decodedURL, ownDomain) {
		return "", ErrInternalPath
	}

	// Strip fragments to prevent XSS through fragments
	parsedURL.Fragment = ""

	// Normalize and return
	return strings.TrimSpace(parsedURL.String()), nil
}

// containsXSS checks if URL contains XSS patterns
func containsXSS(urlStr string) bool {
	lowerURL := strings.ToLower(urlStr)

	// Check against known XSS patterns
	for _, pattern := range xssPatterns {
		if strings.Contains(lowerURL, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// isInternalHost checks if hostname points to internal network
func isInternalHost(host string) bool {
	lowerHost := strings.ToLower(host)

	// Check for blocked hostnames
	for _, blocked := range blockedHosts {
		if strings.EqualFold(lowerHost, blocked) ||
			strings.HasSuffix(lowerHost, "."+blocked) {
			return true
		}
	}

	// Check for metadata endpoints
	for _, pattern := range metadataPatterns {
		if strings.Contains(lowerHost, strings.ToLower(pattern)) {
			return true
		}
	}

	// Try to resolve hostname to IP
	ips, err := net.LookupHost(host)
	if err != nil {
		// Can't resolve - might be a hostname, skip IP check
		return false
	}

	// Check if any resolved IP is in private ranges
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}

		if isPrivateIP(ip) {
			return true
		}
	}

	return false
}

// isPrivateIP checks if IP is in private ranges
func isPrivateIP(ip net.IP) bool {
	for _, block := range privateIPBlocks {
		_, ipnet, _ := net.ParseCIDR(block)
		if ipnet != nil && ipnet.Contains(ip) {
			return true
		}
	}
	return false
}

// isIDN checks if domain is an Internationalized Domain Name
func isIDN(domain string) bool {
	// Check for non-ASCII characters
	for _, r := range domain {
		if r > 127 {
			return true
		}
	}
	return false
}

// isOpenRedirect checks if URL is an open redirect to internal path
func isOpenRedirect(originalURL, ownDomain string) bool {
	parsedURL, err := url.Parse(originalURL)
	if err != nil {
		return true // Invalid URL = reject
	}

	// If not pointing to own domain, allow external URLs
	if parsedURL.Hostname() != ownDomain {
		return false
	}

	// Check for internal paths
	path := parsedURL.Path

	// Block all internal paths
	for _, internal := range internalPaths {
		if strings.HasPrefix(path, internal) {
			return true
		}
	}

	return false
}

// SanitizeForOutput sanitizes a URL for safe HTML output
func SanitizeForOutput(urlStr string) string {
	return html.EscapeString(urlStr)
}

// ValidateAndSanitizeURL validates URL and returns sanitized version
func ValidateAndSanitizeURL(urlStr, ownDomain string) (string, error) {
	// Validate the URL
	validatedURL, err := ValidateURL(urlStr, ownDomain)
	if err != nil {
		return "", err
	}

	// Additional sanitization for storage/storage
	return validatedURL, nil
}

// GetRateLimitIdentifier returns a unique identifier for rate limiting
// Prefers IP address, falls back to device ID if needed
func GetRateLimitIdentifier(remoteAddr string, xForwardedFor string, deviceID string) string {
	// Extract IP from X-Forwarded-For or RemoteAddr
	host, _, _ := net.SplitHostPort(remoteAddr)

	if xForwardedFor != "" {
		// Use first IP in chain (most trusted)
		ips := strings.Split(xForwardedFor, ",")
		if len(ips) > 0 {
			host = strings.TrimSpace(ips[0])
		}
	}

	// Prefer IP-based rate limiting for reliability
	if host != "" && host != "" {
		return fmt.Sprintf("ip:%s", host)
	}

	// Fall back to device ID
	if deviceID != "" {
		return fmt.Sprintf("cookie:%s", deviceID)
	}

	// Last resort: use remote address
	return fmt.Sprintf("remote:%s", remoteAddr)
}

// ValidateContentType ensures content type is safe
func ValidateContentType(contentType string) bool {
	allowedTypes := []string{
		"application/json",
		"text/plain",
	}

	for _, allowed := range allowedTypes {
		if strings.HasPrefix(contentType, allowed) {
			return true
		}
	}

	return false
}

// ContainsOnlyASCII checks if string contains only ASCII characters
func ContainsOnlyASCII(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// ValidateCookieValue validates a cookie value
func ValidateCookieValue(value string) error {
	if value == "" {
		return errors.New("cookie value cannot be empty")
	}

	// Check for suspicious patterns
	suspiciousPatterns := []string{
		"<script",
		"javascript:",
		"onerror=",
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(strings.ToLower(value), pattern) {
			return errors.New("cookie contains suspicious content")
		}
	}

	return nil
}

// ValidateHostHeader validates the Host header
func ValidateHostHeader(host, expectedHost string) error {
	if host == "" {
		return errors.New("host header cannot be empty")
	}

	host = strings.Split(host, ":")[0]

	if expectedHost != "" {
		parsedURL, err := url.Parse(expectedHost)
		if err == nil && parsedURL.Hostname() != "" {
			expectedHost = parsedURL.Hostname()
		}

		if host != expectedHost {
			return fmt.Errorf("invalid host header: expected %s, got %s", expectedHost, host)
		}
	}

	return nil
}
