# Beego Multisite Renderer

**Deployment-metadata-driven, multi-site server-side rendering in Go and Beego.**

Beego Multisite Renderer converts publisher-managed JSON content into HTML using Beego MVC templates. It resolves requests through a startup catalog keyed by **request Host + canonical URL path**, combines shared header/page/footer documents, and caches decoded JSON in memory. Accommodation listings are stored separately in PostgreSQL: country pages render property cards on the server, while filtered search results use a JSON API and client-side rendering.

The JSON content is read from local files under `content/` using the publisher storage-path structure. The application does **not** fetch publisher JSON directly from S3 during a request; external CSS, JavaScript, and permitted image URLs referenced by the content remain external resources.

## 1. Features and technology

| Area | Implementation |
| --- | --- |
| Runtime | Go 1.26 (`go.mod` specifies `go 1.26.0`) |
| One-command local run | `run.sh`: `.env` from `.env.example`, PostgreSQL 17 in Docker, schema, demo data, `bee run` |
| Web framework | Beego v2.1.0, MVC controllers and Go HTML templates |
| Publisher routing | Startup `Catalog` indexed by normalized Host + canonical path |
| Multiple publishers | `NewMultiCatalog`, one deployment manifest per publisher |
| JSON loading | Concurrent reads on cache misses; typed decoded-content caches; single-flight suppression |
| Property data | PostgreSQL (Neon-compatible) via Beego ORM and pgx v5 |
| SQL tables | `locations`, `properties`, `property_availability` |
| Property isolation | Stable `site_key`; scoped unique property identifiers and slugs |
| Country listings | `/all/:country`: server-rendered cards, eight per page |
| Filtered results | `/refine`: JavaScript rendering from `/api/properties`, eight per page |
| Home page tiles | `tiles_component` blocks filled from PostgreSQL through template markers written once into the page JSON by `scripts/tilemarkers` |
| Interactions | Homepage search, date/guest selection, filters, sorting, pagination, loading skeletons, keyboard submission |
| Diagnostics | Go tests, race detector, benchmarks, `pprof`, read-only opt-in database integration tests |

## 2. Architecture and request flow

```text
                     HTTP request (Host + path)
                                 |
                           Beego router
                                 |
               +-----------------+------------------+
               |                 |                  |
        Publisher page      /all/:country          /refine
               |                 |                  |
       Catalog.Resolve      Catalog.Resolve      Shared template
               |                 |                  |
       ContentLoader      SearchCountry()        refine.js
         /   |   
     header page footer      PostgreSQL           /api/properties
         \   |   /                |                  |
       Cached decoded      Go renders cards     PostgreSQL search
          JSON                  |                  |
               +---------------+------------------+
                               |
                       HTML / browser
```

### Publisher routing

1. `services.NewCatalog` reads deployment metadata containing `data.published_domain` and `data.uploaded_file_paths`.
2. It resolves CDN object keys to local files under `content/`, identifies shared `header.json` and `footer.json`, and reads `head.canonical_url` from routable pages.
3. `services.NewMultiCatalog` merges independent publisher catalogs, rejecting colliding Host + path entries.
4. `controllers.PageController` resolves the **actual request Host and path**. Unknown hosts or paths return HTTP 404. For local runs, `LOCAL_SITE_HOST` makes requests addressed to this machine (`localhost`, `127.0.0.1`, `::1`) resolve as that publisher host (see section 4.4).
5. The loader combines shared header, page document, and footer. Publisher HTML and referenced CSS/JS are rendered by `views/page.tpl`.

Routes are derived from the deployment metadata and canonical URLs—not inferred from storage filenames or hardcoded domain mappings. `links.json` is not treated as a page. Publisher HTML is treated as trusted source content and must not be populated from arbitrary visitor input.

### Property results

- **`/all/:country` (SSR):** Go queries the database with `SearchCountry()`, determines the requested eight-card page, and renders those cards into the initial `views/refine.tpl` HTML. It does not fetch `/api/properties` to populate country cards.
- **`/refine` (CSR):** JavaScript fetches `/api/properties` for destination, guest, and date constraints. Price, Eco, amenity filters, sorting, and eight-card pagination update the results in the browser.
- **Filter handoff:** Filter Search from a country page navigates to `/refine?` with the selected parameters. The country page itself keeps server-rendered property content.
- **Interaction feedback:** Skeleton cards appear during result fetching, filtering, sorting, pagination, and the homepage Search transition. Header alignment and responsive styles retain the publisher layout.
- **`/health`:** A Host-independent health route; a successful response by itself does not verify database connectivity or rendered property results.

