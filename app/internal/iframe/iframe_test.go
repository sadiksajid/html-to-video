package iframe

import (
	"testing"
)

func TestExtractIframeURLs(t *testing.T) {
	html := `
	<div class="mockup">
		<iframe class="pif" data-src="https://pixlr.com/" allow="camera"></iframe>
		<iframe src="https://example.com/demo" width="300"></iframe>
		<iframe class="pif" data-src="https://pixlr.com/"></iframe> <!-- duplicate -->
	</div>
	`

	urls := ExtractIframeURLs(html)
	if len(urls) != 2 {
		t.Fatalf("Expected 2 unique URLs, got %d: %v", len(urls), urls)
	}

	if urls[0] != "https://pixlr.com/" || urls[1] != "https://example.com/demo" {
		t.Errorf("Unexpected URLs extracted: %v", urls)
	}
}

func TestCheckPixlrBlock(t *testing.T) {
	// Online test checking pixlr.com
	res := CheckURL("https://pixlr.com/")
	t.Logf("Pixlr Check Result: Working=%v, Reason=%s", res.Working, res.Reason)
	if res.Working {
		t.Errorf("Expected https://pixlr.com/ to be detected as not working / blocked, but got working=true")
	}
}
