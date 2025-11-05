# WeKnora Agent QA Testing Tool

An interactive command-line tool for testing and debugging WeKnora's Agent QA functionality.

## Features

- 🎯 Interactive terminal interface with colored output
- 📊 Real-time streaming of agent events
- 🔧 Display tool calls and their results
- 💭 Show agent thinking process
- 📚 Display knowledge references with scores
- 🤔 Capture agent reflections
- 📝 Track complete conversation flow
- ⏱️ Performance metrics and timing
- 🔄 Session management (create, switch, list)

## Installation

```bash
cd client/cmd/agent_test
go build -o agent_test
```

## Usage

### Basic Usage

```bash
# Start with a new session
./agent_test -url http://localhost:8080 -kb <knowledge_base_id>

# With authentication token
./agent_test -url http://localhost:8080 -token <your_token> -kb <knowledge_base_id>

# Use existing session
./agent_test -url http://localhost:8080 -session <session_id>
```

### Command Line Flags

- `-url` : WeKnora API base URL (default: http://localhost:8080)
- `-token` : API authentication token (optional)
- `-kb` : Knowledge base ID (required for creating new sessions)
- `-session` : Existing session ID to continue conversation (optional)

### Interactive Commands

Once the tool is running, you can use these commands:

#### Session Management

- `new` - Create a new session
  ```
  > new
  ```

- `sessions` - List all sessions
  ```
  > sessions
  ```

- `switch <session_id>` - Switch to another session
  ```
  > switch abc123...
  ```

- `info` - Show current session information
  ```
  > info
  ```

#### Agent Interaction

- `ask <query>` - Ask the agent a question
  ```
  > ask What is machine learning?
  > ask How can I improve model accuracy?
  > ask Explain the difference between supervised and unsupervised learning
  ```

#### General

- `help` - Show help message
- `exit` or `quit` - Exit the program

## Agent Events

The tool displays various agent events in real-time:

### 💭 Thinking Events
Shows the agent's internal reasoning process
```
💭 Thinking: Let me search for information about machine learning...
```

### 🔧 Tool Call Events
Displays when the agent calls a tool
```
🔧 Tool Call: search_knowledge
   Arguments: {"query": "machine learning definition"}
```

### ✓ Tool Result Events
Shows the results from tool execution
```
✓ Tool Result:
   Found 5 relevant knowledge chunks
```

### 📚 Knowledge References
Lists retrieved knowledge with relevance scores
```
📚 Knowledge References: Found 3 reference(s)
   1. [Score: 0.875] Machine learning is a subset of artificial intelligence...
      Knowledge: ML Basics (Chunk: 2)
   2. [Score: 0.823] There are three main types of machine learning...
      Knowledge: ML Types (Chunk: 5)
```

### 📝 Final Answer
The agent's complete response
```
📝 Final Answer:
Machine learning is an application of artificial intelligence that provides 
systems the ability to automatically learn and improve from experience...
```

### 🤔 Reflection Events
Agent's self-reflection on its responses
```
🤔 Reflection: I provided a comprehensive answer covering the main aspects 
of machine learning with relevant examples.
```

### Summary
At the end of each query, a summary is displayed:
```
╔═══ Summary ═══╗
Duration: 3.45s
Tool Calls: 2 (search_knowledge, format_response)
References: 3
Reflections: 1
╚═══════════════╝
```

## Example Session

```bash
$ ./agent_test -url http://localhost:8080 -kb kb_12345

╔════════════════════════════════════════════════════════════╗
║         WeKnora Agent QA Testing Tool                     ║
╚════════════════════════════════════════════════════════════╝

[No Session] > new
✓ Session created successfully!
  Session ID: sess_abc123xyz
  Knowledge Base: kb_12345
  Created At: 2025-11-03T10:30:00Z

[Session: sess_abc] > ask What is neural networks?

╔═══ Agent Processing ═══╗
Query: What is neural networks?

💭 Thinking: I need to search for information about neural networks...

🔧 Tool Call: search_knowledge
   Arguments: map[query:neural networks]

✓ Tool Result:
   Successfully retrieved 3 relevant chunks

📚 Knowledge References: Found 3 reference(s)
   1. [Score: 0.892] A neural network is a series of algorithms...
      Knowledge: Deep Learning Fundamentals (Chunk: 1)
   2. [Score: 0.856] Neural networks consist of layers of neurons...
      Knowledge: Neural Network Architecture (Chunk: 3)

📝 Final Answer:
Neural networks are computing systems inspired by biological neural networks 
that constitute animal brains. They consist of interconnected nodes (neurons) 
organized in layers...

🤔 Reflection: The answer provides a clear definition with supporting 
references from the knowledge base.

╔═══ Summary ═══╗
Duration: 2.87s
Tool Calls: 1 (search_knowledge)
References: 3
Reflections: 1
╚═══════════════╝

[Session: sess_abc] > info

╔═══ Session Information ═══╗
Session ID: sess_abc123xyz
Knowledge Base ID: kb_12345
Max Rounds: 10
Created At: 2025-11-03T10:30:00Z
╚═══════════════════════════╝

[Session: sess_abc] > exit

Goodbye!
```

## Testing Scenarios

### 1. Basic Q&A
Test simple question-answering with knowledge retrieval:
```
> ask What is machine learning?
```

### 2. Multi-Turn Conversation
Test context awareness:
```
> ask What is supervised learning?
> ask Can you give me some examples?
> ask How does it differ from unsupervised learning?
```

### 3. Complex Queries
Test agent's reasoning capabilities:
```
> ask Compare and contrast supervised learning and reinforcement learning, including their use cases
```

### 4. Tool Usage
Test tool calling and reflection:
```
> ask Search for information about deep learning and summarize the key concepts
```

### 5. Edge Cases
Test error handling:
```
> ask [empty query]
> ask [very long query with 1000+ characters]
```

## Debugging

The tool displays detailed information for debugging:

1. **Event Timing**: Track how long each phase takes
2. **Tool Arguments**: See exactly what parameters are passed to tools
3. **Knowledge Scores**: Evaluate retrieval quality
4. **Error Messages**: Clear error reporting for failed operations
5. **Session State**: Monitor session configuration and status

## Troubleshooting

### Connection Issues
```
✗ Request failed: connection refused
```
- Check if WeKnora server is running
- Verify the URL is correct
- Check network connectivity

### Authentication Errors
```
✗ HTTP error 401: Unauthorized
```
- Verify your API token is valid
- Check token hasn't expired

### Session Not Found
```
✗ Failed to get session info: session not found
```
- Session may have been deleted
- Create a new session with `new` command

## Advanced Usage

### Using with Different Environments

Development:
```bash
./agent_test -url http://localhost:8080 -kb kb_dev
```

Staging:
```bash
./agent_test -url https://staging.weknora.com -token $STAGING_TOKEN -kb kb_staging
```

Production:
```bash
./agent_test -url https://api.weknora.com -token $PROD_TOKEN -kb kb_prod
```

### Scripting

You can pipe commands for automated testing:
```bash
echo -e "new\nask What is AI?\nexit" | ./agent_test -url http://localhost:8080 -kb kb_test
```

## License

Copyright (c) 2025 Tencent. All rights reserved.

