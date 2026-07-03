edit my readme file to describe the project; Built a Go CLI that aggregates RSS feeds into PostgreSQL with a
  long-running worker that polls subscribed feeds
  on a configurable interval and deduplicates posts on URL.
  • Designed a normalized Postgres schema (users, feeds, feed follows, posts) with goose migrations and
  sqlc-generated type-safe queries, including joins for per-user feed timelines.
  • Built a command-dispatch CLI with middleware-wrapped authenticated handlers over a JSON-backed config layer.
  the above describes the project a little, but feel free to go thru the codebase and explain all the functions.
  make sure to talk thru download, tiny details abt psql and postgres to get the db running, and all the commands# feed-aggregator

A Go CLI that aggregates RSS feeds into PostgreSQL. A long-running worker polls all subscribed feeds on a configurable interval and deduplicates posts by URL. Built with a normalized Postgres schema, goose migrations, sqlc-generated type-safe queries, and a command-dispatch pattern with middleware-wrapped authenticated handlers over a JSON config layer.

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Database Setup](#database-setup)
- [Configuration](#configuration)
- [Commands](#commands)
- [Architecture](#architecture)

---

## Prerequisites

- [Go](https://go.dev/dl/) 1.21+
- [PostgreSQL](https://www.postgresql.org/download/) 14+
- [goose](https://github.com/pressly/goose) (for running migrations)
- [sqlc](https://sqlc.dev/) (only needed if you modify `.sql` query files)

---

## Installation

```bash
git clone https://github.com/ramzygirgis/feed-aggregator
cd feed-aggregator
go build -o gator .
```

This produces a `gator` binary in the project root. You can move it onto your `$PATH`:

```bash
mv gator /usr/local/bin/gator
```

---

## Database Setup

### 1. Install PostgreSQL

**macOS (Homebrew):**
```bash
brew install postgresql@16
brew services start postgresql@16
```

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install postgresql postgresql-contrib
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

### 2. Create a database

```bash
psql postgres
```

Inside the psql shell:
```sql
CREATE DATABASE feed_aggregator;
\q
```

### 3. Run migrations with goose

goose reads the `-- +goose Up` / `-- +goose Down` annotations in each file under `sql/schema/` and applies them in order.

```bash
goose -dir sql/schema postgres "postgres://localhost/feed_aggregator?sslmode=disable" up
```

This creates five tables in order:

| Migration | What it does |
|---|---|
| `001_users.sql` | Creates `users` table with UUID primary key and unique `name` |
| `002_feeds.sql` | Creates `feeds` table; `url` is unique; `user_id` FK → `users` with cascade delete |
| `003_feed_follows.sql` | Creates `feed_follows` join table; composite unique on `(feed_id, user_id)`; both FKs cascade on delete |
| `004_last_fetched_at.sql` | Adds nullable `last_fetched_at TIMESTAMP` column to `feeds` (used by the agg worker to round-robin feeds) |
| `005_posts.sql` | Creates `posts` table; `url` is unique (deduplication); `feed_id` FK → `feeds` with cascade delete |

To roll back all migrations:
```bash
goose -dir sql/schema postgres "postgres://localhost/feed_aggregator?sslmode=disable" down-to 0
```

---

## Configuration

The app reads and writes a JSON config file at `~/.gatorconfig.json`. You must create it before first use:

```bash
echo '{"db_url":"postgres://localhost/feed_aggregator?sslmode=disable","current_user_name":""}' > ~/.gatorconfig.json
```

The two fields:

| Field | Description |
|---|---|
| `db_url` | Full PostgreSQL connection string |
| `current_user_name` | The currently logged-in user — written automatically by `login` and `register` |

The `config` package (`internal/config/configuration.go`) handles reading and writing this file. `Read()` unmarshals it on startup; `SetUser()` updates `current_user_name` and writes the file back. The `state` struct in `commands.go` holds a pointer to the live config and the database query handle, threading both through every handler.

---

## Commands

Run any command as:
```bash
gator <command> [args...]
```

### User Management

#### `register <username>`
Creates a new user in the database, generates a UUID, and sets that user as the active user in `~/.gatorconfig.json`.

```bash
gator register alice
# Success! New user alice has been registered.
```

#### `login <username>`
Verifies the user exists in the database, then sets them as the active user in the config. Does **not** use passwords — this is a single-machine tool.

```bash
gator login alice
# Success! Username has been set to alice.
```

#### `users`
Lists all registered users. Marks the currently logged-in user with `(current)`.

```bash
gator users
# * alice (current)
# * bob
```

#### `reset`
Deletes all rows from the `users` table. Because all other tables have `ON DELETE CASCADE` foreign keys, this wipes the entire database. Useful for development.

```bash
gator reset
# Database has been successfully reset.
```

---

### Feed Management

These commands require a logged-in user (enforced by `middlewareLoggedIn`).

#### `addfeed <name> <url>`
Adds a new feed to the database and automatically follows it for the current user. Requires exactly two arguments: a human-readable name and the RSS feed URL.

```bash
gator addfeed "Hacker News" "https://news.ycombinator.com/rss"
# Feed Name: Hacker News
# Url: https://news.ycombinator.com/rss
# UserID: <uuid>
```

Internally, this creates a row in `feeds` and then immediately creates a row in `feed_follows` for the current user — so you always follow what you add.

#### `feeds`
Lists every feed in the database (not just ones you follow), with the name of the user who added it.

```bash
gator feeds
# ********* FEED 1 *********
# Name: Hacker News
# Url: https://news.ycombinator.com/rss
# UserName: alice
```

#### `follow <url>`
Follows an existing feed (already in the database) by its URL.

```bash
gator follow "https://news.ycombinator.com/rss"
# ********* FEED FOLLOW *********
# Feed Name: Hacker News
# Username: alice
```

#### `following`
Lists all feeds the current user is following.

```bash
gator following
# ****** FEED FOLLOWS FOR alice ******
# - Hacker News
```

#### `unfollow <url>`
Removes the current user's follow on a feed by URL. Does not delete the feed itself.

```bash
gator unfollow "https://news.ycombinator.com/rss"
# alice has successfully unfollowed Hacker News
```

---

### Aggregation

#### `agg [interval]`
Starts the long-running aggregator worker. Polls feeds in round-robin order, fetching new posts and storing them. Runs until interrupted (`Ctrl+C`).

The optional `interval` argument is a Go duration string. Defaults to `300ms`.

```bash
gator agg 1m      # poll every minute
gator agg 30s     # poll every 30 seconds
gator agg         # poll every 300ms (default)
```

**How it works:**

The `GetNextFeedToFetch` query selects the feed with the oldest (or null) `last_fetched_at`, ensuring round-robin fairness across all feeds. The worker then:

1. Marks that feed as fetched (`last_fetched_at = NOW()`)
2. Fetches the RSS XML over HTTP with a `gator` User-Agent header
3. Parses the XML into `RSSFeed` / `RSSItem` structs, HTML-unescaping all text fields
4. Tries two date formats (`RFC1123Z` and `RFC3339`) for `pubDate`
5. Inserts each item as a `Post` — skipping any with a duplicate URL (Postgres unique constraint on `posts.url`)
6. Prints each new post title to stdout

Run `agg` in a separate terminal while you use the other commands.

---

### Browsing

#### `browse [limit]`
Shows posts from feeds you follow, newest first. Defaults to 2 posts. Pass an integer to get more.

```bash
gator browse        # shows 2 posts
gator browse 10     # shows 10 posts
```

The underlying query joins `posts` → `feed_follows` on `feed_id`, filtering by the current user's ID, then orders by `posts.created_at DESC`.

---

## Architecture

### File layout

```
.
├── main.go                        # Entry point: reads config, opens DB, registers commands, dispatches
├── commands.go                    # state/command types, command map, all handlers
├── middleware.go                  # middlewareLoggedIn — wraps authenticated handlers
├── rss_feed.go                    # HTTP fetch + XML parse + HTML unescape
├── internal/
│   ├── config/
│   │   └── configuration.go      # ~/.gatorconfig.json read/write
│   └── database/
│       ├── db.go                  # sqlc boilerplate (New, DBTX interface)
│       ├── models.go              # Go structs for User, Feed, FeedFollow, Post
│       ├── users.sql.go           # Generated: CreateUser, GetUser, GetUserById, GetUsers, ResetDb
│       ├── feeds.sql.go           # Generated: CreateFeed, GetFeeds, GetFeedByURL, GetFeed, MarkFeedFetched, GetNextFeedToFetch
│       ├── feed_follows.sql.go    # Generated: CreateFeedFollow, GetFeedFollowsForUser, DeleteFeedFollow
│       └── posts.sql.go           # Generated: CreatePost, GetPosts
├── sql/
│   ├── schema/                    # goose migration files (001–005)
│   └── queries/                   # sqlc input query files
└── sqlc.yaml                      # sqlc config: schema dir, queries dir, output package
```

### Command dispatch

`main.go` builds a `commands` map (string → handler func), registers each command, reads `os.Args[1]` as the command name, and calls `c.run()`. Unrecognized commands exit with an error message. The `state` struct carries a `*config.Config` and a `*database.Queries` — the two things every handler needs.

### Middleware

`middlewareLoggedIn` is a higher-order function. It wraps any handler that needs an authenticated user (`func(*state, command, database.User) error`) into the standard `func(*state, command) error` signature. It looks up `s.cfg.CurrentUserName` in the database and injects the `User` struct, or returns an error if the user doesn't exist. This is how `addfeed`, `follow`, `following`, `unfollow`, and `browse` are protected.

### RSS fetching

`fetchFeed` in `rss_feed.go` makes a context-aware HTTP GET with a custom `User-Agent: gator` header (some feeds block default Go user agents). It reads the full body, unmarshals XML into `RSSFeed`, then calls `decodeHTML()` to unescape HTML entities in titles and descriptions (e.g. `&amp;` → `&`).

### Deduplication

Posts are deduplicated at the database level: `posts.url` has a `UNIQUE` constraint. `scrapeFeeds` catches the Postgres unique-violation error by checking if the error string contains `"unique"` and continues to the next item instead of aborting.

### sqlc

`sqlc.yaml` points sqlc at `sql/schema/` (for type information) and `sql/queries/` (for named queries). Running `sqlc generate` regenerates everything under `internal/database/`. The generated code is checked into the repo so you don't need sqlc installed to build.
