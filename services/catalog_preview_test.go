package services

import "testing"

func TestRegisterPreviewAlias(t *testing.T) {
	root := t.TempDir()

	alphaFile := testDeployment(
		t, root, "alpha", "alpha.example.com",
	)
	bravoFile := testDeployment(
		t, root, "bravo", "bravo.example.com",
	)

	catalog, err := NewMultiCatalog(
		root,
		[]string{alphaFile, bravoFile},
	)
	if err != nil {
		t.Fatal(err)
	}

	before := catalog.Count()

	err = catalog.RegisterPreviewAlias(
		"alpha.example.com",
		"sparrow-demo.onrender.com",
	)
	if err != nil {
		t.Fatal(err)
	}

	if catalog.Count() != before+1 {
		t.Fatalf(
			"expected one additional route, got %d",
			catalog.Count(),
		)
	}

	original, found := catalog.Resolve(
		"alpha.example.com", "/",
	)
	if !found {
		t.Fatal("original publisher route disappeared")
	}

	preview, found := catalog.Resolve(
		"sparrow-demo.onrender.com", "/",
	)
	if !found {
		t.Fatal("preview route was not registered")
	}

	if preview != original {
		t.Fatalf(
			"preview location changed publisher identity: %+v",
			preview,
		)
	}

	if _, found := catalog.Resolve(
		"bravo.example.com", "/",
	); !found {
		t.Fatal("second publisher route disappeared")
	}

	for _, tc := range []struct {
		host string
		path string
	}{
		{"unknown.example.com", "/"},
		{"sparrow-demo.onrender.com", "/missing"},
	} {
		if _, found := catalog.Resolve(
			tc.host, tc.path,
		); found {
			t.Fatalf(
				"unexpected route for %s%s",
				tc.host, tc.path,
			)
		}
	}

	if err := catalog.RegisterPreviewAlias(
		"bravo.example.com",
		"sparrow-demo.onrender.com",
	); err == nil {
		t.Fatal("duplicate preview Host was accepted")
	}

	if err := catalog.RegisterPreviewAlias(
		"unknown.example.com",
		"another-demo.onrender.com",
	); err == nil {
		t.Fatal("unregistered source Host was accepted")
	}

	if err := catalog.RegisterPreviewAlias(
		"alpha.example.com",
		"https://invalid.example.com",
	); err == nil {
		t.Fatal("invalid preview hostname was accepted")
	}
}

func TestTwoPreviewHostsPreservePublisherIsolation(t *testing.T) {
	root := t.TempDir()

	alphaFile := testDeployment(
		t, root, "alpha", "alpha.example.com",
	)
	bravoFile := testDeployment(
		t, root, "bravo", "bravo.example.com",
	)

	catalog, err := NewMultiCatalog(
		root,
		[]string{alphaFile, bravoFile},
	)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		publisher string
		preview   string
		directory string
	}{
		{
			"alpha.example.com",
			"alpha-preview.example.com",
			"beta/preview/alpha",
		},
		{
			"bravo.example.com",
			"bravo-preview.example.com",
			"beta/preview/bravo",
		},
	}

	for _, tc := range cases {
		if err := catalog.RegisterPreviewAlias(
			tc.publisher,
			tc.preview,
		); err != nil {
			t.Fatal(err)
		}

		original, ok := catalog.Resolve(tc.publisher, "/")
		if !ok {
			t.Fatalf("missing publisher %q", tc.publisher)
		}

		preview, ok := catalog.Resolve(tc.preview, "/")
		if !ok {
			t.Fatalf("missing preview %q", tc.preview)
		}

		if preview != original {
			t.Fatalf(
				"%s changed publisher PageLocation",
				tc.preview,
			)
		}

		if preview.ContentDirectory != tc.directory {
			t.Fatalf(
				"%s resolved to wrong site: %s",
				tc.preview,
				preview.ContentDirectory,
			)
		}
	}

	// Two publisher routes and two preview routes.
	if got := catalog.Count(); got != 4 {
		t.Fatalf("expected 4 routes, got %d", got)
	}

	// An unknown Host never resolves to either publisher.
	if _, ok := catalog.Resolve(
		"unknown.example.com", "/",
	); ok {
		t.Fatal("unknown Host was accepted")
	}

	// A preview Host cannot be reassigned to another site.
	if err := catalog.RegisterPreviewAlias(
		"bravo.example.com",
		"alpha-preview.example.com",
	); err == nil {
		t.Fatal("duplicate preview Host was accepted")
	}
}
