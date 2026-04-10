.PHONY: test test-backend test-alpaca bench loadtest

# ─── Unit Tests ──────────────────────────────────────────────────────────────

# Run all backend unit tests.
# config.init() in the main module requires REDIS_HOST and POSTGRES_HOST to be
# non-empty; no live service connection is made during unit tests.
test-backend:
	@echo "==> Running backend unit tests (src/models, src/auth)"
	JWT_SECRET=test-secret-key-streamcore-v2 \
	REDIS_HOST=localhost POSTGRES_HOST=localhost \
	POSTGRES_USER=sc2 POSTGRES_PASSWORD=sc2 POSTGRES_DATABASE=sc2 \
	go test ./src/models/... ./src/auth/... -v -count=1

# Run alpaca-feed unit tests.
# config.init() in alpaca-feed requires ALPACA_API_KEY/SECRET to be non-empty;
# no WebSocket connection is made during these tests.
test-alpaca:
	@echo "==> Running alpaca-feed unit tests"
	cd alpaca-feed && \
	ALPACA_API_KEY=test \
	ALPACA_API_SECRET=test \
	ALPACA_SYMBOLS=AAPL,MSFT \
	KAFKA_HOST=localhost \
	REDIS_HOST=localhost \
	go test ./... -v -count=1

# Run all unit tests.
test: test-backend test-alpaca

# ─── Benchmarks ──────────────────────────────────────────────────────────────

# Benchmark the JSON hot path: MarketTick marshal/unmarshal, WSMessage parse,
# and Alpaca batch parse.  No external services required.
bench:
	@echo "==> Benchmarking JSON serialization hot path"
	go test ./src/models/... -bench=. -benchmem -benchtime=3s -count=1

# ─── Load Test ───────────────────────────────────────────────────────────────

# End-to-end throughput and latency test against a running stack.
# Start the stack first: docker-compose up --build
loadtest:
	@echo "==> Running load test against $(or $(ADDR),ws://localhost:7700)"
	cd cmd/loadtest && go run . \
		-addr=$(or $(ADDR),ws://localhost:7700) \
		-redis=$(or $(REDIS),localhost:7704) \
		-clients=$(or $(CLIENTS),10) \
		-duration=$(or $(DURATION),30) \
		-symbol=$(or $(SYMBOL),BTC/USD) \
		-rate=$(or $(RATE),50)
