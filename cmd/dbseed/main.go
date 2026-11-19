package main

import (
	"context"
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

	"sparrow-ssr/models"
	"sparrow-ssr/services"

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
	apply := flag.Bool(
		"apply",
		false,
		"commit image updates (requires -sync-images)",
	)
	flag.Parse()

	if *apply && !*syncImages {
		log.Fatal("-apply requires -sync-images")
	}

	if *verifyImages && *syncImages {
		log.Fatal(
			"-verify-images cannot be combined with -sync-images",
		)
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

	properties = append(properties, demos...)

	// Validate the complete 48-property dataset before
	// starting the PostgreSQL transaction.
	if err := validateSeedData(properties); err != nil {
		log.Fatalf(
			"invalid expanded seed data: %v",
			err,
		)
	}

	var stats seedStats

	err = orm.NewOrm().DoTx(
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
