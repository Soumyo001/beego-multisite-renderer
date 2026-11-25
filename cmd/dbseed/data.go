package main

import (
	"beego-multisite-renderer/models"
	"fmt"
	"strings"
)

// seedProperties is self-contained initial data. It does not read any JSON files.
// Prices, maximum guests, ratings, and availability are illustrative assessment data.
var seedProperties = []seedProperty{
	{
		ID: "prop-001", Name: "The Plaza, A Fairmont Hotel", Type: "Hotel",
		Slug:      "the-plaza-a-fairmont-hotel",
		Location:  models.PropertyLocation{City: "New York", State: "New York", Area: "Midtown Manhattan", Country: "USA"},
		MaxGuests: 2, PriceFrom: 890, Currency: "USD",
		Rating: 8.6, ReviewCount: 509, PopularityScore: 509,
		Amenities:    []string{"Spa and wellness center", "Free Wifi", "Fitness center", "Air Conditioning"},
		ProviderName: "Booking.com", BookingURL: "https://www.booking.com/hotel/us/the-plaza.html",
		Images:        []string{"https://images.unsplash.com/photo-1570129477492-45c003edd2be"},
		AvailableFrom: "2026-11-01",
	},
	{
		ID: "prop-002", Name: "Hilton Hawaiian Village Waikiki Beach Resort", Type: "Resort",
		Slug:      "hilton-hawaiian-village-waikiki-beach-resort",
		Location:  models.PropertyLocation{City: "Honolulu", State: "Hawaii", Area: "Waikiki", Country: "USA"},
		MaxGuests: 6, PriceFrom: 315, Currency: "USD",
		Rating: 8.4, ReviewCount: 10199, PopularityScore: 10199,
		Amenities:    []string{"Pool", "Spa", "Restaurant", "Air Conditioning"},
		ProviderName: "Expedia", BookingURL: "https://www.expedia.com/Honolulu-Hotels-Hilton-Hawaiian-Village-Waikiki-Beach-Resort.h13857.Hotel-Information",
		Images:        []string{"https://images.unsplash.com/photo-1600607687939-ce8a6c25118c"},
		AvailableFrom: "2026-11-01",
	},
	{
		ID: "prop-003", Name: "The Venetian Resort Las Vegas", Type: "Resort",
		Slug:      "the-venetian-resort-las-vegas",
		Location:  models.PropertyLocation{City: "Las Vegas", State: "Nevada", Area: "Las Vegas Strip", Country: "USA"},
		MaxGuests: 8, PriceFrom: 360, Currency: "USD",
		Rating: 8.9, ReviewCount: 9826, PopularityScore: 9826,
		Amenities:    []string{"Pool", "Spa and wellness center", "Fitness center", "Restaurant"},
		ProviderName: "Booking.com", BookingURL: "https://www.booking.com/hotel/us/the-venetian-resort-casino.html",
		Images:        []string{"https://images.unsplash.com/photo-1590490360182-c33d57733427"},
		AvailableFrom: "2026-11-01",
	},
	{
		ID: "prop-004", Name: "Hilton Waikiki Beach Resort & Spa", Type: "Resort",
		Slug:      "hilton-waikiki-beach-resort-spa",
		Location:  models.PropertyLocation{City: "Honolulu", State: "Hawaii", Area: "Waikiki", Country: "USA"},
		MaxGuests: 4, PriceFrom: 265, Currency: "USD",
		Rating: 8.6, ReviewCount: 2439, PopularityScore: 2439,
		Amenities:    []string{"Pool", "Restaurant", "Air Conditioning"},
		ProviderName: "Expedia", BookingURL: "https://www.expedia.com/Honolulu-Hotels-Hilton-Waikiki-Beach.h25227.Hotel-Information",
		Images:        []string{"https://images.unsplash.com/photo-1470770841072-f978cf4d019e"},
		AvailableFrom: "2026-11-01",
	},
	{
		ID: "prop-005", Name: "Fontainebleau Miami Beach", Type: "Resort",
		Slug:      "fontainebleau-miami-beach",
		Location:  models.PropertyLocation{City: "Miami Beach", State: "Florida", Area: "Mid-Beach", Country: "USA"},
		MaxGuests: 6, PriceFrom: 495, Currency: "USD",
		Rating: 7.5, ReviewCount: 2136, PopularityScore: 2136,
		Amenities:    []string{"Pool", "Spa and wellness center", "Beachfront", "Fitness center"},
		ProviderName: "Booking.com", BookingURL: "https://www.booking.com/hotel/us/fontainebleau-resort.html",
		Images:        []string{"https://images.unsplash.com/photo-1512917774080-9991f1c4c750"},
		AvailableFrom: "2026-11-01",
	},
	{
		ID: "prop-006", Name: "Waikiki Beach Marriott Resort & Spa", Type: "Resort",
		Slug:      "waikiki-beach-marriott-resort-spa",
		Location:  models.PropertyLocation{City: "Honolulu", State: "Hawaii", Area: "Waikiki", Country: "USA"},
		MaxGuests: 4, PriceFrom: 355, Currency: "USD",
		Rating: 9.0, ReviewCount: 3863, PopularityScore: 3863,
		Amenities:    []string{"Pool", "Spa", "Restaurant"},
		ProviderName: "Expedia", BookingURL: "https://www.expedia.com/Honolulu-Hotels-Waikiki-Beach-Marriott-Resort-Spa.h5306.Hotel-Information",
		Images:        []string{"https://images.unsplash.com/photo-1571896349842-33c89424de2d"},
		AvailableFrom: "2026-11-01",
	},
	{
		ID: "prop-007", Name: "The Beverly Hills Hotel - Dorchester Collection", Type: "Hotel",
		Slug:      "the-beverly-hills-hotel-dorchester-collection",
		Location:  models.PropertyLocation{City: "Los Angeles", State: "California", Area: "Beverly Hills", Country: "USA"},
		MaxGuests: 2, PriceFrom: 1250, Currency: "USD",
		Rating: 9.2, ReviewCount: 166, PopularityScore: 166,
		Amenities:    []string{"Pool", "Spa and wellness center", "Fitness center", "Restaurant"},
		ProviderName: "Booking.com", BookingURL: "https://www.booking.com/hotel/us/the-beverly-hills.html",
		Images:        []string{"https://images.unsplash.com/photo-1600607687644-c7171b42498f"},
		AvailableFrom: "2026-11-01",
	},
	{
		ID: "prop-008", Name: "Waikiki Resort Hotel", Type: "Hotel",
		Slug:      "waikiki-resort-hotel",
		Location:  models.PropertyLocation{City: "Honolulu", State: "Hawaii", Area: "Waikiki", Country: "USA"},
		MaxGuests: 4, PriceFrom: 180, Currency: "USD",
		Rating: 8.4, ReviewCount: 3696, PopularityScore: 3696,
		Amenities:    []string{"Pool", "Free Wifi", "Restaurant", "Air Conditioning"},
		ProviderName: "Expedia", BookingURL: "https://www.expedia.com/Honolulu-Hotels-Waikiki-Resort-Hotel.h13809.Hotel-Information",
		Images:        []string{"https://images.unsplash.com/photo-1499793983690-e29da59ef1c2"},
		AvailableFrom: "2026-11-01",
	},
	{
		ID: "prop-009", Name: "Hyatt Regency Orlando", Type: "Hotel",
		Slug:      "hyatt-regency-orlando",
		Location:  models.PropertyLocation{City: "Orlando", State: "Florida", Area: "International Drive", Country: "USA"},
		MaxGuests: 4, PriceFrom: 280, Currency: "USD",
		Rating: 8.6, ReviewCount: 795, PopularityScore: 795,
		Amenities:    []string{"Pool", "Spa and wellness center", "Fitness center", "Parking"},
		ProviderName: "Booking.com", BookingURL: "https://www.booking.com/hotel/us/hyatt-regency-orlando.html",
		Images:        []string{"https://images.unsplash.com/photo-1600596542815-ffad4c1539a9"},
		AvailableFrom: "2026-11-01",
	},
	{
		ID: "prop-010", Name: "'Alohilani Resort Waikiki Beach", Type: "Resort",
		Slug:      "alohilani-resort-waikiki-beach",
		Location:  models.PropertyLocation{City: "Honolulu", State: "Hawaii", Area: "Waikiki", Country: "USA"},
		MaxGuests: 6, PriceFrom: 400, Currency: "USD",
		Rating: 8.8, ReviewCount: 4321, PopularityScore: 4321,
		Amenities:    []string{"Pool", "Spa"},
		ProviderName: "Expedia", BookingURL: "https://www.expedia.com/Honolulu-Hotels-Alohilani-Resort-Waikiki-Beach.h21781.Hotel-Information",
		Images:        []string{"https://images.unsplash.com/photo-1600585154340-be6161a56a0c"},
		AvailableFrom: "2026-11-01",
	},
	{
		ID: "prop-011", Name: "Hotel del Coronado, Curio Collection by Hilton", Type: "Resort",
		Slug:      "hotel-del-coronado-curio-collection-by-hilton",
		Location:  models.PropertyLocation{City: "San Diego", State: "California", Area: "Coronado", Country: "USA"},
		MaxGuests: 8, PriceFrom: 640, Currency: "USD",
		Rating: 8.2, ReviewCount: 344, PopularityScore: 344,
		Amenities:    []string{"Pool", "Beachfront", "Spa and wellness center"},
		ProviderName: "Booking.com", BookingURL: "https://www.booking.com/hotel/us/del-coronado.html",
		Images:        []string{"https://images.unsplash.com/photo-1522708323590-d24dbb6b0267"},
		AvailableFrom: "2026-11-15",
	},
	{
		ID: "prop-012", Name: "Hyatt Place Waikiki Beach", Type: "Hotel",
		Slug:      "hyatt-place-waikiki-beach",
		Location:  models.PropertyLocation{City: "Honolulu", State: "Hawaii", Area: "Waikiki", Country: "USA"},
		MaxGuests: 4, PriceFrom: 220, Currency: "USD",
		Rating: 8.6, ReviewCount: 4943, PopularityScore: 4943,
		Amenities:    []string{"Pool", "Free Wifi", "Breakfast included"},
		ProviderName: "Expedia", BookingURL: "https://www.expedia.com/Honolulu-Hotels-Hyatt-Place-Waikiki-Beach.h2766.Hotel-Information",
		Images:        []string{"https://images.unsplash.com/photo-1600566753190-17f0baa2a6c3"},
		AvailableFrom: "2026-11-15",
	},
	{
		ID: "prop-013", Name: "Fairmont San Francisco", Type: "Hotel",
		Slug:      "fairmont-san-francisco",
		Location:  models.PropertyLocation{City: "San Francisco", State: "California", Area: "Nob Hill", Country: "USA"},
		MaxGuests: 2, PriceFrom: 520, Currency: "USD",
		Rating: 8.4, ReviewCount: 862, PopularityScore: 862,
		Amenities:    []string{"Spa and wellness center", "Fitness center", "Garden", "Parking"},
		ProviderName: "Booking.com", BookingURL: "https://www.booking.com/hotel/us/the-fairmont-san-francisco-san-francisco-california.html",
		Images:        []string{"https://images.unsplash.com/photo-1493809842364-78817add7ffb"},
		AvailableFrom: "2026-11-15",
	},
	{
		ID: "prop-014", Name: "Halepuna Waikiki by Halekulani", Type: "Hotel",
		Slug:      "halepuna-waikiki-by-halekulani",
		Location:  models.PropertyLocation{City: "Honolulu", State: "Hawaii", Area: "Waikiki", Country: "USA"},
		MaxGuests: 2, PriceFrom: 345, Currency: "USD",
		Rating: 9.4, ReviewCount: 2144, PopularityScore: 2144,
		Amenities:    []string{"Pool"},
		ProviderName: "Expedia", BookingURL: "https://www.expedia.com/Honolulu-Hotels-Halepuna-Waikiki.h40007622.Hotel-Information",
		Images:        []string{"https://images.unsplash.com/photo-1566073771259-6a8506099945"},
		AvailableFrom: "2026-11-15",
	},
	{
		ID: "prop-015", Name: "The Royal Hawaiian, A Luxury Collection Resort, Waikiki", Type: "Resort",
		Slug:      "the-royal-hawaiian-a-luxury-collection-resort-waikiki",
		Location:  models.PropertyLocation{City: "Honolulu", State: "Hawaii", Area: "Waikiki", Country: "USA"},
		MaxGuests: 6, PriceFrom: 690, Currency: "USD",
		Rating: 8.4, ReviewCount: 452, PopularityScore: 452,
		Amenities:    []string{"Pool", "Beachfront", "Fitness center", "Free Wifi"},
		ProviderName: "Booking.com", BookingURL: "https://www.booking.com/hotel/us/the-royal-hawaiian.html",
		Images:        []string{"https://images.unsplash.com/photo-1580587771525-78b9dba3b914"},
		AvailableFrom: "2026-11-15",
	},
}