## 3. Project structure

```text
beego-multisite-renderer/
├── run.sh                       # One-command local setup and run
├── main.go                      # Startup, catalog, loader, database, listeners
├── go.mod / go.sum
├── .env.example                 # Environment-variable template (copied to .env by run.sh)
├── conf/
│   ├── app.conf
│   └── deployment.json          # Publisher deployment metadata
├── cmd/
│   ├── dbsetup/                 # Fresh database schema setup
│   └── dbseed/                  # Initial property/availability records
├── scripts/
│   └── tilemarkers/             # One-off: adds tile markers to a page JSON (not part of the server)
├── models/                     # Publisher JSON, rendering and ORM models
├── services/
│   ├── catalog.go
│   ├── multi_catalog.go
│   ├── content_loader.go
│   ├── file_cache.go
│   ├── property_database.go
│   ├── property_db_search.go
│   ├── property_country.go
│   ├── property_filters.go
│   ├── property_details.go
│   ├── property_tiles.go       # tiles_component database query
│   ├── tiles_html.go           # goquery alternative: replace tile blocks per request (kept for comparison)
│   └── *_test.go
├── controllers/
│   ├── page.go
│   ├── home_tiles.go           # tiles_component → property cards; renders the marked page body
│   ├── country.go
│   ├── refine.go
│   ├── property_details.go
│   ├── property_api.go
│   └── health.go
├── routers/router.go
├── views/
│   ├── page.tpl
│   ├── refine.tpl
│   └── property.tpl
├── static/
│   ├── css/                    # refine.css, home_tiles.css, property_details.css, search popup CSS
│   ├── js/                     # refine.js, listing_cards.js, home_tiles.js, property_details.js, search popup
│   └── img/
├── content/                    # Local publisher JSON storage hierarchy
└── tests/                      # HTTP integration tests and benchmarks
```

## 4. Setup and running locally

### 4.1 Quick start: one command

**Requirements**

- **Go 1.26 or newer**: <https://go.dev/dl/>
- **Docker**, running, and usable by your user without `sudo` (the user is in the `docker` group)
- **Bee CLI v2**: `go install github.com/beego/bee/v2@latest` (make sure `$(go env GOPATH)/bin` is on your `PATH`)
- Internet access on the first run (Go modules and the `postgres:17` image are downloaded)

**Run**

```bash
git clone https://github.com/Soumyo001/beego-multisite-renderer.git
cd beego-multisite-renderer
./run.sh
```

Then open **<http://localhost:8080/>**. Press `Ctrl+C` to stop the server.

`run.sh` performs these steps:

| Step | What happens |
| --- | --- |
| 1. `.env` | Copies `.env.example` to `.env` if `.env` does not exist yet. An existing `.env` is kept. |
| 2. Database | If `DATABASE_URL` in `.env` is empty, starts a PostgreSQL 17 container named `bmr-postgres` on `127.0.0.1:55432` with a **randomly generated password**, and writes its connection URL into `.env`. No credentials are stored in the repository. If `DATABASE_URL` points to another database, Docker is not used. |
| 3. Schema | `go run ./cmd/dbsetup` creates the tables, indexes and foreign keys. |
| 4. Demo data | `go run ./cmd/dbseed` inserts the demo dataset. |
| 5. Server | `bee run` builds and starts the application on `127.0.0.1:8080`. |

A first run ends like this:

```text
==> Created .env from .env.example
==> Creating PostgreSQL container bmr-postgres (first run downloads the image)
==> Waiting for PostgreSQL
==> Creating tables
... PostgreSQL schema ready: tables, indexes, and foreign keys
==> Inserting demo data
... Seed complete. New records: locations=24 properties=72 availability=72
==> Starting: open http://localhost:8080/  (Ctrl+C to stop)
... SUCCESS  ▶ 0004 Built Successfully!
```

**Running it again** is safe: the existing container is started again, existing tables and rows are kept, and the seed reports `properties=0` new records.

**Stopping and removing the database**

```bash
docker stop bmr-postgres          # stop it; ./run.sh starts it again with the data kept
docker rm -f -v bmr-postgres      # remove it and its data; the next ./run.sh creates a fresh one
```

### 4.2 Pages to try

