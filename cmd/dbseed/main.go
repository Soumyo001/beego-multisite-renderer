package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"beego-multisite-renderer/models"
	"beego-multisite-renderer/services"

	"github.com/beego/beego/v2/client/orm"
	"github.com/joho/godotenv"
)

const siteKey = "beta/preview/test-travel-templates"
const availabilityEnd = "2027-12-31"

type seedProperty struct {
	ID              string
	Slug            string
	Name            string
	Type            string
	Location        models.PropertyLocation
	MaxGuests       int
	SizeSqFt        int
	Rating          float64
	ReviewCount     int
	PriceFrom       int
	Currency        string
	Amenities       []string
	ProviderName    string
	BookingURL      string
	PopularityScore int
	EcoFriendly     bool
	Images          []string
	AvailableFrom   string
}

type seedStats struct {
	Locations  int
	Properties int
	Windows    int
}

func main() {
	verifyImages := flag.Bool(
		"verify-images",
		false,
		"check gallery URLs without accessing PostgreSQL",
	)
	syncImages := flag.Bool(
		"sync-images",
		false,
		"preview image updates for existing properties",
	)
	syncDetails := flag.Bool(
		"sync-details", false, "preview property-details backfill",
	)
	syncLinks := flag.Bool(
		"sync-links", false, "preview demo alternative-provider links",
	)
	seedInternational := flag.Bool(
		"seed-international", false,
		"preview adding the 24 Toronto and Dhaka demo listings",
	)
	apply := flag.Bool(
		"apply", false, "commit the selected sync mode",
	)
	flag.Parse()

	modes := 0
	for _, enabled := range []bool{*syncImages, *syncDetails, *syncLinks, *seedInternational} {
		if enabled {
			modes++
		}
	}
	if *apply && modes != 1 {
		log.Fatal("-apply requires exactly one sync mode")
	}
	if modes > 1 || (*verifyImages && modes > 0) {
		log.Fatal("sync modes and -verify-images are mutually exclusive")
	}

	properties, err := gallerySeedProperties()
	if err != nil {
		log.Fatalf("prepare gallery seed data: %v", err)
	}

	if err := validateSeedData(properties); err != nil {
		log.Fatalf("invalid gallery seed data: %v", err)
	}

	// URL verification never initializes or accesses PostgreSQL.
	if *verifyImages {
		if err := verifyGalleryURLs(); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := godotenv.Load(); err != nil &&
		!errors.Is(err, os.ErrNotExist) {
		log.Fatalf("load .env: %v", err)
	}

	databaseURL := os.Getenv("DATABASE_URL")

	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL_POOLED")
	}

	if err := services.InitPropertyDatabase(databaseURL); err != nil {
		log.Fatalf("initialize PostgreSQL: %v", err)
	}

	if *syncImages {
		if err := syncExistingGalleryImages(
			properties, *apply,
		); err != nil {
			log.Fatalf(
				"image sync failed; transaction rolled back: %v",
				err,
			)
		}
		return
	}

	// LD-1A: Add the illustrative demonstration properties
	// only during normal database seeding.
	//
	// The existing -verify-images and -sync-images modes
	// have already returned before reaching this point.
	demos, err := additionalDemoSeedProperties(properties)
	if err != nil {
		log.Fatalf(
			"prepare additional demo properties: %v",
			err,
		)
	}

	// Toronto and Dhaka listings for the home-page tiles.
	international, err := internationalDemoSeedProperties()
	if err != nil {
		log.Fatalf("prepare international demo properties: %v", err)
	}

	properties = append(properties, demos...)
	properties = append(properties, international...)

	// Validate the complete 72-property dataset before
	// starting any PostgreSQL transaction.
	if err := validateSeedData(properties); err != nil {
		log.Fatalf(
			"invalid expanded seed data: %v",
			err,
		)
	}

	// Provider-link synchronization is a separate dry-run-first operation.
	if *syncLinks {
		if err := syncDemoProviderLinks(demos, len(properties), *apply); err != nil {
			log.Fatalf("demo-link synchronization failed: %v", err)
		}
		return
	}

	// PD-1B.2: details backfill is separate from normal seeding.
	// Without -apply, this operation only previews changes.
	if *syncDetails {
		if err := syncPropertyDetails(
			properties, *apply,
		); err != nil {
			log.Fatalf(
				"property-details backfill failed; no changes committed: %v",
				err,
			)
		}
		return
	}

	// Adding the 24 international rows is dry-run-first, like the
	// other modes. Normal seeding below also creates them.
	if *seedInternational {
		if err := seedInternationalProperties(
			international, 48, *apply,
		); err != nil {
			log.Fatalf("international seeding failed; no changes committed: %v", err)
		}
		return
	}

	stats, err := insertSeedProperties(properties)
	if err != nil {
		log.Fatalf("seed transaction rolled back: %v", err)
	}

	log.Printf(
		"Seed complete. New records: locations=%d properties=%d availability=%d",
		stats.Locations, stats.Properties, stats.Windows,
	)
	log.Println(
		"Existing rows were preserved; rerunning does not overwrite modified property data.",
	)
}

