# Beego Multisite Renderer

**Deployment-metadata-driven, multi-site server-side rendering in Go and Beego.**

Beego Multisite Renderer converts publisher-managed JSON content into HTML using Beego MVC templates. It resolves requests through a startup catalog keyed by **request Host + canonical URL path**, combines shared header/page/footer documents, and caches decoded JSON in memory. Accommodation listings are stored separately in PostgreSQL: country pages render property cards on the server, while filtered search results use a JSON API and client-side rendering.

The JSON content is read from local files under `content/` using the publisher storage-path structure. The application does **not** fetch publisher JSON directly from S3 during a request; external CSS, JavaScript, and permitted image URLs referenced by the content remain external resources.

## 1. Features and technology

| Area | Implementation |
| --- | --- |
| Runtime | Go 1.25 (`go.mod` specifies `go 1.25.0`) |
| Web framework | Beego v2.1.0, MVC controllers and Go HTML templates |
| Publisher routing | Startup `Catalog` indexed by normalized Host + canonical path |
| Multiple publishers | `NewMultiCatalog`, one deployment manifest per publisher |
| JSON loading | Concurrent reads on cache misses; typed decoded-content caches; single-flight suppression |
| Property data | PostgreSQL (Neon-compatible) via Beego ORM and pgx v5 |
| SQL tables | `locations`, `properties`, `property_availability` |
| Property isolation | Stable `site_key`; scoped unique property identifiers and slugs |
| Country listings | `/all/:country`: server-rendered cards, eight per page |
| Filtered results | `/refine`: JavaScript rendering from `/api/properties`, eight per page |
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
4. `controllers.PageController` resolves the **actual request Host and path**. Unknown hosts or paths return HTTP 404.
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
├── main.go                      # Startup, catalog, loader, database, listeners
├── go.mod / go.sum
├── .env.example                 # Environment-variable template
├── conf/
│   ├── app.conf
│   └── deployment.json          # Publisher deployment metadata
├── cmd/
│   ├── dbsetup/                 # Fresh database schema setup
│   └── dbseed/                  # Initial property/availability records
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
│   └── *_test.go
├── controllers/
│   ├── page.go
│   ├── country.go
│   ├── refine.go
│   ├── property_api.go
│   └── health.go
├── routers/router.go
├── views/
│   ├── page.tpl
│   └── refine.tpl
├── static/
│   ├── css/                    # refine.css, all.css, search popup CSS
│   ├── js/                     # refine.js, property_search_popup.js
│   └── img/
├── content/                    # Local publisher JSON storage hierarchy
└── tests/                      # HTTP integration tests and benchmarks
```

## 4. Setup and running locally

### Prerequisites

- Go 1.25 or a compatible newer toolchain.
- Access to a PostgreSQL database with TLS enabled; Neon PostgreSQL is the development database used for this project.
- Publisher deployment metadata in `conf/deployment.json` and the corresponding local files under `content/`.
- Node.js is optional for checking JavaScript syntax. Bee CLI is optional if you prefer `bee run` to `go run .`.

### 4.1 Install dependencies and configure the environment

Clone the repository, then work from the project root:

```bash
git clone https://github.com/Soumyo001/beego-multisite-renderer.git
cd beego-multisite-renderer
go mod download
cp .env.example .env
```

Edit `.env` with your own database connection URL. The standard configuration has these keys (no additional database variable is required):

```dotenv
DEPLOYMENTS=conf/deployment.json
PORT=8080
RUNMODE=
DATABASE_URL=
DATABASE_URL_POOLED=
```

Set **`DATABASE_URL`** to a valid `postgres://` or `postgresql://` connection string with `sslmode=require` (or `verify-ca` / `verify-full`). Do not commit `.env` or include credentials in logs. If `DATABASE_URL` is empty, the running server can use `DATABASE_URL_POOLED`; the database setup utility needs `DATABASE_URL` explicitly.

`PORT` controls the listener:

| Configuration | Listening address |
| --- | --- |
| `PORT` unset or blank | `127.0.0.1:8080` |
| `PORT=8080` | `0.0.0.0:8080` |
| Any valid integer from 1 to 65535 | `0.0.0.0:<PORT>` |
| Invalid or out-of-range value | Startup error |

