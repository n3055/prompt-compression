# Prompt Compression Engine

A Go middleware that compresses conversation history into keywords before sending it to any LLM — cutting token usage by 60-90%.

---

## The Problem (Why This Exists)

Imagine you're having a conversation with ChatGPT. You send message 1, then message 2, then message 3. By message 20, every single API call is carrying **all 20 previous messages** along with your new question. You're paying for those old tokens *again and again*.

It's like writing a letter to a friend, but every time you write a new letter, you photocopy all your previous letters and stuff them in the same envelope. By letter 20, your envelope is massive and expensive to mail.

**This engine solves that.** Instead of stuffing old letters in the envelope, it writes a sticky note: *"We talked about: goroutines, channels, context package, error handling."* That sticky note is 19 words instead of 2,000.

---

## How It Actually Works (The Feynman Explanation)

### Step 1: What are "tokens" and why do they cost money?

LLMs (like GPT, Claude, Gemini) read text in chunks called **tokens**. A token is roughly ¾ of a word. Every token you send costs money, and there's a hard ceiling on how many tokens you can send per request (the "context window").

So there are two problems:
- **Cost**: More tokens = more money
- **Limit**: Too many tokens = request rejected

### Step 2: The naive approach (what everyone does)

```
┌─────────────────────────────────────────────────────┐
│ Message 1: "Explain goroutines in Go"               │  16 tokens
│ Message 2: "How do channels work?"                  │  24 tokens
│ Message 3: "Show me the context package"            │  20 tokens
│ ...                                                 │
│ Message 20: "How do I deploy this?"                 │  10 tokens
│                                                     │
│ TOTAL CONTEXT SENT: ~1,400 tokens                   │
│ YOUR ACTUAL QUESTION: 10 tokens                     │
│                                                     │
│ You're paying for 1,400 tokens of OLD stuff         │
│ just to ask a 10-token question.                    │
└─────────────────────────────────────────────────────┘
```

93% of your tokens are *repeat payments* for things you already said.

### Step 3: What this engine does instead

Think of it like a human taking notes. When you sit in a 2-hour lecture, you don't write down every single word the professor says. You write down **key concepts**:

```
Lecture notes: goroutines, lightweight threads, M:N scheduling,
channels, synchronization, buffered vs unbuffered, context pkg,
cancellation, timeouts, microservices
```

That's exactly what this engine does to your conversation history.

### Step 4: The compression algorithm (plain English)

Here's what happens to every message, step by step:

```
Input: "Can you explain how goroutines work in Go and how
        they differ from operating system threads?"

Step 1 — Split into words:
  [Can, you, explain, how, goroutines, work, in, Go, and,
   how, they, differ, from, operating, system, threads]

Step 2 — Throw away filler words:
  Remove: Can, you, how, in, and, how, they, from
  Keep:   [explain, goroutines, work, Go, differ, operating, system, threads]

Step 3 — Score what's left:
  Each word gets a score based on:
  • How many times it appears (frequency)
  • Where it appears (first/last sentence = bonus)
  • How long it is (longer words = more meaningful)

Step 4 — Keep the top scorers:
  Result: [goroutines, differ, explain, operating, system, threads, work]
```

**16 words → 7 keywords. That's 56% savings on the first message alone.**

And here's the beautiful part: savings *compound* as the conversation grows.

### Step 5: Why savings compound (the key insight)

```
Turn 1:  16 raw tokens  →  7 compressed  (56% saved)
Turn 2:  40 raw tokens  → 13 compressed  (67% saved)
Turn 3:  60 raw tokens  → 19 compressed  (68% saved)
Turn 10: 200 raw tokens → 35 compressed  (82% saved)
Turn 20: 400 raw tokens → 50 compressed  (87% saved)
```

Why? Because of **deduplication**. When you talk about "goroutines" in message 1, 3, 5, and 7 — the compressed context stores it *once*. The naive approach sends the full text of all four messages every time.

It's like a dictionary: no matter how many times a word appears in a book, the dictionary only lists it once.

### Step 6: The caching trick (never reprocess old messages)

There's one more optimization. When message 5 arrives, we don't re-compress messages 1-4. We already did that. We cached the result.

```
Message 1 arrives → compress → cache: [goroutines, threads, differ]
Message 2 arrives → compress → cache: [goroutines, threads, differ, channels, sync]
                    ↑ only this is new work
Message 3 arrives → compress → cache: [goroutines, threads, differ, channels, sync, context, cancel]
                    ↑ only this is new work
```

Each turn, we only process the *new* message and append its keywords to the cache. O(1) per turn, not O(n).

---

## Quick Start