// insertSeedProperties creates any missing location, property and
// availability rows in one transaction. Existing rows are read, never
// overwritten, so rerunning it is safe.
func insertSeedProperties(properties []seedProperty) (seedStats, error) {
	var stats seedStats

	err := orm.NewOrm().DoTx(
		func(ctx context.Context, tx orm.TxOrmer) error {
			knownLocations := make(
				map[models.PropertyLocation]*models.LocationRecord,
			)

			end, err := time.Parse("2006-01-02", availabilityEnd)
			if err != nil {
				return err
			}

			for _, p := range properties {
				location := knownLocations[p.Location]

				if location == nil {
					candidate := &models.LocationRecord{
						City:    p.Location.City,
						State:   p.Location.State,
						Area:    p.Location.Area,
						Country: p.Location.Country,
					}

					isNew, locationID, err := tx.ReadOrCreate(
						candidate,
						"Country", "State", "City", "Area",
					)
					if err != nil {
						return fmt.Errorf(
							"seed location for %s: %w", p.ID, err,
						)
					}

					candidate.Id = locationID
					if isNew {
						stats.Locations++
					}

					knownLocations[p.Location] = candidate
					location = candidate
				}

				amenities, err := json.Marshal(p.Amenities)
				if err != nil {
					return fmt.Errorf(
						"encode amenities for %s: %w", p.ID, err,
					)
				}

				images, err := json.Marshal(p.Images)
				if err != nil {
					return fmt.Errorf(
						"encode images for %s: %w", p.ID, err,
					)
				}

				detailsSeed, err := propertyDetailsForSeed(p)
				if err != nil {
					return fmt.Errorf(
						"prepare details for %s: %w",
						p.ID, err,
					)
				}

				record := &models.PropertyRecord{
					SiteKey:         siteKey,
					Slug:            p.Slug,
					Location:        location,
					PropertyID:      p.ID,
					Name:            p.Name,
					PropertyType:    p.Type,
					MaxGuests:       p.MaxGuests,
					SizeSqFt:        p.SizeSqFt,
					Rating:          p.Rating,
					ReviewCount:     p.ReviewCount,
					PriceFrom:       p.PriceFrom,
					Currency:        p.Currency,
					AmenitiesJSON:   string(amenities),
					ProviderName:    p.ProviderName,
					BookingURL:      p.BookingURL,
					PopularityScore: p.PopularityScore,
					EcoFriendly:     p.EcoFriendly,
					ImageJSON:       string(images),
					Details:         detailsSeed.Details,
					Bedrooms:        detailsSeed.Bedrooms,
					Bathrooms:       detailsSeed.Bathrooms,
					PetFriendly:     detailsSeed.PetFriendly,
				}

				isNew, propertyID, err := tx.ReadOrCreate(
					record, "SiteKey", "PropertyID",
				)
				if err != nil {
					return fmt.Errorf(
						"seed property %s: %w", p.ID, err,
					)
				}

				record.Id = propertyID
				if isNew {
					stats.Properties++
				}

				start, err := time.Parse(
					"2006-01-02", p.AvailableFrom,
				)
				if err != nil {
					return fmt.Errorf(
						"parse availability for %s: %w", p.ID, err,
					)
				}

				window := &models.AvailabilityRecord{
					Property:  record,
					StartDate: start,
					EndDate:   end,
				}

				isNew, _, err = tx.ReadOrCreate(
					window,
					"Property", "StartDate", "EndDate",
				)
				if err != nil {
					return fmt.Errorf(
						"seed availability for %s: %w", p.ID, err,
					)
				}

				if isNew {
					stats.Windows++
				}
			}

			return nil
		},
	)

	return stats, err
}

