package main

import (
	"fmt"
	"sort"
	"strings"

	"sparrow-ssr/models"
)

// internationalListing is one illustrative listing for the
// home-page tiles. Each demo redirects to a real Booking.com
// hotel page in the same city (taken from Booking.com's own
// city listing pages on 23 Nov 2026). The demo is NOT that hotel.
type internationalListing struct {
	Name       string
	Type       string
	Area       string
	MaxGuests  int
	SizeSqFt   int
	PriceFrom  int
	BookingURL string
	Amenities  []string
	Eco        bool
}

var torontoLocation = models.PropertyLocation{
	City: "Toronto", State: "Ontario", Country: "Canada",
}

var dhakaLocation = models.PropertyLocation{
	City: "Dhaka", State: "Dhaka Division", Country: "Bangladesh",
}

// Fallback redirect pages if a hotel page ever disappears.
// Both were confirmed to exist on 23 Nov 2026.
const (
	torontoCityPage = "https://www.booking.com/city/ca/toronto.html"
	dhakaCityPage   = "https://www.booking.com/city/bd/dhaka.html"
)

var torontoListings = []internationalListing{
	{"Harbourfront Lakeview Suites", "Apartment", "Harbourfront", 4, 820, 410,
		"https://www.booking.com/hotel/ca/the-westin-harbour-castle-toronto-ontario.html",
		[]string{"Air Conditioning", "Free Wifi", "Fitness center", "Kitchen"}, false},
	{"King Street Loft Residence", "Apartment", "Entertainment District", 4, 760, 385,
		"https://www.booking.com/hotel/ca/nobu-toronto.html",
		[]string{"Air Conditioning", "Free Wifi", "Kitchen"}, false},
	{"Yorkville Garden Hotel", "Hotel", "Bloor-Yorkville", 2, 420, 620,
		"https://www.booking.com/hotel/ca/four-seasons-toronto-at-yorkville.html",
		[]string{"Spa and wellness center", "Fitness center", "Restaurant", "Free Wifi"}, false},
	{"Union Station Grand Hotel", "Hotel", "Financial District", 2, 390, 455,
		"https://www.booking.com/hotel/ca/fairmont-royal-york.html",
		[]string{"Pool", "Fitness center", "Restaurant", "Air Conditioning"}, false},
	{"Old Town Heritage Inn", "Hotel", "Old Town Toronto", 2, 360, 340,
		"https://www.booking.com/hotel/ca/omni-king-edward-hotel.html",
		[]string{"Restaurant", "Fitness center", "Free Wifi"}, false},
	{"Queen West Urban Stay", "Hotel", "Entertainment District", 2, 340, 295,
		"https://www.booking.com/hotel/ca/bisha-toronto.html",
		[]string{"Pool", "Restaurant", "Free Wifi", "Air Conditioning"}, false},
	{"University Avenue Skyline Suites", "Hotel", "Entertainment District", 4, 480, 520,
		"https://www.booking.com/hotel/ca/shangri-la-toronto.html",
		[]string{"Pool", "Spa and wellness center", "Restaurant"}, false},
	{"Eaton Centre City Hotel", "Hotel", "Downtown Toronto", 2, 330, 260,
		"https://www.booking.com/hotel/ca/le-germain-toronto.html",
		[]string{"Restaurant", "Free Wifi", "Air Conditioning"}, false},
	{"Front Street Tower Hotel", "Hotel", "Downtown Toronto", 4, 400, 230,
		"https://www.booking.com/hotel/ca/riu-plaza-toronto.html",
		[]string{"Pool", "Fitness center", "Restaurant"}, false},
	{"Chinatown Corner Rooms", "Hotel", "Downtown Toronto", 4, 350, 210,
		"https://www.booking.com/hotel/ca/doubletree-toronto-downtown.html",
		[]string{"Pool", "Fitness center", "Free Wifi"}, false},
	{"Bay Street Executive Suites", "Hotel", "Financial District", 2, 450, 575,
		"https://www.booking.com/hotel/ca/st-regis-toronto.html",
		[]string{"Pool", "Spa and wellness center", "Fitness center", "Restaurant"}, false},
	{"Parkdale Green Guesthouse", "House", "Church-Wellesley", 6, 1100, 180,
		"https://www.booking.com/hotel/ca/the-anndore-house.html",
		[]string{"Free Wifi", "Kitchen", "Garden"}, true},
}