| URL | What it shows |
| --- | --- |
| <http://localhost:8080/> | Publisher home page; the Dhaka and Toronto tile blocks are filled from the database (`tiles_component`) |
| <http://localhost:8080/all/usa> | 48 USA properties, rendered on the server 8 at a time; more load while scrolling |
| <http://localhost:8080/all/canada>, <http://localhost:8080/all/bangladesh> | 12 properties each (Toronto, Dhaka) |
| <http://localhost:8080/refine?search=USA&pax=2&order=1> | Search results with filters and sorting (JSON API + browser rendering) |
| Click any property card | Property details page with "Vacation Rentals Near" |
| <http://localhost:8080/privacy-policy>, <http://localhost:8080/site-terms>, <http://localhost:8080/blog/untitled-post-wrurmp> | Other publisher pages |
| <http://localhost:8080/health> | Health check |

The demo dataset (`cmd/dbseed`) contains **24 locations, 72 properties and 72 availability windows**: 15 real US hotels, 33 US demo listings, 12 Toronto and 12 Dhaka demo listings. Demo listings are named "… (Demo)" and their button reads **View Other Hotel**, linking to a real hotel in the same city on Booking.com.

### 4.3 Configuration (`.env`)

`.env` is created from `.env.example` and is ignored by git.

| Variable | Default | Purpose |
| --- | --- | --- |
| `DATABASE_URL` | empty | PostgreSQL connection URL. Empty: `run.sh` starts the Docker database and fills it in. |
| `DATABASE_URL_POOLED` | empty | Optional pooled URL, used only when `DATABASE_URL` is empty. |
| `LOCAL_SITE_HOST` | `test-travel-templates.beta.123presto.com` | Publisher served at `http://localhost:8080/` (section 4.4). |
| `DEPLOYMENTS` | `conf/deployment.json` | Comma-separated deployment manifests (section 4.6). |
| `PORT` | unset | Listener, see the table below. |
| `RUNMODE` | `dev` | Beego run mode. |
| `TEST_NEON` | unset | `1` enables the read-only integration tests against the configured database. |

**Database URL rules:** a remote database must use TLS (`sslmode=require`, `verify-ca` or `verify-full`). A database on this machine (`localhost`, `127.0.0.1`, `::1`, such as the Docker container) may use `sslmode=disable`.

`PORT` controls the listener:

| Configuration | Listening address |
| --- | --- |
| `PORT` unset or blank | `127.0.0.1:8080` (this computer only) |
| `PORT=8080` | `0.0.0.0:8080` (also reachable from the local network) |
| Any valid integer from 1 to 65535 | `0.0.0.0:<PORT>` |
| Invalid or out-of-range value | Startup error |

### 4.4 How `localhost` shows the publisher site

Publisher pages are found by **request Host + path** (section 2), and the browser sends `Host: localhost:8080`, which is not a publisher domain. At startup `main.go` passes `LOCAL_SITE_HOST` to `Catalog.SetLocalSiteHost`, which checks that the host is registered. `Catalog.Resolve` then treats requests addressed to this machine (`localhost`, `127.0.0.1`, `::1`, any port) as requests for that host. Every other unknown host still returns 404, and canonical links keep the real publisher domain.

Without `LOCAL_SITE_HOST`, send the publisher Host explicitly:

```bash
curl -i -H 'Host: test-travel-templates.beta.123presto.com' http://127.0.0.1:8080/
```

### 4.5 Using your own PostgreSQL, or running the steps by hand

To use an existing database, put its URL into `DATABASE_URL` in `.env` and run `./run.sh`; Docker is skipped. Or run the steps yourself:

```bash
cp .env.example .env          # then set DATABASE_URL
go run ./cmd/dbsetup          # tables, indexes, foreign keys
go run ./cmd/dbseed           # demo data
bee run                       # or: go run .
```

`cmd/dbsetup` uses Beego ORM synchronization and creates the three application tables:

- `locations`: reusable geographical records.
- `properties`: publisher-scoped property records identified by `site_key`, `property_id`, and `slug`.
- `property_availability`: date windows referencing property rows.

It also creates indexes, scoped uniqueness, and the foreign keys from properties to locations and from availability to properties. **There is no SQL `sites` table.** Both commands can be run again safely: existing tables are kept, and `cmd/dbseed` uses `ReadOrCreate`, so existing rows are never overwritten.

**Tile markers (only for new or updated page JSON).** The bundled `home.json` is already marked. After publishing a page whose `tiles_component` blocks should be filled from the database, mark it once:

