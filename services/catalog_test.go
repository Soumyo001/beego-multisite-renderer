package services

import "testing"

func TestCatalogFromUploadedData(t *testing.T) {
	catalog, err := NewCatalog(
		"../content",
		"../conf/deployment.json",
	)
	if err != nil {
		t.Fatal(err)
	}

	if catalog.Count() != 4 {
		t.Fatalf(
			"expected 4 pages, got %d",
			catalog.Count(),
		)
	}

	host := "test-travel-templates.beta.123presto.com"

	tests := []struct {
		urlPath  string
		jsonFile string
	}{
		{"/", "home.json"},
		{"/privacy-policy", "privacy-policy.json"},
		{"/site-terms", "site-terms.json"},
		{
			"/blog/untitled-post-wrurmp",
			"untitled-post-wrurmp.json",
		},
	}

	for _, test := range tests {
		got, found := catalog.Resolve(
			host, test.urlPath,
		)

		expected := "beta/preview/test-travel-templates/" +
			test.jsonFile

		if !found || got.PageJSON != expected {
			t.Errorf(
				"incorrect resolution for %q: %+v, found %v",
				test.urlPath, got, found,
			)
		}
	}

	for _, request := range []struct {
		host    string
		urlPath string
	}{
		{"other.example.com", "/"},
		{host, "/missing"},
		{host, "/blog"},
	} {
		if _, found := catalog.Resolve(
			request.host, request.urlPath,
		); found {
			t.Errorf(
				"unexpected resolution for %s%s",
				request.host,
				request.urlPath,
			)
		}
	}
}

func TestCatalogLocalSiteHost(t *testing.T) {
	catalog, err := NewCatalog("../content", "../conf/deployment.json")
	if err != nil {
		t.Fatal(err)
	}

	const publisher = "test-travel-templates.beta.123presto.com"

	// Without a local site host, localhost is just an unknown host.
	if _, found := catalog.Resolve("127.0.0.1:8080", "/"); found {
		t.Fatal("localhost resolved before SetLocalSiteHost")
	}

	if err := catalog.SetLocalSiteHost("unknown.example.com"); err == nil {
		t.Fatal("accepted a host with no routes")
	}
	if err := catalog.SetLocalSiteHost(publisher); err != nil {
		t.Fatal(err)
	}

	for _, host := range []string{"localhost:8080", "127.0.0.1:8080", "[::1]:8080", "LOCALHOST"} {
		got, found := catalog.Resolve(host, "/privacy-policy")
		if !found || got.Domain != publisher {
			t.Errorf("%s: found=%v domain=%q", host, found, got.Domain)
		}
	}

	// Other hosts are unaffected.
	for _, host := range []string{"192.168.0.233:8080", "example.com"} {
		if _, found := catalog.Resolve(host, "/"); found {
			t.Errorf("%s resolved", host)
		}
	}
	if _, found := catalog.Resolve(publisher, "/"); !found {
		t.Error("the publisher host no longer resolves")
	}
}