// galleryExtraPhotoIDs stores four additional illustrative images
// for each seeded property.
//
// The original first image is preserved.
// These photographs are illustrative, not authenticated images
// of the named hotels.
var galleryExtraPhotoIDs = map[string][]string{
	"prop-001": {
		"photo-1631049307264-da0ec9d70304",
		"photo-1573052905904-34ad8c27f0cc",
		"photo-1590381105924-c72589b9ef3f",
		"photo-1600210492486-724fe5c67fb0",
	},
	"prop-002": {
		"photo-1507876466758-bc54f384809c",
		"photo-1509934053274-6ef492293340",
		"photo-1623055576799-e0644e49887a",
		"photo-1562122583-eacb4151baa3",
	},
	"prop-003": {
		"photo-1582719508461-905c673771fd",
		"photo-1611892440504-42a792e24d32",
		"photo-1542314831-068cd1dbfeeb",
		"photo-1599696848652-f0ff23bc911f",
	},
	"prop-004": {
		"photo-1561501900-3701fa6a0864",
		"photo-1656115883666-1a8c605de3bd",
		"photo-1696735648220-498fc4ee1655",
		"photo-1675302197721-9f36d27a406a",
	},
	"prop-005": {
		"photo-1540541338287-41700207dee6",
		"photo-1551882547-ff40c63fe5fa",
		"photo-1520250497591-112f2f40a3f4",
		"photo-1549294413-26f195200c16",
	},
	"prop-006": {
		"photo-1589817559780-02ce2a79a5be",
		"photo-1682922212654-cb6a7e158ce5",
		"photo-1638396243368-efd7781ded5b",
		"photo-1673138660624-d76ebf4410f9",
	},
	"prop-007": {
		"photo-1629140727571-9b5c6f6267b4",
		"photo-1621293954908-907159247fc8",
		"photo-1578683010236-d716f9a3f461",
		"photo-1560448204-e02f11c3d0e2",
	},
	"prop-008": {
		"photo-1755716963484-129960b33ca7",
		"photo-1540202404-a2f29016b523",
		"photo-1507525428034-b723cf961d3e",
		"photo-1510414842594-a61c69b5ae57",
	},
	"prop-009": {
		"photo-1610641818989-c2051b5e2cfd",
		"photo-1551918120-9739cb430c6d",
		"photo-1623718649591-311775a30c43",
		"photo-1586611292717-f828b167408c",
	},
	"prop-010": {
		"photo-1611048267451-e6ed903d4a38",
		"photo-1568495248636-6432b97bd949",
		"photo-1631049552057-403cdb8f0658",
		"photo-1590490360182-c33d57733427",
	},
	"prop-011": {
		"photo-1514282401047-d79a71a590e8",
		"photo-1525498128493-380d1990a112",
		"photo-1501785888041-af3ef285b470",
		"photo-1505693416388-ac5ce068fe85",
	},
	"prop-012": {
		"photo-1582719478250-c89cae4dc85b",
		"photo-1566665797739-1674de7a421a",
		"photo-1512918728675-ed5a9ecdebfd",
		"photo-1576354302919-96748cb8299e",
	},
	"prop-013": {
		"photo-1711059985570-4c32ed12a12c",
		"photo-1562438668-bcf0ca6578f0",
		"photo-1546967900-1bea5f16b69d",
		"photo-1584132967334-10e028bd69f7",
	},
	"prop-014": {
		"photo-1509934053274-6ef492293340",
		"photo-1540541338287-41700207dee6",
		"photo-1582719478250-c89cae4dc85b",
		"photo-1600607687920-4e2a09cf159d",
	},
	"prop-015": {
		"photo-1582719508461-905c673771fd",
		"photo-1540202404-a2f29016b523",
		"photo-1566073771259-6a8506099945",
		"photo-1600566752355-35792bedcfea",
	},
}

