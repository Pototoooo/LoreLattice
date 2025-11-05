#!/bin/bash

# WeKnora Agent Test Tool - Quick Start Script
# This script helps you quickly build and run the agent test tool

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
DEFAULT_URL="http://localhost:8080"
BINARY_NAME="agent_test"

# Function to print colored output
print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

# Function to build the binary
build() {
    print_info "Building agent test tool..."
    if go build -o "$BINARY_NAME" .; then
        print_success "Build successful!"
        return 0
    else
        print_error "Build failed!"
        return 1
    fi
}

# Function to show usage
usage() {
    cat << EOF
WeKnora Agent Test Tool - Quick Start Script

Usage:
    $0 [OPTIONS]

Options:
    -u, --url URL          WeKnora API URL (default: $DEFAULT_URL)
    -k, --kb KB_ID         Knowledge Base ID
    -s, --session ID       Existing Session ID
    -t, --token TOKEN      API authentication token
    -b, --build-only       Only build, don't run
    -h, --help             Show this help message

Examples:
    # Build and run with a knowledge base
    $0 -k kb_123456

    # Use a different server URL
    $0 -u https://api.weknora.com -k kb_123456 -t mytoken

    # Continue an existing session
    $0 -s sess_123456

    # Only build the binary
    $0 --build-only

Environment Variables:
    WEKNORA_URL            Default API URL
    WEKNORA_KB_ID          Default Knowledge Base ID
    WEKNORA_TOKEN          Default API Token
    WEKNORA_SESSION_ID     Default Session ID

EOF
}

# Parse command line arguments
URL="${WEKNORA_URL:-$DEFAULT_URL}"
KB_ID="${WEKNORA_KB_ID:-}"
SESSION_ID="${WEKNORA_SESSION_ID:-}"
TOKEN="${WEKNORA_TOKEN:-}"
BUILD_ONLY=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -u|--url)
            URL="$2"
            shift 2
            ;;
        -k|--kb)
            KB_ID="$2"
            shift 2
            ;;
        -s|--session)
            SESSION_ID="$2"
            shift 2
            ;;
        -t|--token)
            TOKEN="$2"
            shift 2
            ;;
        -b|--build-only)
            BUILD_ONLY=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Build the binary
if ! build; then
    exit 1
fi

# Exit if build-only mode
if [ "$BUILD_ONLY" = true ]; then
    print_info "Build-only mode. Binary created: ./$BINARY_NAME"
    exit 0
fi

# Validate parameters
if [ -z "$SESSION_ID" ] && [ -z "$KB_ID" ]; then
    print_warning "Neither session ID nor knowledge base ID provided."
    print_info "You can create a new session after starting the tool."
fi

# Build command arguments
ARGS="-url $URL"

if [ -n "$KB_ID" ]; then
    ARGS="$ARGS -kb $KB_ID"
fi

if [ -n "$SESSION_ID" ]; then
    ARGS="$ARGS -session $SESSION_ID"
fi

if [ -n "$TOKEN" ]; then
    ARGS="$ARGS -token $TOKEN"
fi

# Show configuration
echo ""
print_info "Configuration:"
echo "  URL: $URL"
[ -n "$KB_ID" ] && echo "  Knowledge Base: $KB_ID"
[ -n "$SESSION_ID" ] && echo "  Session: $SESSION_ID"
[ -n "$TOKEN" ] && echo "  Token: [provided]"
echo ""

# Run the tool
print_info "Starting agent test tool..."
echo ""
./$BINARY_NAME $ARGS