```bash
go run ./scripts/tilemarkers content/<path>/page.json          # dry run: lists the blocks it would mark
go run ./scripts/tilemarkers -write content/<path>/page.json   # writes body.content back to the file
```

The script only wraps each block's existing inner HTML in `<!--{{if index .Tiles "<block id>"}}-->…<!--{{else}}-->original<!--{{end}}-->` markers. Without data, the page renders exactly as published. Running it again changes nothing.

### 4.6 Register multiple publishers at startup

Set `DEPLOYMENTS` to a comma-separated list of deployment manifests whose content files are present under `content/`:

```bash
DEPLOYMENTS='conf/deployment.json,conf/second-deployment.json' bee run
```

`conf/second-deployment.json` is a **placeholder**, not a file bundled with this repository. Each manifest must use its own published hostname and canonical routes. `LOCAL_SITE_HOST` maps `localhost` to one publisher; reach the others by their own hostname (for example with `curl -H 'Host: …'`). Tests use synthetic multi-publisher manifests to verify routing isolation.

### 4.7 Troubleshooting

| Message | Fix |
| --- | --- |
| `Go is not installed` | Install Go 1.26+ from <https://go.dev/dl/>. |
| `Docker is not running, or this user cannot use it` | Start Docker; or run `sudo usermod -aG docker $USER`, then log out and in again. |
| `bee: command not found` | `go install github.com/beego/bee/v2@latest` and add `$(go env GOPATH)/bin` to `PATH`. |
| Port `55432` or `8080` already in use | Stop the program using it (`ss -ltnp \| grep -E ':55432\|:8080'`). |
| `Page not found` at `localhost:8080` | `.env` has no `LOCAL_SITE_HOST`; copy the line from `.env.example`. |
| `PostgreSQL did not start` | `docker logs bmr-postgres` |

## 5. Concurrent content loading and caching

### Selected design

`services.ContentLoader` caches **decoded JSON documents individually**, not entire rendered HTML pages:

- `fileCache[models.SiteHeader]` for shared headers;
- `fileCache[models.PageDocument]` for individual pages;
- `fileCache[models.SiteFooter]` for shared footers.

Each cache key is a **storage-relative file path** (for example, `beta/preview/site-a/header.json`), so two sites with identically named files do not collide. Each `ContentLoader` owns its caches; they are **not shared across server processes**.

On a cold or partially cached request, the loader can start three independent goroutines and wait with `sync.WaitGroup`. A generic mutex-protected cache stores in-progress entries with a `ready` channel. Concurrent requests for the **same missing file** wait for the first load, avoiding redundant reads and decodes. File I/O takes place outside the cache map mutex. On a fully warm request, the loader returns cached decoded content **without launching new goroutines**.

`ContentLoader.Stats()` reports attempted file reads and typed cache entry counts. `ContentLoader.Invalidate(relativePaths...)` explicitly evicts selected files. A failed file load is not retained as a successful cache entry. In-flight requests may still complete using previous values after invalidation; invalidation is **not an atomic site-wide publication mechanism**.

The cache currently has **no TTL, eviction policy, or memory-size bound**, and there is no automatic deployment-triggered invalidation. JSON changes require a process restart or an explicit call to `Invalidate` (which is a service method, **not a public endpoint**).

### Tested behaviors

Tests submitted on 5 November 2026 passed for shared header/footer reuse, 50 simultaneous cold-homepage loads producing **three attempted file reads**, targeted invalidation, rejection of ordinary lexical path traversal, multi-site cache separation, and the race detector. The two-site HTTP integration fixture exercises two hosts with identical paths; four pages across those two sites required **eight unique document reads**, and repeated requests caused no additional reads. These results apply to the tested fixture and conditions.

## 6. Optimization alternatives and selection rationale

**Evaluation goal:** Compare practical optimization approaches and select the one supported by the application workload and measured evidence.

There is no finite list of every possible optimization; the table evaluates the **relevant options for this implementation**. Only the selected content-cache design has been directly compared with uncached loading in the submitted benchmarks. Other choices remain candidates, not measured winners or losers.

