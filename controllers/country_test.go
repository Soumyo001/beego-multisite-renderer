package controllers

import "testing"

func TestCountryFromSlug(t *testing.T) {
	for _, tc := range []struct {
		slug, want string
		valid      bool
	}{
		{"usa", "USA", true},
		{"canada", "Canada", true},
		{"united-kingdom", "United Kingdom", true},
		{"USA", "", false},
		{"usa%2fcanada", "", false},
		{"us--ca", "", false},
		{"", "", false},
	} {
		got, valid := countryFromSlug(tc.slug)
		if got != tc.want || valid != tc.valid {
			t.Errorf(
				"countryFromSlug(%q) = (%q, %v), want (%q, %v)",
				tc.slug, got, valid, tc.want, tc.valid,
			)
		}
	}
}

func TestSafeBooking(t *testing.T) {
	if got := safeBooking(
		"Booking.com",
		"https://www.booking.com/hotel/us/example.html",
	); got == "" {
		t.Fatal("expected a valid booking-provider URL")
	}

	for _, raw := range []string{
		"https://booking.com.evil.example/hotel/steal",
		"javascript:alert(1)",
		"https://booking.com/",
		"https://user:pass@booking.com/hotel/example",
	} {
		if got := safeBooking("Booking.com", raw); got != "" {
			t.Errorf("unsafe provider URL was accepted: %q", raw)
		}
	}
}

func TestCountryPageBounds(t *testing.T) {
	cases := []struct {
		name      string
		total     int
		raw       string
		page      int
		pageCount int
		start     int
		end       int
		valid     bool
	}{
		{
			name: "first of fifteen", total: 15, raw: "",
			page: 1, pageCount: 2, start: 0, end: 8, valid: true,
		},
		{
			name: "second of fifteen", total: 15, raw: "2",
			page: 2, pageCount: 2, start: 8, end: 15, valid: true,
		},
		{
			name: "empty country", total: 0, raw: "",
			page: 1, pageCount: 1, start: 0, end: 0, valid: true,
		},
		{
			name: "page beyond last", total: 15, raw: "200",
			page: 2, pageCount: 2, start: 8, end: 15, valid: true,
		},
		{
			name: "page three clamps to two", total: 15, raw: "3",
			page: 2, pageCount: 2, start: 8, end: 15, valid: true,
		},
		{
			name: "zero page clamps to first", total: 15, raw: "0",
			page: 1, pageCount: 2, start: 0, end: 8, valid: true,
		},
		{
			name: "negative page clamps to first", total: 15, raw: "-5",
			page: 1, pageCount: 2, start: 0, end: 8, valid: true,
		},
		{
			name: "invalid page", total: 15, raw: "abc",
			valid: false,
		},
		{
			name:  "empty country second page clamps to first",
			total: 0, raw: "2",
			page: 1, pageCount: 1, start: 0, end: 0, valid: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			page, count, start, end, ok := countryPageBounds(
				tc.total, tc.raw,
			)

			if ok != tc.valid {
				t.Fatalf(
					"valid = %v, want %v",
					ok, tc.valid,
				)
			}

			if !tc.valid {
				return
			}

			if page != tc.page ||
				count != tc.pageCount ||
				start != tc.start ||
				end != tc.end {
				t.Fatalf(
					"got page=%d count=%d start=%d end=%d; "+
						"want page=%d count=%d start=%d end=%d",
					page, count, start, end,
					tc.page, tc.pageCount, tc.start, tc.end,
				)
			}
		})
	}
}
