# QueueCTL - Background Job Queue System

A simple CLI tool to manage background jobs with automatic retries and Dead Letter Queue support.

## Quick Start

### Build

```bash
go build -o queuectl ./cmd/queuectl
```

### Basic Usage

```bash
# Add a job
./queuectl enqueue '{"id":"job1","command":"echo hello"}'

# Start workers
./queuectl worker start --count 2

# Check status
./queuectl status

# List jobs
./queuectl list
```

## Commands

### Enqueue Jobs

```bash
# Simple job
./queuectl enqueue '{"id":"job1","command":"echo hello"}'

# Job with custom retries
./queuectl enqueue '{"id":"job2","command":"sleep 2","max_retries":5}'
```

### Workers

```bash
# Start workers (default: 1)
./queuectl worker start

# Start multiple workers
./queuectl worker start --count 3

# Stop workers
./queuectl worker stop
```

### View Jobs

```bash
# Check queue status
./queuectl status

# List all jobs
./queuectl list

# Filter by state
./queuectl list --state pending
./queuectl list --state completed
./queuectl list --state failed
./queuectl list --state dead
```

### Dead Letter Queue (DLQ)

```bash
# View failed jobs
./queuectl dlq list

# Retry a failed job
./queuectl dlq retry job1
```

### Configuration

```bash
# Get config
./queuectl config get max-retries
./queuectl config get backoff-base
./queuectl config get worker-count

# Set config
./queuectl config set max-retries 5
./queuectl config set backoff-base 2.5
./queuectl config set worker-count 3
```

### Reset Database

```bash
# Clear all jobs
./queuectl reset
```

## How It Works

### Job States

1. **pending** → Job waiting to be processed
2. **processing** → Job currently running
3. **completed** → Job finished successfully
4. **failed** → Job failed, will retry
5. **dead** → Job failed permanently (moved to DLQ)

### Retry Logic

- Failed jobs retry automatically with exponential backoff
- Delay = `base ^ attempts` seconds
- Example with base=2: 2s, 4s, 8s, 16s...
- After `max_retries`, job moves to DLQ

### Storage

- Database: `~/.queuectl/queuectl.db` (SQLite)
- Config: `~/.queuectl/config.json`

## Examples

### Example 1: Simple Job

```bash
# Add job
./queuectl enqueue '{"id":"hello","command":"echo Hello World"}'

# Start worker
./queuectl worker start --count 1

# Check result
./queuectl list --state completed
```

### Example 2: Failed Job with Retry

```bash
# Add job that will fail
./queuectl enqueue '{"id":"fail","command":"false","max_retries":2}'

# Start worker
./queuectl worker start --count 1

# Watch it retry
./queuectl list --state failed

# After max retries, check DLQ
./queuectl dlq list
```

### Example 3: Multiple Workers

```bash
# Add multiple jobs
./queuectl enqueue '{"id":"job1","command":"sleep 2"}'
./queuectl enqueue '{"id":"job2","command":"sleep 2"}'
./queuectl enqueue '{"id":"job3","command":"sleep 2"}'

# Start 3 workers (jobs process in parallel)
./queuectl worker start --count 3
```

## Configuration

Default values:
- `max-retries`: 3
- `backoff-base`: 2.0
- `worker-count`: 1

## Requirements

- Go 1.21+
- SQLite (pure Go)

## Project Structure

```
QueueCTL/
├── cmd/queuectl/          # CLI entry point
├── internal/
│   ├── cli/              # CLI commands
│   ├── db/               # Database layer
│   ├── job/              # Job management
│   ├── worker/           # Worker system
│   └── config/           # Configuration
└── README.md
```

## Testing

### A `test_all_commands.sh` bash testing script has been provided

### Quick Test

```bash
# 1. Reset database
./queuectl reset

# 2. Add test jobs
./queuectl enqueue '{"id":"test1","command":"echo success"}'
./queuectl enqueue '{"id":"test2","command":"false","max_retries":2}'

# 3. Start worker
./queuectl worker start --count 1

# 4. Check results
./queuectl status
./queuectl list
```

## Architecture Overview

When you enqueue a job, it starts in the `pending` state. A worker picks it up and moves it to `processing` while executing the command. If the command succeeds, the job moves to `completed`. If it fails, the job goes to `failed` and gets scheduled for retry with exponential backoff. After all retries are exhausted, the job moves to `dead` and ends up in the Dead Letter Queue.

The system uses SQLite to store all jobs persistently. The database lives at `~/.queuectl/queuectl.db` with a single `jobs` table. I enabled WAL mode for better concurrency since multiple workers need to read and write simultaneously. There's also a 5-second busy timeout so workers don't fail immediately when the database is locked.

Workers run as goroutines in the same process. When a worker needs a job, it first selects the next pending job ID, then atomically updates that job's state to `processing` - but only if it's still in `pending` or `failed` state. If the update affects zero rows, it means another worker already claimed that job, so the worker tries again. This prevents duplicate processing without needing `SELECT ... FOR UPDATE` (which SQLite doesn't support well).

For graceful shutdown, workers check for shutdown signals before picking up new jobs. If a job is currently executing when shutdown is requested, the worker finishes that job before exiting. This ensures no jobs are left hanging in the `processing` state.

All state changes happen inside database transactions to keep things atomic. When a job fails, I calculate the next retry time using exponential backoff (base^attempts) and store it in `next_retry_at`. Workers only pick up failed jobs when their retry time has passed. Once a job hits `max_retries`, it moves to the `dead` state and can be manually retried from the DLQ if needed.

### My Assumptions as per the assignment's requirement

I made a few assumptions while building this. First, I assume users are trusted - commands are executed as-is without any sanitization. This keeps things simple but means you shouldn't run untrusted input. Second, workers run as goroutines in the same process rather than separate OS processes. This makes communication faster and management simpler. Third, the database is local - SQLite lives on the same machine, not networked. Finally, commands run through the shell - `sh -c` on Unix/Linux/macOS and `cmd.exe /c` on Windows.

### Trade-offs & Limitations

There are some limitations I decided to live with. Jobs don't have timeouts, so a job could theoretically run forever. If a worker crashes while processing a job, that job stays stuck in `processing` state until you manually fix it. Jobs are processed first-in-first-out with no priority system. There's no scheduling - jobs run immediately when picked up, no `run_at` field. SQLite's concurrency is limited compared to PostgreSQL, though WAL mode helps. I don't capture or log command output - you only see success or failure. And the retry strategy is simple exponential backoff with no jitter or other fancy retry patterns.

I chose SQLite over PostgreSQL because it's simpler - no external dependencies, pure Go driver, works out of the box. Goroutines instead of OS processes because they're easier to manage and communicate faster. The atomic UPDATE approach instead of `SELECT ... FOR UPDATE` because SQLite doesn't handle that well, and this solution is simpler anyway. JSON for config because it's human-readable and easy to edit. And CLI-only because a web interface would add complexity without being in the requirements.

## Notes

- **Cross-platform**: Works on Linux, macOS, and Windows
- **Persistent**: Jobs survive restarts
- **Concurrent**: Multiple workers process jobs safely
- **Graceful**: Workers finish current job before shutdown

## Built for Backend Developer Internship Assignment
