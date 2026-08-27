package store

import "testing"

func TestValidateImageURLAcceptsLocalUploadPath(t *testing.T) {
	if err := ValidateImageURL("/uploads/a/b.png", nil); err != nil {
		t.Fatalf("expected local upload path to pass, got %v", err)
	}
	if err := ValidateImageURL("/uploads/cover.jpg", []string{"cdn.example.com"}); err != nil {
		t.Fatalf("expected local upload path to pass with allowlist, got %v", err)
	}
}

func TestValidateImageURLAcceptsAllowedHTTPSDomain(t *testing.T) {
	if err := ValidateImageURL("https://cdn.example.com/img/a.png", []string{"cdn.example.com"}); err != nil {
		t.Fatalf("expected allowed https domain to pass, got %v", err)
	}
}

func TestValidateImageURLRejectsSchemeForms(t *testing.T) {
	allow := []string{"cdn.example.com"}
	cases := []string{
		"javascript:alert(1)",
		"data:image/png;base64,AAAA",
		"file:///etc/passwd",
		"ftp://cdn.example.com/a.png",
		"//cdn.example.com/a.png",
		"not a url",
		"cdn.example.com/a.png", // no scheme
	}
	for _, image := range cases {
		if err := ValidateImageURL(image, allow); err == nil {
			t.Fatalf("expected rejection for %q", image)
		}
	}
}

func TestValidateImageURLRejectsPrivateAndLoopback(t *testing.T) {
	allow := []string{"cdn.example.com"}
	cases := []string{
		"https://127.0.0.1/a.png",
		"https://localhost/a.png",
		"https://10.0.0.5/a.png",
		"https://172.16.0.1/a.png",
		"https://192.168.1.20/a.png",
		"https://169.254.169.254/a.png",
	}
	for _, image := range cases {
		if err := ValidateImageURL(image, allow); err == nil {
			t.Fatalf("expected rejection for private URL %q", image)
		}
	}
}

func TestValidateImageURLRejectsNonAllowlistedDomain(t *testing.T) {
	if err := ValidateImageURL("https://evil.example.net/a.png", []string{"cdn.example.com"}); err == nil {
		t.Fatal("expected rejection for a host outside the allowlist")
	}
	// A sibling domain must not satisfy a wildcard-free allowlist.
	if err := ValidateImageURL("https://notcdn.example.com/a.png", []string{"cdn.example.com"}); err == nil {
		t.Fatal("expected rejection for a sibling domain")
	}
}

func TestValidateImageURLAllowsWildcardDomain(t *testing.T) {
	if err := ValidateImageURL("https://img.cdn.example.com/a.png", []string{"*.cdn.example.com"}); err != nil {
		t.Fatalf("expected wildcard allowlist match, got %v", err)
	}
}

func TestValidateImageURLsRejectsAnyBadImage(t *testing.T) {
	err := ValidateImageURLs([]string{"/uploads/a.png", "javascript:bad()"}, nil)
	if err == nil {
		t.Fatal("expected ValidateImageURLs to reject the bad entry")
	}
	if err := ValidateImageURLs([]string{"/uploads/a.png", "/uploads/b.png"}, nil); err != nil {
		t.Fatalf("expected all-local images to pass, got %v", err)
	}
}

func TestValidateImageURLRejectsEmpty(t *testing.T) {
	if err := ValidateImageURL("", nil); err == nil {
		t.Fatal("expected an empty url to be rejected")
	}
	if err := ValidateImageURL("   ", nil); err == nil {
		t.Fatal("expected a whitespace-only url to be rejected")
	}
}

func TestValidateImageURLsDeduplicates(t *testing.T) {
	if err := ValidateImageURLs([]string{"javascript:bad()", "javascript:bad()"}, nil); err == nil {
		t.Fatal("expected rejection even when the bad value is duplicated")
	}
}
