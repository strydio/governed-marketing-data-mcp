# Smoke test flow

## Purpose

The smoke test verifies the local backend MCP server is running, authenticated, governed, and returning the expected seeded evidence.

It is intended as a quick local check after database setup and server startup. It is not a replacement for the automated test suite.

## What the smoke test checks

The smoke test command:

```bash
make smoke
```

runs:

```bash
go run ./cmd/smoke-test
```

The Go smoke client uses the official MCP SDK Streamable HTTP client and connects to:

```dotenv
MCP_ENDPOINT=http://localhost:8080/mcp
```

It authenticates with:

```dotenv
DEMO_BEARER_TOKEN=replace-me
```

## Flow

The smoke test performs this sequence:

```text
1. Connect to the MCP server using Streamable HTTP.
2. Send the configured bearer token via the SDK client auth handler.
3. Call tools/list.
4. Verify the MCP server advertises only the approved governed tools.
5. Treat `run_sql` as an example of an unapproved tool that must not appear in the advertised tool list.
6. Call get_campaign_performance for UK autumn-sale.
7. Verify the expected evidence ID is returned.
8. Call get_campaign_performance for US summer-sale.
9. Verify the unauthorised US call is denied safely.
```

Expected governed tools:

```text
get_campaign_performance
get_conversion_funnel
get_tracking_health
get_data_freshness
```

Example unapproved tool that must not be advertised:

```text
run_sql
```

Expected authorised UK evidence ID:

```text
ev-campaign-uk-autumn-sale-20260914-20260920
```

## Running locally

From a fresh checkout:

```bash
cp .env.example .env
```

Set a local token in `.env`, for example:

```dotenv
DEMO_BEARER_TOKEN=replace-me
```

Prepare the database:

```bash
make db-setup
```

Start the server in terminal 1:

```bash
DEMO_BEARER_TOKEN=replace-me make run-server
```

Run the smoke test in terminal 2:

```bash
DEMO_BEARER_TOKEN=replace-me make smoke
```

Expected output:

```text
✓ MCP server advertises only the approved governed tools
✓ authorised UK campaign call returned expected evidence
✓ unauthorised US campaign call was denied safely
smoke test passed
```

## Notes

Normal server startup does not reset or seed the database. Database setup is explicit:

```bash
make db-reset
make db-seed
```

or:

```bash
make db-setup
```

The smoke test expects the server to already be running.