| Optimization option | Benefits | Costs / limits | Decision |
| --- | --- | --- | --- |
| Scan JSON directories or routes on every request | Simple initial implementation | Repeated file discovery and parsing | **Not selected**; build a catalog at startup |
| Startup host+canonical-path lookup index | Request-time map lookup; deployment metadata remains authoritative | Requires catalog rebuild for deployment changes | **Selected** |
| Sequential header/page/footer reads | Simple control flow | Independent cache misses cannot overlap | **Not selected** for uncached loading |
| Concurrent reads with goroutines + `WaitGroup` | Overlaps independent file I/O | Goroutine and synchronization overhead | **Selected only for cold/partial misses** |
| No content cache | Always reads current local files | Repeated I/O, decode and allocations | **Benchmark baseline**, not selected |
| Cache decoded JSON in Go process memory | Reuses headers/footers, avoids I/O and JSON decode on hits, no network round trip | Process-local, potentially stale, unbounded memory | **Selected and measured** |
| Cache complete rendered HTML | Potentially avoids repeated template execution | More complicated page/site invalidation; higher duplicated HTML memory | **Deferred**, not benchmarked |
| Redis/distributed cache | Shared cache across server instances; centralized invalidation is possible with additional design | Separate service/network traffic, serialization and invalidation complexity | **Not selected for the current single-process application**; not directly benchmarked |
| Local + Redis hybrid cache | Local hits plus distributed coordination | Two-layer invalidation and operational complexity | **Deferred** |
| `sync.Map` or `sync.RWMutex` instead of mutex map | Might help particular contention patterns | Requires dedicated benchmark and correctness review | **Not changed** |
| Bounded cache, TTL, LRU or explicit deployment versioning | Controls memory growth/staleness | Requires lifetime, eviction and publishing semantics | **Future work** |
| Preallocate buffers, stream template output or cache templates/HTML | Might reduce response allocations | Must preserve exact HTML and correct Beego response behavior | **Investigate only with A/B tests** |
| HTTP compression, browser caching, CDN optimization | Can reduce transferred bytes or asset delivery latency | Separate from JSON decode; depends on proxy, assets and clients | **Not measured by this service benchmark** |
| Additional goroutines on warm-cache hits | No independent filesystem I/O remains | Adds scheduling/synchronization overhead | **Not selected** |

### Why in-process decoded JSON caching was selected over Redis

This application uses a **single Beego process reading local files**. Cached Go structs can be reused directly in the same process, with no Redis service or network request. Individual file caching also shares one header and footer among multiple pages while retaining distinct storage-path keys across sites.

Redis becomes valuable when **multiple application instances** need a shared cache, coordinated updates, or more extensive operational controls. It is **not claimed to be intrinsically slower in every deployment**; Redis and this in-process cache have **not been directly benchmarked** here. Adding Redis today would introduce components that do not address the remaining observed template/output-buffer allocation costs. Redis's documentation itself discusses the latency advantage of keeping frequently used data in an application-local cache.

**Selection:** retain the startup catalog, concurrent loads for cache misses, deduplicated process-local decoded JSON cache, and a no-goroutine warm-cache path. Consider Redis, bounded eviction, or HTML caching only if deployment requirements and new measurements justify them.

## 7. Tests and reproducible commands

### Functional and race tests

```bash
go test ./... -count=1
go test -race ./... -count=1

go test ./services -run 'TestMultiCatalog' -v -count=1
go test ./tests -run '^TestMultiSiteSSRHTTP$' -v -count=1
```

The functional and race suites passed in the submitted local test runs, including multi-site Beego integration tests. A passing `-race` run means **no race was detected under those test executions**, not that all possible schedules are proven race-free.

### Loader-only benchmarks

```bash
go test ./services -run '^$' \
  -bench 'BenchmarkContentLoad' -benchmem -count=5
```

### End-to-end loopback HTTP benchmarks

```bash
go test ./tests -run '^$' \
  -bench '^BenchmarkSSRHTTP$' -benchmem -benchtime=3s

go test ./tests -run '^$' \
  -bench '^BenchmarkSSRHTTP$' -benchmem -count=5

go test -race ./tests -run '^$' \
  -bench '^BenchmarkSSRHTTP/ParallelWarm$' -benchtime=100x
```

The HTTP benchmarks use `httptest.NewServer` with an actual loopback HTTP client and Beego handler. They include client **and** server work in the same Go process. `ColdWithInvalidation` includes a cache invalidation operation before **every** timed HTTP request; it is **not** a cold process-start benchmark. `Warm` primes the cache first. `ParallelWarm` reports aggregate operations/time rather than a per-visitor latency percentile. Browser layout, remote assets, and internet latency are excluded.

## 8. Measured performance (submitted 5 November 2026)

**Measurement environment:** Linux `amd64`; Intel Core i3-4130 @ 3.40 GHz; benchmarks displayed `-4` (GOMAXPROCS = 4). Values are from the recorded local benchmark output and are not measurements on other hardware.

