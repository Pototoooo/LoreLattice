package client

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ExampleAgentQABasic demonstrates basic agent QA usage
func ExampleAgentQABasic() {
	// Create client
	client := NewClient("http://localhost:8080", WithTimeout(60*time.Second))

	// Create a session
	ctx := context.Background()
	session, err := client.CreateSession(ctx, &CreateSessionRequest{
		KnowledgeBaseID: "your_kb_id",
		SessionStrategy: &SessionStrategy{
			MaxRounds:        10,
			EnableRewrite:    false,
			FallbackStrategy: "default",
			EmbeddingTopK:    10,
		},
	})
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	fmt.Printf("Session created: %s\n", session.ID)

	// Create agent session wrapper
	agentSession := client.NewAgentSession(session.ID)

	// Ask a question
	err = agentSession.Ask(ctx, "What is machine learning?", func(resp *AgentStreamResponse) error {
		switch resp.ResponseType {
		case AgentResponseTypeThinking:
			if resp.Done {
				fmt.Printf("Thinking: %s\n", resp.Content)
			}
		case AgentResponseTypeAnswer:
			fmt.Print(resp.Content) // Print answer incrementally
			if resp.Done {
				fmt.Println() // New line at the end
			}
		}
		return nil
	})

	if err != nil {
		log.Fatalf("Agent QA failed: %v", err)
	}
}

// ExampleAgentQAWithToolTracking demonstrates tracking tool calls
func ExampleAgentQAWithToolTracking() {
	client := NewClient("http://localhost:8080")
	agentSession := client.NewAgentSession("existing_session_id")

	ctx := context.Background()
	toolCalls := make([]string, 0)

	err := agentSession.Ask(ctx, "Search for information about neural networks", func(resp *AgentStreamResponse) error {
		switch resp.ResponseType {
		case AgentResponseTypeToolCall:
			if resp.Data != nil {
				if toolName, ok := resp.Data["tool_name"].(string); ok {
					toolCalls = append(toolCalls, toolName)
					fmt.Printf("Tool called: %s\n", toolName)
				}
			}
		case AgentResponseTypeToolResult:
			fmt.Printf("Tool result: %s\n", resp.Content)
		case AgentResponseTypeAnswer:
			if resp.Done {
				fmt.Printf("Final answer: %s\n", resp.Content)
			}
		}
		return nil
	})

	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("Total tool calls: %d\n", len(toolCalls))
}

// ExampleAgentQAWithReferences demonstrates capturing knowledge references
func ExampleAgentQAWithReferences() {
	client := NewClient("http://localhost:8080")
	agentSession := client.NewAgentSession("existing_session_id")

	ctx := context.Background()
	var references []*SearchResult
	var finalAnswer string

	err := agentSession.Ask(ctx, "Explain deep learning", func(resp *AgentStreamResponse) error {
		switch resp.ResponseType {
		case AgentResponseTypeReferences:
			if resp.KnowledgeReferences != nil {
				references = append(references, resp.KnowledgeReferences...)
				fmt.Printf("Found %d references\n", len(resp.KnowledgeReferences))
			}
		case AgentResponseTypeAnswer:
			finalAnswer += resp.Content
		}
		return nil
	})

	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Process results
	fmt.Println("Final Answer:", finalAnswer)
	fmt.Printf("\nBased on %d knowledge references:\n", len(references))
	for i, ref := range references {
		fmt.Printf("%d. [Score: %.3f] %s\n", i+1, ref.Score, ref.KnowledgeTitle)
	}
}

