package routers

import (
	"beego-multisite-renderer/controllers"
	"beego-multisite-renderer/services"

	beego "github.com/beego/beego/v2/server/web"
)

// Register keeps the existing server-rendered routes.
func Register(
	catalog *services.Catalog,
	loader *services.ContentLoader,
) {
	controllers.SetDependencies(catalog, loader)

	beego.Router("/", &controllers.PageController{})

	beego.Router(
		"/refine",
		&controllers.RefineController{},
		"get:Get",
	)

	// Infrastructure health route.
	// This does not resolve the request Host.
	beego.Router(
		"/health",
		&controllers.HealthController{},
		"get:Get",
	)

	// country page SSR route
	beego.Router(
		"/all/:country",
		&controllers.CountryController{},
		"get:Get",
	)

	// Property details must be registered before
	// the publisher's catch-all page route.
	beego.Router(
		"/property/:slug/:propertyID",
		&controllers.PropertyDetailsController{},
		"get:Get",
	)

	// Preserve the existing publisher page routing.
	beego.Router("/*", &controllers.PageController{})
}

// RegisterPropertyAPI adds the property-search endpoint.
// It is separate from Register so the existing SSR
// registration interface remains unchanged.
func RegisterPropertyAPI(
	catalog *services.Catalog,
	search services.PropertySearcher,
) {
	controllers.SetPropertyAPIDependencies(
		catalog,
		search,
	)
	controllers.SetCountrySearchDependencies(search)

	// The production database service implements both interfaces.
	// Existing listing-only test fixtures remain compatible.
	if reader, ok := search.(services.PropertyDetailsReader); ok {
		controllers.SetPropertyDetailsDependencies(reader, search)
	}

	// Home-page tile blocks (tiles_component) use the same service.
	if tiles, ok := search.(services.PropertyTilesSearcher); ok {
		controllers.SetTilesDependencies(tiles)
	}

	beego.Router(
		"/api/properties",
		&controllers.PropertyAPIController{},
		"get:Get",
	)
}