// gallerySeedProperties builds the five-image seed dataset
// without modifying the original seedProperties variable.
func gallerySeedProperties() ([]seedProperty, error) {
	if len(galleryExtraPhotoIDs) != len(seedProperties) {
		return nil, fmt.Errorf(
			"gallery entry count %d differs from seed count %d",
			len(galleryExtraPhotoIDs),
			len(seedProperties),
		)
	}

	result := make([]seedProperty, 0, len(seedProperties))
	visited := make(map[string]bool, len(seedProperties))

	for _, property := range seedProperties {
		if visited[property.ID] {
			return nil, fmt.Errorf(
				"duplicate seed ID %q", property.ID,
			)
		}
		visited[property.ID] = true

		ids, ok := galleryExtraPhotoIDs[property.ID]

		if !ok || len(ids) != 4 ||
			len(property.Images) != 1 ||
			strings.TrimSpace(property.Images[0]) == "" {
			return nil, fmt.Errorf(
				"%s: expected 1 original and 4 additional images",
				property.ID,
			)
		}

		images := append([]string(nil), property.Images...)
		seen := make(map[string]bool, 5)

		for _, id := range ids {
			if !strings.HasPrefix(id, "photo-") ||
				len(id) <= len("photo-") ||
				seen[id] ||
				strings.Contains(property.Images[0], id) {
				return nil, fmt.Errorf(
					"%s: invalid or duplicate gallery photo %q",
					property.ID, id,
				)
			}

			seen[id] = true

			images = append(
				images,
				"https://images.unsplash.com/"+id+
					"?auto=format&fit=crop&w=1000&q=75",
			)
		}

		property.Images = images
		result = append(result, property)
	}

	return result, nil
}