### Decoded-content loader only

| Benchmark | First run | 5-run median |
| --- | ---: | ---: |
| Uncached concurrent JSON load | 454,484 ns/op | **439,985 ns/op** |
| Cached warm load | 193.7 ns/op | **200.0 ns/op** |

The five-run median comparison is approximately **2,200×** in favor of the warm cache **for this loader-only operation**. It is **not** the whole-page improvement.

### Full loopback HTTP processing

| Scenario | First run | Median of five subsequent runs | Five-run range |
| --- | ---: | ---: | ---: |
| Cold with invalidation | 1,028,646 ns/op | **845,968 ns/op** | 813,265–1,136,400 ns/op |
| Warm | 315,757 ns/op | **369,861 ns/op** | 344,092–593,251 ns/op |
| Parallel warm | 195,989 ns/op | **221,758 ns/op** | 212,743–295,980 ns/op |

Compared with `ColdWithInvalidation`, the **sequential warm median** is approximately **2.29× faster**, corresponding to **56.3% lower ns/op** for these defined benchmark workloads. Parallel warm `ns/op` represents aggregate throughput, **not** individual request latency.

In the initial HTTP benchmark, cold versus warm allocations were **280,773 versus 152,819 B/op** (about **45.6% less**) and **364 versus 267 allocs/op** (about **26.6% fewer**). The allocations reported by the HTTP benchmark include both client and server components.

### CPU and allocation profiling

The corrected **sequential-only warm** profile used:

```bash
go test ./tests -run '^$' \
  -bench '^BenchmarkSSRHTTP$/^Warm$' -benchtime=5s -benchmem \
  -cpuprofile=cpu_warm_sequential.out \
  -memprofile=mem_warm_sequential.out

go tool pprof -top cpu_warm_sequential.out
go tool pprof -top -cum cpu_warm_sequential.out
go tool pprof -alloc_space -top mem_warm_sequential.out
go tool pprof -alloc_objects -top mem_warm_sequential.out
```

This run measured **306,784 ns/op**, **150,711 B/op**, **266 allocs/op**, and **112.33 MB/s** of response-body throughput. In the total-allocated-bytes profile, `bytes.growSlice` accounted for approximately **90.60%** of sampled/estimated allocation volume. Template execution and Beego response writing contributed substantial **overlapping cumulative call paths**. These percentages represent allocations **over the profiling run**, not the application's resident memory or a direct percentage of HTTP response latency.

**Interpretation:** Warm content loading is not the observed main allocation source. Template evaluation and output buffering are candidates for further study, but the existing data do **not** prove that a different renderer or buffer policy is better. No rendering-engine rewrite was justified by these measurements.

## 9. Known limitations and follow-up work

- **Deployment lifecycle:** multiple metadata files can be registered at startup, but no dynamic catalog updates, authenticated deployment callbacks, or coordinated invalidation across instances exist.
- **Cache lifecycle:** in-memory only, unbounded, without automatic TTL/eviction or deployment-driven refresh. Explicit invalidation can leave some already in-flight requests using old content.
- **Storage scope:** JSON is local; remote S3 retrieval and deployment-system integration have not been implemented.
- **Tile markers:** a newly published or updated page JSON must be marked with `scripts/tilemarkers` before its tile blocks are filled; until then the publisher's tiles are shown and a one-time log line names the command.
- **Input and trust boundaries:** lexical relative-path validation is present but is not a full symlink-containment policy. Publisher-supplied HTML is rendered as trusted; do not feed untrusted visitor HTML into those fields.
- **Asset status:** some local `/static/img/...` URLs returned 404 and certain external image URLs returned 403 during browser testing. Root cause for external 403 has not been established. These are intentionally deferred; CSS/JS CDN links remain unchanged.
- **Benchmark scope:** loopback HTTP benchmarks are not browser page-load or production traffic measurements. End-to-end network/CDN timings, peak load tests, and latency percentiles have not been measured.

Future optimizations should start with **measured requirements**, then isolated tests that preserve the same HTML and multi-site routing behavior. Profiling indicates template/output-buffer allocation paths are logical candidates for an experiment, not an automatic rewrite.


## 10. References

- [Go `sync` package](https://pkg.go.dev/sync)
- [Go diagnostics, profiling and tracing](https://go.dev/doc/diagnostics)
- [Beego documentation](https://beegodoc.com/)
- [Redis client-side caching](https://redis.io/docs/latest/develop/clients/client-side-caching/)