// ExampleAgentQAWithFullEventTracking demonstrates comprehensive event tracking
func ExampleAgentQAWithFullEventTracking() {
	client := NewClient("http://localhost:8080")
	agentSession := client.NewAgentSession("existing_session_id")

	ctx := context.Background()

	// Tracking state
	state := struct {
		thinking    string
		toolCalls   []map[string]interface{}
		toolResults []string
		references  []*SearchResult
		answer      string
		reflections []string
		errors      []string
	}{}

	err := agentSession.Ask(ctx, "Your question here", func(resp *AgentStreamResponse) error {
		switch resp.ResponseType {
		case AgentResponseTypeThinking:
			state.thinking += resp.Content
			if resp.Done {
				fmt.Printf("💭 Thinking: %s\n", state.thinking)
			}

		case AgentResponseTypeToolCall:
			toolCall := map[string]interface{}{
				"name":      resp.Data["tool_name"],
				"arguments": resp.Data["arguments"],
			}
			state.toolCalls = append(state.toolCalls, toolCall)
			fmt.Printf("🔧 Tool Call: %v\n", resp.Data["tool_name"])

		case AgentResponseTypeToolResult:
			state.toolResults = append(state.toolResults, resp.Content)
			success := "✓"
			if resp.Data != nil {
				if s, ok := resp.Data["success"].(bool); ok && !s {
					success = "✗"
				}
			}
			fmt.Printf("%s Tool Result: %s\n", success, resp.Content)

		case AgentResponseTypeReferences:
			if resp.KnowledgeReferences != nil {
				state.references = append(state.references, resp.KnowledgeReferences...)
				fmt.Printf("📚 References: %d\n", len(resp.KnowledgeReferences))
			}

		case AgentResponseTypeAnswer:
			state.answer += resp.Content
			if resp.Done {
				fmt.Printf("📝 Answer: %s\n", state.answer)
			}

		case AgentResponseTypeReflection:
			if resp.Done {
				state.reflections = append(state.reflections, resp.Content)
				fmt.Printf("🤔 Reflection: %s\n", resp.Content)
			}

		case AgentResponseTypeError:
			state.errors = append(state.errors, resp.Content)
			fmt.Printf("❌ Error: %s\n", resp.Content)
		}

		return nil
	})

	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Print summary
	fmt.Println("\n=== Summary ===")
	fmt.Printf("Tool Calls: %d\n", len(state.toolCalls))
	fmt.Printf("References: %d\n", len(state.references))
	fmt.Printf("Reflections: %d\n", len(state.reflections))
	fmt.Printf("Errors: %d\n", len(state.errors))
}

// ExampleAgentQAWithCustomErrorHandling demonstrates custom error handling
func ExampleAgentQAWithCustomErrorHandling() {
	client := NewClient("http://localhost:8080")
	agentSession := client.NewAgentSession("existing_session_id")

	ctx := context.Background()
	hasError := false

	err := agentSession.Ask(ctx, "Your question", func(resp *AgentStreamResponse) error {
		if resp.ResponseType == AgentResponseTypeError {
			hasError = true
			// Log error to your monitoring system
			fmt.Printf("Agent error occurred: %s\n", resp.Content)
			// You can return an error here to stop processing
			// return fmt.Errorf("stopping due to error: %s", resp.Content)
		}
		return nil
	})

	if err != nil {
		log.Printf("Stream error: %v", err)
	}

	if hasError {
		log.Println("Agent encountered errors during processing")
	}
}

// ExampleAgentQAStreamCancellation demonstrates canceling a stream
func ExampleAgentQAStreamCancellation() {
	client := NewClient("http://localhost:8080")
	agentSession := client.NewAgentSession("existing_session_id")

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Or cancel based on a condition
	maxToolCalls := 5
	toolCallCount := 0

	err := agentSession.Ask(ctx, "Complex query that might take long", func(resp *AgentStreamResponse) error {
		if resp.ResponseType == AgentResponseTypeToolCall {
			toolCallCount++
			if toolCallCount > maxToolCalls {
				// Stop processing if too many tool calls
				return fmt.Errorf("too many tool calls: %d", toolCallCount)
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("Stream stopped: %v", err)
	}
}

// ExampleMultipleAgentSessions demonstrates managing multiple agent sessions
func ExampleMultipleAgentSessions() {
	client := NewClient("http://localhost:8080")

	ctx := context.Background()

	// Create multiple sessions for different purposes
	sessions := map[string]*AgentSession{
		"technical": client.NewAgentSession("session_id_1"),
		"general":   client.NewAgentSession("session_id_2"),
		"customer":  client.NewAgentSession("session_id_3"),
	}

	// Use appropriate session based on query type
	queryType := "technical"
	query := "How does gradient descent work?"

	if session, ok := sessions[queryType]; ok {
		err := session.Ask(ctx, query, func(resp *AgentStreamResponse) error {
			if resp.ResponseType == AgentResponseTypeAnswer && resp.Done {
				fmt.Printf("[%s] Answer: %s\n", queryType, resp.Content)
			}
			return nil
		})
		if err != nil {
			log.Printf("Error in %s session: %v", queryType, err)
		}
	}
}