// additionalDemoSeedProperties creates 33 illustrative listings
// using the existing, validated five-image gallery properties.
//
// The original 15 properties are not modified. All demo listings
// retain valid locations, amenities, availability and five photos.
//
// These entries are assessment data, not real bookable properties.
func additionalDemoSeedProperties(
	originalGallery []seedProperty,
) ([]seedProperty, error) {
	if len(originalGallery) != 15 {
		return nil, fmt.Errorf(
			"expected 15 original gallery properties, got %d",
			len(originalGallery),
		)
	}

	// Names are ordered to match the corresponding source
	// property's location, repeating the 15-location pattern.
	names := []string{
		"Midtown Garden Suites",
		"Waikiki Ocean View Villas",
		"Desert Lantern Resort",
		"Waikiki Palm Courtyard",
		"Miami Seabreeze Retreat",
		"Pacific Terrace Suites",
		"Beverly Grove Hideaway",
		"Waikiki Island Nook",
		"Orlando Magnolia Residence",
		"Honolulu Sunset Stay",
		"Coronado Sandstone Inn",
		"Waikiki Morning Villas",
		"Nob Hill Skyline Suites",
		"Waikiki Moonlight Retreat",
		"Royal Waikiki Garden Suites",
		"Midtown Art Deco Suites",
		"Waikiki Coral Grove",
		"Las Vegas Sunset Courtyard",
		"Waikiki Beach Garden Stay",
		"Miami Palm Shore Retreat",
		"Honolulu Bluewater Villas",
		"Beverly Hills Canyon Residence",
		"Waikiki Lagoon Loft",
		"Orlando Lakeside Suites",
		"Honolulu Oceanfront Nest",
		"Coronado Coastal Cottage",
		"Waikiki Island Breeze Haven",
		"Nob Hill Bayview Retreat",
		"Waikiki Sunset Terrace",
		"Waikiki Royal Palm House",
		"Midtown Central Residence",
		"Waikiki Morning Breeze Suites",
		"Desert Mirage Poolside Stay",
	}

	demos := make([]seedProperty, 0, len(names))

	for i, name := range names {
		source := originalGallery[i%len(originalGallery)]

		if len(source.Images) != 5 {
			return nil, fmt.Errorf(
				"%s: expected five source gallery images",
				source.ID,
			)
		}

		// Copy the source's valid location, availability,
		// property type, capacity and other seed fields.
		demo := source

		// Allocate stable, nonconflicting property identities.
		demo.ID = fmt.Sprintf("prop-%03d", i+16)
		demo.Name = name + " (Demo)"
		demo.Slug = "demo-" + strings.ToLower(
			strings.ReplaceAll(name, " ", "-"),
		)

		// Point to the original location-matched hotel's real provider page.
		// The demo remains illustrative and is NOT that hotel.
		demo.ProviderName = source.ProviderName
		demo.BookingURL = source.BookingURL

		// Illustrative assessment metrics.
		demo.Rating = 8.0 + float64(i%11)/10
		demo.ReviewCount = 80 + i*13
		demo.PopularityScore = demo.ReviewCount

		// Clone slices so original gallery properties
		// cannot be accidentally modified through demos.
		demo.Images = append(
			[]string(nil), source.Images...,
		)
		demo.Amenities = append(
			[]string(nil), source.Amenities...,
		)

		demos = append(demos, demo)
	}

	return demos, nil
}
