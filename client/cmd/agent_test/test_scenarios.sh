#!/bin/bash

# Test Scenarios for WeKnora Agent QA
# This script contains various test scenarios you can run

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_scenario() {
    echo -e "\n${BLUE}═══════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════${NC}\n"
}

# Check if binary exists
if [ ! -f "./agent_test" ]; then
    echo "Building agent_test..."
    make build
fi

# Load environment variables if .env exists
if [ -f ".env" ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

# Check required variables
if [ -z "$WEKNORA_KB_ID" ]; then
    echo "Error: WEKNORA_KB_ID is not set"
    echo "Please set it in .env file or export WEKNORA_KB_ID=your_kb_id"
    exit 1
fi

URL="${WEKNORA_URL:-http://localhost:8080}"
KB_ID="$WEKNORA_KB_ID"
TOKEN="${WEKNORA_TOKEN:-}"

# Scenario 1: Basic Q&A
scenario_basic() {
    print_scenario "Scenario 1: Basic Question & Answer"
    echo -e "${YELLOW}This will test basic agent Q&A functionality${NC}\n"
    echo "Try asking: What is machine learning?"
    echo ""
    
    if [ -n "$TOKEN" ]; then
        ./agent_test -url "$URL" -kb "$KB_ID" -token "$TOKEN"
    else
        ./agent_test -url "$URL" -kb "$KB_ID"
    fi
}

# Scenario 2: Complex query with tool usage
scenario_complex() {
    print_scenario "Scenario 2: Complex Query with Tool Usage"
    echo -e "${YELLOW}This will test agent's ability to use tools${NC}\n"
    echo "Try asking: Search for information about neural networks and explain the key concepts"
    echo ""
    
    if [ -n "$TOKEN" ]; then
        ./agent_test -url "$URL" -kb "$KB_ID" -token "$TOKEN"
    else
        ./agent_test -url "$URL" -kb "$KB_ID"
    fi
}

# Scenario 3: Multi-turn conversation
scenario_multiturn() {
    print_scenario "Scenario 3: Multi-turn Conversation"
    echo -e "${YELLOW}This will test agent's context awareness${NC}\n"
    echo "Try this sequence:"
    echo "  1. What is supervised learning?"
    echo "  2. Can you give me some examples?"
    echo "  3. How does it compare to unsupervised learning?"
    echo ""
    
    if [ -n "$TOKEN" ]; then
        ./agent_test -url "$URL" -kb "$KB_ID" -token "$TOKEN"
    else
        ./agent_test -url "$URL" -kb "$KB_ID"
    fi
}

# Show menu
show_menu() {
    echo -e "\n${GREEN}WeKnora Agent Test Scenarios${NC}\n"
    echo "Select a test scenario to run:"
    echo "  1) Basic Q&A"
    echo "  2) Complex Query with Tool Usage"
    echo "  3) Multi-turn Conversation"
    echo "  4) Custom (manual testing)"
    echo "  5) Exit"
    echo ""
    read -p "Enter your choice [1-5]: " choice
    
    case $choice in
        1) scenario_basic ;;
        2) scenario_complex ;;
        3) scenario_multiturn ;;
        4) 
            print_scenario "Custom Testing"
            if [ -n "$TOKEN" ]; then
                ./agent_test -url "$URL" -kb "$KB_ID" -token "$TOKEN"
            else
                ./agent_test -url "$URL" -kb "$KB_ID"
            fi
            ;;
        5) 
            echo "Goodbye!"
            exit 0
            ;;
        *)
            echo "Invalid choice. Please try again."
            show_menu
            ;;
    esac
}

# Main
if [ "$1" == "--help" ] || [ "$1" == "-h" ]; then
    echo "Usage: $0 [scenario_number]"
    echo ""
    echo "Available scenarios:"
    echo "  1 - Basic Q&A"
    echo "  2 - Complex Query with Tool Usage"
    echo "  3 - Multi-turn Conversation"
    echo "  4 - Custom (manual testing)"
    echo ""
    echo "If no argument is provided, an interactive menu will be shown."
    exit 0
fi

# If scenario number is provided as argument
if [ -n "$1" ]; then
    case $1 in
        1) scenario_basic ;;
        2) scenario_complex ;;
        3) scenario_multiturn ;;
        4) 
            if [ -n "$TOKEN" ]; then
                ./agent_test -url "$URL" -kb "$KB_ID" -token "$TOKEN"
            else
                ./agent_test -url "$URL" -kb "$KB_ID"
            fi
            ;;
        *)
            echo "Invalid scenario number: $1"
            echo "Use --help to see available scenarios"
            exit 1
            ;;
    esac
else
    # Show interactive menu
    show_menu
fi

