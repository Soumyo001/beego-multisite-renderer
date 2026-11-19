package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// syncPropertyDetails is dry-run by default.
// Applying changes only the four new details columns.
func syncPropertyDetails(
	properties []seedProperty,
	apply bool,
) error {
	db, err := orm.GetDB("default")
	if err != nil {
		return fmt.Errorf("get database: %w", err)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(), 60*time.Second,
	)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(
			"begin details transaction: %w", err,
		)
	}
	defer tx.Rollback()

	// Refuse to operate on an unexpected publisher dataset.
	var count int
	if err := tx.QueryRowContext(
		ctx,
		`SELECT COUNT(*)
		 FROM public.properties
		 WHERE site_key = $1`,
		siteKey,
	).Scan(&count); err != nil {
		return fmt.Errorf(
			"count publisher properties: %w", err,
		)
	}

	if count != len(properties) {
		return fmt.Errorf(
			"publisher has %d properties; expected exactly %d; refusing backfill",
			count, len(properties),
		)
	}

	pending, unchanged := 0, 0

	for _, p := range properties {
		expected, err := propertyDetailsForSeed(p)
		if err != nil {
			return err
		}

		query := `
			SELECT slug, name,
			       details, bedrooms, bathrooms, pet_friendly
			FROM public.properties
			WHERE site_key = $1 AND property_id = $2
		`

		if apply {
			query += " FOR UPDATE"
		}

		var current propertyDetailsSeed
		var currentSlug, currentName string

		err = tx.QueryRowContext(
			ctx, query, siteKey, p.ID,
		).Scan(
			&currentSlug,
			&currentName,
			&current.Details,
			&current.Bedrooms,
			&current.Bathrooms,
			&current.PetFriendly,
		)

		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf(
				"missing property %s for publisher %s",
				p.ID, siteKey,
			)
		}
		if err != nil {
			return fmt.Errorf(
				"read details for %s: %w", p.ID, err,
			)
		}

		// Do not generate descriptions from stale seed identities.
		if currentSlug != p.Slug ||
			currentName != p.Name {
			return fmt.Errorf(
				"property %s has changed slug or name; refusing backfill",
				p.ID,
			)
		}

		switch classifyPropertyDetails(current, expected) {
		case "unchanged":
			unchanged++

		case "pending":
			pending++

			if !apply {
				continue
			}

			result, err := tx.ExecContext(ctx, `
				UPDATE public.properties
				SET details = $1,
				    bedrooms = $2,
				    bathrooms = $3,
				    pet_friendly = $4
				WHERE site_key = $5
				  AND property_id = $6
				  AND details IS NULL
				  AND bedrooms IS NULL
				  AND bathrooms IS NULL
				  AND pet_friendly IS NULL
			`,
				expected.Details,
				expected.Bedrooms,
				expected.Bathrooms,
				expected.PetFriendly,
				siteKey,
				p.ID,
			)

			if err != nil {
				return fmt.Errorf(
					"update details for %s: %w",
					p.ID, err,
				)
			}

			changed, err := result.RowsAffected()
			if err != nil || changed != 1 {
				return fmt.Errorf(
					"update details for %s affected %d rows: %v",
					p.ID, changed, err,
				)
			}

		default:
			return fmt.Errorf(
				"property %s already has different or partially populated details; refusing overwrite",
				p.ID,
			)
		}
	}

	if !apply {
		log.Printf(
			"DRY RUN: %d would update, %d already matched; no changes made",
			pending, unchanged,
		)
		return nil
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(
			"commit details backfill: %w", err,
		)
	}

	log.Printf(
		"COMMITTED: %d updated, %d already matched",
		pending, unchanged,
	)
	return nil
}
