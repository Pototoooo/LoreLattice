# Quick Start Guide

Get started with the WeKnora Agent Test Tool in 5 minutes!

## Prerequisites

- Go 1.21 or higher
- Access to a running WeKnora instance
- A knowledge base ID (or create one through the tool)

## Quick Start

### Option 1: Using the Build Script (Recommended)

```bash
# 1. Navigate to the tool directory
cd client/cmd/agent_test

# 2. Build and run with your knowledge base
./run.sh -k kb_your_knowledge_base_id

# Or with a remote server
./run.sh -u https://api.weknora.com -k kb_12345 -t your_token
```

### Option 2: Using Make

```bash
# Build the binary
make build

# Run with environment variables
KB_ID=kb_12345 make run-local

# Or run the binary directly
./agent_test -url http://localhost:8080 -kb kb_12345
```

### Option 3: Direct Go Build

```bash
# Build
go build -o agent_test .

# Run
./agent_test -url http://localhost:8080 -kb kb_your_knowledge_base_id
```

## First Steps

Once the tool is running, you'll see an interactive prompt:

```
╔════════════════════════════════════════════════════════════╗
║         WeKnora Agent QA Testing Tool                     ║
╚════════════════════════════════════════════════════════════╝

[No Session] > 
```

### 1. Create a New Session

```
> new
```

The tool will create a new session and show you the session ID.

### 2. Ask Your First Question

```
[Session: abc12345] > ask What is machine learning?
```

You'll see real-time output showing:
- 💭 Agent thinking process
- 🔧 Tool calls (if any)
- 📚 Knowledge references
- 📝 Final answer
- 🤔 Reflections (if any)

### 3. Continue the Conversation

```
[Session: abc12345] > ask Can you give me examples?
[Session: abc12345] > ask How does it work?
```

### 4. View Session Info

```
[Session: abc12345] > info
```

### 5. List All Sessions

```
[Session: abc12345] > sessions
```

### 6. Exit

```
[Session: abc12345] > exit
```

## Command Reference

| Command | Description | Example |
|---------|-------------|---------|
| `new` | Create a new session | `> new` |
| `ask <query>` | Ask the agent a question | `> ask What is AI?` |
| `info` | Show current session details | `> info` |
| `sessions` | List all your sessions | `> sessions` |
| `switch <id>` | Switch to another session | `> switch sess_123` |
| `help` | Show help message | `> help` |
| `exit` or `quit` | Exit the program | `> exit` |

## Example Session

```bash
$ ./run.sh -k kb_ml_docs

[No Session] > new
✓ Session created successfully!
  Session ID: sess_a1b2c3d4
  Knowledge Base: kb_ml_docs
  Created At: 2025-11-03T10:00:00Z

[Session: sess_a1b] > ask What is a neural network?

╔═══ Agent Processing ═══╗
Query: What is a neural network?

💭 Thinking: I need to search the knowledge base for information about neural networks...

🔧 Tool Call: search_knowledge
   Arguments: map[query:neural network definition]

✓ Tool Result:
   Found 3 relevant knowledge chunks

📚 Knowledge References: Found 3 reference(s)
   1. [Score: 0.892] A neural network is a series of algorithms...
      Knowledge: Neural Networks 101 (Chunk: 2)
   2. [Score: 0.856] Neural networks mimic biological neurons...
      Knowledge: Deep Learning Basics (Chunk: 5)
   3. [Score: 0.834] The structure consists of layers...
      Knowledge: ML Architectures (Chunk: 8)

📝 Final Answer:
A neural network is a computing system inspired by biological neural networks 
in animal brains. It consists of interconnected nodes (artificial neurons) 
organized in layers that process information using a connectionist approach 
to computation. Neural networks learn to perform tasks by considering examples, 
generally without being programmed with task-specific rules.

🤔 Reflection: I successfully found relevant information in the knowledge base 
and provided a comprehensive answer covering the definition, structure, and 
learning mechanism of neural networks.

╔═══ Summary ═══╗
Duration: 2.34s
Tool Calls: 1 (search_knowledge)
References: 3
Reflections: 1
╚═══════════════╝

[Session: sess_a1b] > ask What are the main types?

╔═══ Agent Processing ═══╗
Query: What are the main types?

💭 Thinking: The user is asking about types of neural networks based on the 
previous context...

[... continuing the conversation ...]
```

## Testing Scenarios

We provide a test scenarios script for common testing patterns:

```bash
# Run interactive scenario menu
./test_scenarios.sh

# Run a specific scenario
./test_scenarios.sh 1  # Basic Q&A
./test_scenarios.sh 2  # Complex queries
./test_scenarios.sh 3  # Multi-turn conversation
```

## Configuration with Environment Variables

Create a configuration file for repeated use:

```bash
# Create config file
cat > config.env << EOF
export WEKNORA_URL=http://localhost:8080
export WEKNORA_KB_ID=kb_your_knowledge_base_id
export WEKNORA_TOKEN=your_token_if_needed
EOF

# Load and run
source config.env
./run.sh
```

## Troubleshooting

### "Connection refused"
- Make sure WeKnora server is running
- Check the URL is correct
- Verify network connectivity

### "Knowledge base not found"
- Verify your knowledge base ID is correct
- Check you have access to the knowledge base
- Ensure the knowledge base has been initialized

### "Session not found"
- The session may have expired or been deleted
- Create a new session with `new` command

### No tool calls happening
- Check your knowledge base has content
- Verify the agent configuration includes tool definitions
- Look for error messages in the output

## Advanced Usage

### Using with Docker

If WeKnora is running in Docker:

```bash
# Get the container's IP or use host networking
./agent_test -url http://localhost:8080 -kb kb_123
```

### Testing with Production

```bash
# Use production URL with authentication
./agent_test \
  -url https://api.weknora.com \
  -token $PROD_TOKEN \
  -kb kb_prod_123
```

### Continuing an Existing Session

```bash
# Skip session creation and continue existing one
./agent_test -url http://localhost:8080 -session sess_existing_id
```

### Batch Testing

```bash
# Pipe commands for automated testing
echo -e "new\nask What is AI?\ninfo\nexit" | ./agent_test -url http://localhost:8080 -kb kb_123
```

## Next Steps

- Read the full [README.md](README.md) for detailed documentation
- Check [agent_example.go](../../agent_example.go) for code examples
- Explore the [client package documentation](../../README.md)
- Review agent events and response types in the main README

## Getting Help

If you encounter issues:

1. Check the [README.md](README.md) for detailed documentation
2. Review the error messages - they usually indicate the problem
3. Verify your configuration (URL, KB ID, token)
4. Check WeKnora server logs for backend issues
5. Use `info` command to check session state

## Tips

- **Start Simple**: Begin with basic questions to verify connectivity
- **Use Info**: The `info` command shows valuable session configuration
- **Watch Events**: Pay attention to all event types to understand agent behavior
- **Check References**: Knowledge references show if retrieval is working correctly
- **Monitor Performance**: The summary shows timing and tool usage statistics

Happy testing! 🚀