// seedInternationalProperties previews (default) or inserts the 24
// Toronto and Dhaka demo rows. It refuses to run when the publisher
// has rows it does not expect, or when an ID or slug is already used
// by a different property.
func seedInternationalProperties(
	international []seedProperty, existingBase int, apply bool,
) error {
	db, err := orm.GetDB("default")
	if err != nil {
		return fmt.Errorf("get database: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var total int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM public.properties WHERE site_key = $1`, siteKey,
	).Scan(&total); err != nil {
		return fmt.Errorf("count publisher properties: %w", err)
	}

	pending, unchanged := 0, 0
	for _, p := range international {
		var id, slug, name string
		err := db.QueryRowContext(ctx, `
			SELECT property_id, slug, name FROM public.properties
			WHERE site_key = $1 AND (property_id = $2 OR slug = $3)
			LIMIT 1`,
			siteKey, p.ID, p.Slug,
		).Scan(&id, &slug, &name)

		switch {
		case errors.Is(err, sql.ErrNoRows):
			pending++
			log.Printf("%s: new %s (%s)", p.ID, p.Name, p.Location.City)
		case err != nil:
			return fmt.Errorf("check %s: %w", p.ID, err)
		case id == p.ID && slug == p.Slug && name == p.Name:
			unchanged++
		default:
			return fmt.Errorf(
				"%s or slug %s is already used by %s (%s); refusing",
				p.ID, p.Slug, id, name,
			)
		}
	}

	if total != existingBase+unchanged {
		return fmt.Errorf(
			"publisher has %d properties; expected %d; refusing",
			total, existingBase+unchanged,
		)
	}

	if !apply {
		log.Printf(
			"DRY RUN: %d would be added, %d already present; no changes made",
			pending, unchanged,
		)
		return nil
	}

	stats, err := insertSeedProperties(international)
	if err != nil {
		return err
	}
	log.Printf(
		"COMMITTED: properties=%d locations=%d availability=%d added; %d already present",
		stats.Properties, stats.Locations, stats.Windows, unchanged,
	)
	return nil
}

// syncDemoProviderLinks updates only the two provider columns of the 33
// pre-existing, publisher-scoped demo rows. Dry-run is the default.
func syncDemoProviderLinks(demos []seedProperty, expectedTotal int, apply bool) error {
	if len(demos) != 33 {
		return fmt.Errorf("expected 33 demo properties, got %d", len(demos))
	}

	db, err := orm.GetDB("default")
	if err != nil {
		return fmt.Errorf("get database: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin redirect transaction: %w", err)
	}
	defer tx.Rollback()

	var total int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM public.properties WHERE site_key = $1`, siteKey,
	).Scan(&total); err != nil {
		return fmt.Errorf("count publisher properties: %w", err)
	}
	if total != expectedTotal {
		return fmt.Errorf("publisher has %d properties; expected %d; refusing update", total, expectedTotal)
	}

	pending, unchanged := 0, 0
	for _, p := range demos {
		if !strings.HasPrefix(p.Slug, "demo-") ||
			!strings.HasSuffix(p.Name, " (Demo)") {
			return fmt.Errorf("refusing non-demo identity %s", p.ID)
		}

		query := `SELECT slug, name, provider_name, booking_url
			FROM public.properties WHERE site_key = $1 AND property_id = $2`
		if apply {
			query += " FOR UPDATE"
		}
		var slug, name, provider, booking string
		if err := tx.QueryRowContext(ctx, query, siteKey, p.ID).Scan(
			&slug, &name, &provider, &booking,
		); err != nil {
			return fmt.Errorf("read demo %s: %w", p.ID, err)
		}
		if slug != p.Slug || name != p.Name {
			return fmt.Errorf("demo %s changed slug or name; refusing update", p.ID)
		}
		if provider == p.ProviderName && booking == p.BookingURL {
			unchanged++
			continue
		}
		placeholder := "https://example.com/demo/" + p.ID
		if provider != "Demo Listing" || booking != placeholder {
			return fmt.Errorf("demo %s has a custom provider/link; refusing overwrite", p.ID)
		}
		pending++
		if !apply {
			continue
		}
		result, err := tx.ExecContext(ctx, `
			UPDATE public.properties
			SET provider_name = $1, booking_url = $2
			WHERE site_key = $3 AND property_id = $4
			  AND slug = $5 AND name = $6
			  AND provider_name = 'Demo Listing'
			  AND booking_url = $7`,
			p.ProviderName, p.BookingURL, siteKey, p.ID, p.Slug, p.Name, placeholder,
		)
		if err != nil {
			return fmt.Errorf("update demo %s: %w", p.ID, err)
		}
		rows, err := result.RowsAffected()
		if err != nil || rows != 1 {
			return fmt.Errorf("update demo %s affected %d rows: %v", p.ID, rows, err)
		}
	}
	if !apply {
		log.Printf("DRY RUN: %d demo links pending, %d already matched; no changes made", pending, unchanged)
		return nil
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit demo links: %w", err)
	}
	log.Printf("COMMITTED: %d demo links updated, %d already matched", pending, unchanged)
	return nil
}

func validateSeedData(properties []seedProperty) error {
	if len(properties) == 0 {
		return errors.New("no properties supplied")
	}

	ids := make(map[string]bool, len(properties))
	slugs := make(map[string]bool, len(properties))

	end, err := time.Parse("2006-01-02", availabilityEnd)
	if err != nil {
		return fmt.Errorf("invalid common availability end: %w", err)
	}

	for _, p := range properties {
		if p.ID == "" || ids[p.ID] {
			return fmt.Errorf(
				"missing or duplicate property ID %q", p.ID,
			)
		}
		ids[p.ID] = true

		if !validSlug(p.Slug) || slugs[p.Slug] {
			return fmt.Errorf(
				"invalid or duplicate property slug %q for %s",
				p.Slug, p.ID,
			)
		}
		slugs[p.Slug] = true

		if p.Name == "" ||
			p.Type == "" ||
			p.Location.City == "" ||
			p.Location.State == "" ||
			p.Location.Country == "" ||
			p.Location.Area == "" {
			return fmt.Errorf(
				"missing identity or location for %s", p.ID,
			)
		}

		if p.MaxGuests < 1 ||
			p.PriceFrom < 1 ||
			p.Currency == "" {
			return fmt.Errorf(
				"invalid capacity or price for %s", p.ID,
			)
		}

		if p.ProviderName == "" ||
			p.BookingURL == "" ||
			len(p.Images) == 0 {
			return fmt.Errorf(
				"missing provider or image for %s", p.ID,
			)
		}

		if len(p.Amenities) == 0 {
			return fmt.Errorf(
				"missing amenities for %s", p.ID,
			)
		}

		start, err := time.Parse(
			"2006-01-02", p.AvailableFrom,
		)
		if err != nil || !end.After(start) {
			return fmt.Errorf(
				"invalid availability range for %s", p.ID,
			)
		}
	}

	return nil
}

// validSlug keeps stable property slugs lowercase and URL-safe.
func validSlug(slug string) bool {
	if slug == "" ||
		len(slug) > 255 ||
		slug[0] == '-' ||
		slug[len(slug)-1] == '-' {
		return false
	}

	previousHyphen := false

	for _, r := range slug {
		switch {
		case r >= 'a' && r <= 'z',
			r >= '0' && r <= '9':
			previousHyphen = false

		case r == '-' && !previousHyphen:
			previousHyphen = true

		default:
			return false
		}
	}

	return true
}

// verifyGalleryURLs checks each distinct supplemental image
// using a small HTTP GET request rather than HEAD.
//
// It does not connect to PostgreSQL and does not change data.
func verifyGalleryURLs() error {
	unique := make(map[string]bool)

	for _, ids := range galleryExtraPhotoIDs {
		for _, id := range ids {
			unique[id] = true
		}
	}

	ids := make([]string, 0, len(unique))
	for id := range unique {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	failures := 0

	for _, id := range ids {
		target := "https://images.unsplash.com/" + id +
			"?auto=format&fit=crop&w=200&q=35"

		response, err := client.Get(target)
		if err != nil {
			log.Printf("FAIL %s: %v", id, err)
			failures++
			continue
		}

		contentType := response.Header.Get("Content-Type")
		status := response.StatusCode

		// Confirm that the server sends actual image data.
		chunk, readErr := io.ReadAll(
			io.LimitReader(response.Body, 1024),
		)
		response.Body.Close()

		if status != http.StatusOK ||
			!strings.HasPrefix(contentType, "image/") ||
			readErr != nil ||
			len(chunk) == 0 {

			log.Printf(
				"FAIL %s: HTTP %d, type %q, read error %v",
				id, status, contentType, readErr,
			)

			failures++
		}
	}

	if failures != 0 {
		return fmt.Errorf(
			"%d of %d distinct image links failed verification; do not apply",
			failures, len(ids),
		)
	}

	log.Printf(
		"Image verification PASSED: %d distinct URLs",
		len(ids),
	)
	return nil
}

// syncExistingGalleryImages changes only the images JSONB field
// for the explicitly seeded publisher properties.
//
// The operation is transactional and refuses unexpected existing
// image data rather than silently overwriting it.
func syncExistingGalleryImages(
	properties []seedProperty,
	apply bool,
) error {
	pending := 0
	unchanged := 0

	err := orm.NewOrm().DoTx(
		func(ctx context.Context, tx orm.TxOrmer) error {
			for _, property := range properties {
				record := &models.PropertyRecord{
					SiteKey:    siteKey,
					PropertyID: property.ID,
				}

				if err := tx.Read(
					record, "SiteKey", "PropertyID",
				); err != nil {
					return fmt.Errorf(
						"read property %s: %w",
						property.ID, err,
					)
				}

				var current []string

				if err := json.Unmarshal(
					[]byte(record.ImageJSON), &current,
				); err != nil {
					return fmt.Errorf(
						"decode images for %s: %w",
						property.ID, err,
					)
				}

				// A previously completed update is safe to skip.
				if slices.Equal(current, property.Images) {
					unchanged++
					continue
				}

				// Only the expected one-image seed can be updated.
				if len(current) != 1 ||
					current[0] != property.Images[0] {
					return fmt.Errorf(
						"%s has unexpected image data (%d entries); refusing overwrite",
						property.ID, len(current),
					)
				}

				pending++

				log.Printf(
					"%s: 1 -> 5 images",
					property.ID,
				)

				// Dry run does not modify the record.
				if !apply {
					continue
				}

				encoded, err := json.Marshal(property.Images)
				if err != nil {
					return fmt.Errorf(
						"encode images for %s: %w",
						property.ID, err,
					)
				}

				record.ImageJSON = string(encoded)

				affected, err := tx.Update(
					record, "ImageJSON",
				)

				if err != nil || affected != 1 {
					return fmt.Errorf(
						"update %s affected %d rows: %v",
						property.ID, affected, err,
					)
				}
			}

			return nil
		},
	)

	if err != nil {
		return err
	}

	if apply {
		log.Printf(
			"COMMITTED: %d updated, %d already matched",
			pending, unchanged,
		)
	} else {
		log.Printf(
			"DRY RUN: %d would update, %d already matched; no changes made",
			pending, unchanged,
		)
	}

	return nil
}