var dhakaListings = []internationalListing{
	{"Gulshan Lakeside Residence", "Hotel", "Gulshan", 2, 380, 165,
		"https://www.booking.com/hotel/bd/the-westin-dhaka.html",
		[]string{"Pool", "Spa and wellness center", "Fitness center", "Restaurant"}, false},
	{"Gulshan Avenue Grand Hotel", "Hotel", "Gulshan", 2, 360, 150,
		"https://www.booking.com/hotel/bd/renaissance-dhaka-gulshan-dhaka.html",
		[]string{"Pool", "Fitness center", "Restaurant", "Free Wifi"}, false},
	{"North Avenue Business Hotel", "Hotel", "Gulshan", 2, 340, 135,
		"https://www.booking.com/hotel/bd/crowne-plaza-dhaka-gulshan.html",
		[]string{"Fitness center", "Restaurant", "Air Conditioning"}, false},
	{"Minto Road Heritage Hotel", "Hotel", "Shahbagh", 2, 400, 185,
		"https://www.booking.com/hotel/bd/intercontinental-hotels-dhaka.html",
		[]string{"Pool", "Spa and wellness center", "Restaurant"}, false},
	{"Tejgaon City Centre Suites", "Hotel", "Tejgaon", 4, 420, 120,
		"https://www.booking.com/hotel/bd/holiday-inn-dhaka-city-centre.html",
		[]string{"Pool", "Fitness center", "Free Wifi"}, false},
	{"Uttara Maple Garden Inn", "Hotel", "Uttara", 2, 300, 75,
		"https://www.booking.com/hotel/bd/best-western-plus-maple-leaf.html",
		[]string{"Restaurant", "Free Wifi", "Air Conditioning"}, false},
	{"Banani Serviced Apartments", "Apartment", "Banani", 4, 900, 95,
		"https://www.booking.com/hotel/bd/ascott-palace.html",
		[]string{"Kitchen", "Free Wifi", "Air Conditioning", "Parking"}, false},
	{"Dhanmondi Lake View Rooms", "Hotel", "Dhanmondi", 2, 280, 55,
		"https://www.booking.com/hotel/bd/the-capital-ltd.html",
		[]string{"Restaurant", "Free Wifi", "Air Conditioning"}, false},
	{"Motijheel Pearl Residence", "Hotel", "Motijheel", 2, 260, 45,
		"https://www.booking.com/hotel/bd/pearl-dhaka.html",
		[]string{"Free Wifi", "Air Conditioning", "Parking"}, false},
	{"Baridhara Green Guesthouse", "House", "Baridhara", 6, 1400, 110,
		"https://www.booking.com/hotel/bd/white-palace.html",
		[]string{"Garden", "Kitchen", "Free Wifi", "Parking"}, true},
	{"Banani Rooftop Studio", "Apartment", "Banani", 2, 450, 60,
		"https://www.booking.com/hotel/bd/sarina.html",
		[]string{"Kitchen", "Free Wifi", "Air Conditioning"}, false},
	{"Old Dhaka Courtyard Stay", "House", "Old Dhaka", 4, 800, 40,
		"https://www.booking.com/hotel/bd/71.html",
		[]string{"Free Wifi", "Breakfast included", "Air Conditioning"}, false},
}

// verifiedPhotoPool returns the distinct Unsplash photo IDs that
// -verify-images already checks (galleryExtraPhotoIDs), sorted.
func verifiedPhotoPool() []string {
	seen := make(map[string]bool)
	for _, ids := range galleryExtraPhotoIDs {
		for _, id := range ids {
			seen[id] = true
		}
	}

	pool := make([]string, 0, len(seen))
	for id := range seen {
		pool = append(pool, id)
	}
	sort.Strings(pool)
	return pool
}

// internationalDemoSeedProperties builds 24 illustrative listings:
// prop-049..prop-060 in Toronto and prop-061..prop-072 in Dhaka.
// They exist for the home-page tiles_component blocks.
func internationalDemoSeedProperties() ([]seedProperty, error) {
	pool := verifiedPhotoPool()
	if len(pool) < 5 {
		return nil, fmt.Errorf("photo pool has only %d images", len(pool))
	}

	groups := []struct {
		location models.PropertyLocation
		listings []internationalListing
		hotelURL string
	}{
		{torontoLocation, torontoListings, "https://www.booking.com/hotel/ca/"},
		{dhakaLocation, dhakaListings, "https://www.booking.com/hotel/bd/"},
	}

	result := make([]seedProperty, 0, 24)
	n := 0

	for _, group := range groups {
		if len(group.listings) != 12 {
			return nil, fmt.Errorf(
				"%s: expected 12 listings, got %d",
				group.location.City, len(group.listings),
			)
		}

		for _, l := range group.listings {
			if !strings.HasPrefix(l.BookingURL, group.hotelURL) {
				return nil, fmt.Errorf(
					"%s: booking URL %q is not a %s hotel page",
					l.Name, l.BookingURL, group.location.City,
				)
			}

			location := group.location
			location.Area = l.Area

			// Five consecutive distinct photos from the verified pool.
			images := make([]string, 0, 5)
			for k := 0; k < 5; k++ {
				id := pool[(n*5+k)%len(pool)]
				images = append(images,
					"https://images.unsplash.com/"+id+
						"?auto=format&fit=crop&w=1000&q=75")
			}

			reviews := 60 + n*17
			result = append(result, seedProperty{
				ID:   fmt.Sprintf("prop-%03d", 49+n),
				Slug: "demo-" + strings.ToLower(strings.ReplaceAll(l.Name, " ", "-")),
				Name: l.Name + " (Demo)",
				Type: l.Type,

				Location:        location,
				MaxGuests:       l.MaxGuests,
				SizeSqFt:        l.SizeSqFt,
				Rating:          7.8 + float64(n%13)/10,
				ReviewCount:     reviews,
				PopularityScore: reviews,
				PriceFrom:       l.PriceFrom,
				Currency:        "USD",
				Amenities:       append([]string(nil), l.Amenities...),
				ProviderName:    "Booking.com",
				BookingURL:      l.BookingURL,
				EcoFriendly:     l.Eco,
				Images:          images,
				AvailableFrom:   "2026-11-15",
			})
			n++
		}
	}

	return result, nil
}