A `0.0.0.0` listener can be reached through network interfaces permitted by firewall and routing rules. A listening address alone does **not** register a request Host in the application's catalog.`RUNMODE` can be left blank to use the configured Beego default.

### 4.2 Initialize a brand-new PostgreSQL database

On **first setup of an empty database**, with the required `DATABASE_URL` configured, run:

```bash
go run ./cmd/dbsetup
go run ./cmd/dbseed
```

`cmd/dbsetup` uses Beego ORM synchronization and creates the three application tables:

- `locations`: reusable geographical records.
- `properties`: publisher-scoped property records identified by `site_key`, `property_id`, and `slug`.
- `property_availability`: date windows referencing property rows.

It also establishes indexes, scoped uniqueness, and foreign-key relationships from properties to locations and availability to properties. **There is no SQL `sites` table.**

`cmd/dbseed` inserts the bundled demonstration dataset for the registered sample publisher: **8 locations, 15 properties, and 15 availability records** on an empty database. It uses `DATABASE_URL` in the standard environment configuration; its code also recognizes an optional direct-connection override, but that override is **not required** in `.env.example`. Seeding uses `ReadOrCreate`, so existing matching records are preserved rather than overwritten.

**Run schema setup and seeding only against a database you intend to initialize.** The commands can write schema/data; neither is needed for normal application startup once the database has been prepared. Schema privileges and a database connection are required.

### 4.3 Start the application

```bash
go run .
```

Or, if Bee CLI is installed:

```bash
bee run
```

Startup loads the publisher catalog and decoded-content loader, establishes the PostgreSQL connection, registers the routes, then begins serving HTTP. An unavailable or unconfigured database causes startup to fail.

Because publisher routes require a registered hostname, the following curl example explicitly supplies the expected Host header:

```bash
curl -i -H 'Host: test-travel-templates.beta.123presto.com' \
  http://127.0.0.1:8080/

curl -i -H 'Host: test-travel-templates.beta.123presto.com' \
  http://127.0.0.1:8080/all/usa

curl -i -H 'Host: test-travel-templates.beta.123presto.com' \
  'http://127.0.0.1:8080/api/properties?search=USA&pax=2&order=1'

curl -i http://127.0.0.1:8080/health
```

With the bundled 15-property dataset, `/all/usa` renders eight property cards in the initial HTML and `/all/usa?page=2` renders seven. `/all/canada` is a valid empty country page with this seed data. `/refine?search=USA&pax=2&order=1` uses the property API and client-side rendering. Unknown publisher hosts return 404 for content and property routes.

### 4.4 Register multiple publishers at startup

Set `DEPLOYMENTS` to a comma-separated list of deployment manifests whose local content files are present:

```bash
DEPLOYMENTS='conf/deployment.json,conf/second-deployment.json' go run .
```

`conf/second-deployment.json` is a **placeholder**, not a file bundled with this repository. Each manifest must use its own valid published hostname and canonical routes. Tests use synthetic multi-publisher manifests to verify routing isolation.

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
- **Input and trust boundaries:** lexical relative-path validation is present but is not a full symlink-containment policy. Publisher-supplied HTML is rendered as trusted; do not feed untrusted visitor HTML into those fields.
- **Asset status:** some local `/static/img/...` URLs returned 404 and certain external image URLs returned 403 during browser testing. Root cause for external 403 has not been established. These are intentionally deferred; CSS/JS CDN links remain unchanged.
- **Benchmark scope:** loopback HTTP benchmarks are not browser page-load or production traffic measurements. End-to-end network/CDN timings, peak load tests, and latency percentiles have not been measured.

Future optimizations should start with **measured requirements**, then isolated tests that preserve the same HTML and multi-site routing behavior. Profiling indicates template/output-buffer allocation paths are logical candidates for an experiment, not an automatic rewrite.


## 10. References

- [Go `sync` package](https://pkg.go.dev/sync)
- [Go diagnostics, profiling and tracing](https://go.dev/doc/diagnostics)
- [Beego documentation](https://beegodoc.com/)
- [Redis client-side caching](https://redis.io/docs/latest/develop/clients/client-side-caching/)
