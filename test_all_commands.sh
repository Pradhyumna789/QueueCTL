#!/bin/bash
# Comprehensive test of all CLI commands

set -e

echo "=========================================="
echo "Testing All CLI Commands"
echo "=========================================="
echo ""

# Build
echo "1. Building..."
go build -o queuectl ./cmd/queuectl
echo "✅ Build successful"
echo ""

# Reset database
echo "2. Resetting database..."
./queuectl reset
echo ""

# Test enqueue with various inputs
echo "3. Testing enqueue command..."
echo "3.1. Valid job..."
./queuectl enqueue '{"id":"test1","command":"echo hello"}'
echo ""

echo "3.2. Job with max_retries..."
./queuectl enqueue '{"id":"test2","command":"sleep 1","max_retries":5}'
echo ""

echo "3.3. Job with all fields..."
./queuectl enqueue '{"id":"test3","command":"echo test","max_retries":3,"state":"pending"}'
echo ""

echo "3.4. Testing duplicate ID (should show friendly error)..."
./queuectl enqueue '{"id":"test1","command":"echo duplicate"}' 2>&1 || true
echo ""

echo "3.5. Testing invalid JSON..."
./queuectl enqueue '{"id":"test4","command":"echo invalid' 2>&1 || true
echo ""

echo "3.6. Testing missing required fields..."
./queuectl enqueue '{"id":"test5"}' 2>&1 || true
echo ""

# Test status
echo "4. Testing status command..."
./queuectl status
echo ""

# Test list commands
echo "5. Testing list commands..."
echo "5.1. List all jobs..."
./queuectl list
echo ""

echo "5.2. List pending jobs..."
./queuectl list --state pending
echo ""

echo "5.3. List completed jobs..."
./queuectl list --state completed
echo ""

echo "5.4. List failed jobs..."
./queuectl list --state failed
echo ""

echo "5.5. List dead jobs..."
./queuectl list --state dead
echo ""

echo "5.6. List processing jobs..."
./queuectl list --state processing
echo ""

echo "5.7. Testing invalid state..."
./queuectl list --state invalid 2>&1 || true
echo ""

# Test config commands
echo "6. Testing config commands..."
echo "6.1. Get max-retries..."
./queuectl config get max-retries
echo ""

echo "6.2. Get backoff-base..."
./queuectl config get backoff-base
echo ""

echo "6.3. Get worker-count..."
./queuectl config get worker-count
echo ""

echo "6.4. Get invalid key..."
./queuectl config get invalid-key 2>&1 || true
echo ""

echo "6.5. Set max-retries..."
./queuectl config set max-retries 5
./queuectl config get max-retries
echo ""

echo "6.6. Set backoff-base..."
./queuectl config set backoff-base 2.5
./queuectl config get backoff-base
echo ""

echo "6.7. Set worker-count..."
./queuectl config set worker-count 3
./queuectl config get worker-count
echo ""

echo "6.8. Set invalid key..."
./queuectl config set invalid-key value 2>&1 || true
echo ""

# Test DLQ commands
echo "7. Testing DLQ commands..."
echo "7.1. List DLQ (should be empty)..."
./queuectl dlq list
echo ""

echo "7.2. Retry non-existent job..."
./queuectl dlq retry nonexistent 2>&1 || true
echo ""

# Test worker commands
echo "8. Testing worker commands..."
echo "8.1. Stop workers when none running..."
./queuectl worker stop 2>&1 || true
echo ""

echo "8.2. Start workers with default count..."
timeout 2 ./queuectl worker start --count 1 2>&1 || true
echo ""

echo "8.3. Start workers with custom count..."
timeout 2 ./queuectl worker start --count 2 2>&1 || true
echo ""

echo "8.4. Start workers with invalid count..."
./queuectl worker start --count 0 2>&1 || true
echo ""

echo "8.5. Start workers with negative count..."
./queuectl worker start --count -1 2>&1 || true
echo ""

# Test help commands
echo "9. Testing help commands..."
echo "9.1. Root help..."
./queuectl --help | head -20
echo ""

echo "9.2. Enqueue help..."
./queuectl enqueue --help
echo ""

echo "9.3. Worker help..."
./queuectl worker --help
echo ""

echo "9.4. Config help..."
./queuectl config --help
echo ""

# Test edge cases
echo "10. Testing edge cases..."
echo "10.1. Empty command..."
./queuectl enqueue '{"id":"empty","command":""}' 2>&1 || true
echo ""

echo "10.2. Very long job ID..."
./queuectl enqueue '{"id":"'"$(python3 -c 'print("a"*100)')"'","command":"echo test"}' 2>&1 || true
echo ""

echo "10.3. Special characters in command..."
./queuectl enqueue '{"id":"special","command":"echo \"hello world\""}'
echo ""

echo "10.4. Command with quotes..."
./queuectl enqueue '{"id":"quotes","command":"echo '\''test'\''"}'
echo ""

# Test with actual job processing
echo "11. Testing job processing..."
echo "11.1. Add jobs..."
./queuectl enqueue '{"id":"proc1","command":"echo processing1"}'
./queuectl enqueue '{"id":"proc2","command":"echo processing2"}'
echo ""

echo "11.2. Start worker and process jobs..."
timeout 3 ./queuectl worker start --count 1 2>&1 || true
echo ""

echo "11.3. Check status after processing..."
./queuectl status
echo ""

echo "11.4. List completed jobs..."
./queuectl list --state completed
echo ""

# Test failed job and DLQ
echo "12. Testing failed job and DLQ..."
echo "12.1. Add job that will fail..."
./queuectl enqueue '{"id":"fail1","command":"false","max_retries":1}'
echo ""

echo "12.2. Process failed job..."
timeout 5 ./queuectl worker start --count 1 2>&1 || true
echo ""

echo "12.3. Check DLQ..."
./queuectl dlq list
echo ""

echo "12.4. Retry DLQ job..."
./queuectl dlq retry fail1
echo ""

echo "12.5. Check status..."
./queuectl status
echo ""

# Test reset command
echo "13. Testing reset command..."
echo "13.1. Reset database..."
./queuectl reset
echo ""

echo "13.2. Reset again (should handle gracefully)..."
./queuectl reset
echo ""

echo "=========================================="
echo "All tests completed!"
echo "=========================================="