### Prerequisites
- Go 1.22+ installed ([download](https://go.dev/dl/))

### Run the server
```bash
# Clone and enter the project
cd codeMiddleware

# Install dependencies
go mod tidy

# Start the server
go run ./cmd/server
```

You'll see:
```json
{"level":"INFO","msg":"starting prompt compression engine","port":"8080","max_keywords":8}
{"level":"INFO","msg":"server listening","addr":":8080"}
```

### Send your first message

```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "instructions": "You are a Go programming expert",
    "query": "Explain goroutines and how they differ from threads"
  }'
```

Response:
```json
{
  "success": true,
  "data": {
    "session_id": "550e8400-e29b-...",
    "compressed_prompt": "[Instructions]\nYou are a Go programming expert\n\n[Compressed Context]\ngoroutines | differ | threads | explain\n\n[Current Query]\nExplain goroutines and how they differ from threads",
    "original_context_tokens": 9,
    "compressed_context_tokens": 4,
    "savings_percent": 55.56,
    "turn_number": 1,
    "message": "Prompt compressed and ready. Forward compressed_prompt to your LLM."
  }
}
```

### Continue the conversation (pass back the session_id)

```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "550e8400-e29b-...",
    "instructions": "You are a Go programming expert",
    "query": "Now explain channels and how they synchronize goroutines"
  }'
```

The `compressed_prompt` in the response now contains keywords from *both* messages — but using far fewer tokens than sending both raw messages.

---

## API Reference

### `POST /api/v1/chat`

The main endpoint. Compresses your message and builds a prompt.

**Request Body:**
```json
{
  "session_id": "optional-uuid",
  "instructions": "System prompt for the LLM",
  "query": "User's current message"
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `session_id` | No | Omit to start a new conversation. Include to continue one. |
| `instructions` | Yes | Your system prompt / persona instructions. |
| `query` | Yes | The user's current message. |

**Response:**
```json
{
  "success": true,
  "data": {
    "session_id": "uuid",
    "compressed_prompt": "The full prompt to forward to your LLM",
    "original_context_tokens": 200,
    "compressed_context_tokens": 30,
    "savings_percent": 85.0,
    "turn_number": 5,
    "message": "Prompt compressed and ready."
  }
}
```

### `GET /api/v1/sessions/{id}`

Get session stats — how many turns, total savings, cached keywords.

### `GET /api/v1/sessions/{id}/history`

Full message history with per-turn compression stats.

### `DELETE /api/v1/sessions/{id}`

Delete a session and its cache.

### `GET /health`

Health check. Returns `{"status": "healthy"}`.

---

## Configuration

All settings are via environment variables with sensible defaults:

```bash
# Server
SERVER_PORT=8080              # Port to listen on
SERVER_READ_TIMEOUT=15s       # Max time to read request
SERVER_WRITE_TIMEOUT=15s      # Max time to write response
SERVER_SHUTDOWN_TIMEOUT=10s   # Grace period on shutdown

# Compressor tuning
COMPRESSOR_MAX_KEYWORDS=8     # Max keywords extracted per message
COMPRESSOR_MIN_WORD_LENGTH=3  # Ignore words shorter than this
COMPRESSOR_POSITION_BOOST=1.5 # Score multiplier for first/last sentence
COMPRESSOR_LENGTH_THRESHOLD=5 # Words longer than this get a bonus
COMPRESSOR_LENGTH_BONUS=1.2   # Score multiplier for long words

# Rate limiting
RATE_LIMIT_RPS=10             # Requests per second
RATE_LIMIT_BURST=20           # Burst capacity
```

Copy `.env.example` to `.env` and modify as needed.

---

## Project Structure

```
codeMiddleware/
├── cmd/server/main.go              # Entry point, wires dependencies, graceful shutdown
├── internal/
│   ├── api/
│   │   ├── router.go               # Routes + middleware chain
│   │   ├── handler.go              # HTTP handlers (thin — delegate to services)
│   │   └── response.go             # JSON response helpers
│   ├── config/config.go            # Environment-based configuration
│   ├── domain/models.go            # Core types (Message, Session, ProcessResult)
│   ├── middleware/
│   │   ├── middleware.go           # Logging, panic recovery, CORS, request IDs
│   │   └── ratelimit.go           # Token bucket rate limiter
│   ├── service/
│   │   ├── compressor.go          # ⭐ Keyword extraction + scoring engine
│   │   ├── cache.go               # Incremental prompt cache
│   │   ├── conversation.go        # Orchestrates the full flow
│   │   └── compressor_test.go     # Unit tests
│   └── store/
│       ├── store.go               # Storage interface (repository pattern)
│       └── memory.go              # Thread-safe in-memory implementation
├── pkg/logger/logger.go           # Structured JSON logger
├── docs/
│   └── llm-integration.md        # Guide for connecting to LLM APIs
├── go.mod
├── Makefile
└── .env.example
```

### Why this structure?

Each package has **one job** (Single Responsibility):
- `domain` — defines what things *are* (Message, Session)
- `store` — saves and retrieves things
- `service` — does the thinking (compress, cache, orchestrate)
- `api` — translates HTTP ↔ service calls
- `middleware` — cross-cutting concerns (logging, auth, rate limits)

The `domain` package imports **nothing** from the project. It's the innermost circle. Everything else depends on it, but it depends on nothing. That's Clean Architecture.

---

## Running Tests

```bash
go test -v ./...
```

---

## What This Project Does NOT Do (By Design)

- **Does not call any LLM** — it's middleware. It prepares the prompt, you forward it.
- **Does not persist to disk** — swap `MemoryStore` for Redis/Postgres via the `Store` interface.
- **Does not use ML/NLP** — pure algorithmic compression. Zero model dependencies.

This keeps it fast, portable, and free of external service requirements.

---

## License

MIT
