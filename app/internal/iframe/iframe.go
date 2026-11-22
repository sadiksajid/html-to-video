package iframe

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

type CheckResult struct {
	URL     string `json:"url"`
	Working bool   `json:"working"`
	Reason  string `json:"reason,omitempty"`
}

var (
	iframeTagRegex = regexp.MustCompile(`(?i)<iframe[^>]+>`)
	dataSrcRegex   = regexp.MustCompile(`(?i)data-src=["']([^"']+)["']`)
	srcRegex       = regexp.MustCompile(`(?i)\bsrc=["']([^"']+)["']`)
)

func ExtractIframeURLs(htmlContent string) []string {
	var urls []string
	seen := make(map[string]bool)

	tags := iframeTagRegex.FindAllString(htmlContent, -1)
	for _, tag := range tags {
		var foundUrl string
		if m := dataSrcRegex.FindStringSubmatch(tag); len(m) > 1 {
			foundUrl = strings.TrimSpace(m[1])
		} else if m := srcRegex.FindStringSubmatch(tag); len(m) > 1 {
			foundUrl = strings.TrimSpace(m[1])
		}

		if foundUrl != "" && strings.HasPrefix(foundUrl, "http") && !seen[foundUrl] {
			seen[foundUrl] = true
			urls = append(urls, foundUrl)
		}
	}

	return urls
}

func CheckURL(targetURL string) CheckResult {
	parsed, err := url.Parse(targetURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return CheckResult{
			URL:     targetURL,
			Working: false,
			Reason:  "Invalid URL format",
		}
	}

	client := &http.Client{
		Timeout: 6 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return CheckResult{
			URL:     targetURL,
			Working: false,
			Reason:  err.Error(),
		}
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return CheckResult{
			URL:     targetURL,
			Working: false,
			Reason:  fmt.Sprintf("Failed to reach website: %v", err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return CheckResult{
			URL:     targetURL,
			Working: false,
			Reason:  fmt.Sprintf("Website returned HTTP status %d", resp.StatusCode),
		}
	}

	// Check X-Frame-Options
	xfo := resp.Header.Get("X-Frame-Options")
	if xfo != "" {
		upper := strings.ToUpper(xfo)
		if strings.Contains(upper, "DENY") || strings.Contains(upper, "SAMEORIGIN") {
			return CheckResult{
				URL:     targetURL,
				Working: false,
				Reason:  fmt.Sprintf("Website blocks iframes via X-Frame-Options: %s", xfo),
			}
		}
	}

	// Check Content-Security-Policy frame-ancestors
	csp := resp.Header.Get("Content-Security-Policy")
	if csp != "" && strings.Contains(strings.ToLower(csp), "frame-ancestors") {
		// If frame-ancestors is present and not '*', it restricts embedding
		lower := strings.ToLower(csp)
		if !strings.Contains(lower, "frame-ancestors *") {
			return CheckResult{
				URL:     targetURL,
				Working: false,
				Reason:  "Website blocks iframe embedding via Content-Security-Policy (frame-ancestors)",
			}
		}
	}

	return CheckResult{
		URL:     targetURL,
		Working: true,
	}
}

func CheckReelFile(htmlPath string) ([]CheckResult, error) {
	bytes, err := os.ReadFile(htmlPath)
	if err != nil {
		return nil, err
	}

	urls := ExtractIframeURLs(string(bytes))
	results := make([]CheckResult, 0, len(urls))

	for _, u := range urls {
		results = append(results, CheckURL(u))
	}

	return results, nil
}
