package util

import (
	"errors"
	"net/url"
	"strings"
)

func NormalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("empty url")
	}

	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}

	if !u.IsAbs() || u.Host == "" {
		return "", errors.New("invalid url")
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.New("unsupported url scheme")
	}

	u.Fragment = ""

	return u.String(), nil
}
