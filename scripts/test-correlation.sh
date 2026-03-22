#!/bin/bash

# LocalFinance Correlation ID Testing Script
# Tests the correlation ID flow across all services

echo "🔗 LocalFinance Correlation ID Testing"
echo "======================================"

# Configuration
BASE_URL="http://10.0.0.16:8080"
TEST_FILE="/tmp/test-statement.csv"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

# Create test CSV file
create_test_file() {
    log "📄 Creating test CSV file..."
    
    cat > "$TEST_FILE" << 'EOF'
Date,Description,Amount
2024-01-01,"Grocery Store",-45.67
2024-01-02,"Salary Deposit",2500.00
2024-01-03,"Gas Station",-32.15
EOF

    echo "✅ Test file created: $TEST_FILE"
}

# Test user registration/login
test_authentication() {
    log "🔐 Testing authentication..."
    
    # Generate unique test user
    TIMESTAMP=$(date +%s)
    TEST_EMAIL="test-correlation-${TIMESTAMP}@example.com"
    TEST_PASSWORD="test123456"
    
    info "📝 Registering user: $TEST_EMAIL"
    
    # Register user
    REGISTER_RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}\nCORRELATION_ID:%{header_x-correlation-id}" \
        -X POST "$BASE_URL/api/auth/register" \
        -H "Content-Type: application/json" \
        -d "{
            \"email\": \"$TEST_EMAIL\",
            \"password\": \"$TEST_PASSWORD\",
            \"first_name\": \"Test\",
            \"last_name\": \"User\"
        }")
    
    # Parse response
    HTTP_CODE=$(echo "$REGISTER_RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
    CORRELATION_ID=$(echo "$REGISTER_RESPONSE" | grep "CORRELATION_ID:" | cut -d: -f2)
    REGISTER_BODY=$(echo "$REGISTER_RESPONSE" | sed '/HTTP_CODE:/,$d')
    
    if [[ "$HTTP_CODE" == "200" ]]; then
        echo "✅ Registration successful"
        echo "🔗 Correlation ID: $CORRELATION_ID"
        
        # Extract token
        TOKEN=$(echo "$REGISTER_BODY" | python3 -c "import json,sys; print(json.load(sys.stdin)['token'])" 2>/dev/null)
        
        if [[ -n "$TOKEN" ]]; then
            echo "🎫 Token obtained: ${TOKEN:0:20}..."
            export AUTH_TOKEN="$TOKEN"
            export CORRELATION_ID_1="$CORRELATION_ID"
        else
            error "Failed to extract token from registration response"
            return 1
        fi
    else
        error "Registration failed with HTTP $HTTP_CODE"
        echo "$REGISTER_BODY"
        return 1
    fi
}

# Test file upload with correlation ID tracking
test_upload() {
    log "📤 Testing file upload with correlation ID tracking..."
    
    if [[ -z "$AUTH_TOKEN" ]]; then
        error "No auth token available. Run authentication test first."
        return 1
    fi
    
    info "📁 Uploading: $TEST_FILE"
    
    # Upload file
    UPLOAD_RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}\nCORRELATION_ID:%{header_x-correlation-id}" \
        -X POST "$BASE_URL/api/v1/upload" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -F "file=@$TEST_FILE")
    
    # Parse response
    HTTP_CODE=$(echo "$UPLOAD_RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
    CORRELATION_ID=$(echo "$UPLOAD_RESPONSE" | grep "CORRELATION_ID:" | cut -d: -f2)
    UPLOAD_BODY=$(echo "$UPLOAD_RESPONSE" | sed '/HTTP_CODE:/,$d')
    
    if [[ "$HTTP_CODE" == "200" ]]; then
        echo "✅ Upload successful"
        echo "🔗 Correlation ID: $CORRELATION_ID"
        
        # Extract document ID
        DOCUMENT_ID=$(echo "$UPLOAD_BODY" | python3 -c "import json,sys; print(json.load(sys.stdin)['document_id'])" 2>/dev/null)
        
        if [[ -n "$DOCUMENT_ID" ]]; then
            echo "📄 Document ID: $DOCUMENT_ID"
            export DOCUMENT_ID="$DOCUMENT_ID"
            export CORRELATION_ID_2="$CORRELATION_ID"
        fi
        
        # Show response body
        echo "📋 Response:"
        echo "$UPLOAD_BODY" | python3 -m json.tool 2>/dev/null || echo "$UPLOAD_BODY"
        
    else
        error "Upload failed with HTTP $HTTP_CODE"
        echo "$UPLOAD_BODY"
        return 1
    fi
}

# Test processing status tracking
test_processing_status() {
    log "📊 Testing processing status with correlation ID..."
    
    if [[ -z "$DOCUMENT_ID" ]]; then
        error "No document ID available. Run upload test first."
        return 1
    fi
    
    info "🔄 Checking processing status for: $DOCUMENT_ID"
    
    # Check status (will show correlation ID for each request)
    for i in {1..5}; do
        echo "📡 Status check #$i:"
        
        STATUS_RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}\nCORRELATION_ID:%{header_x-correlation-id}" \
            -X GET "$BASE_URL/api/v1/processing-status/$DOCUMENT_ID" \
            -H "Authorization: Bearer $AUTH_TOKEN")
        
        # Parse response
        HTTP_CODE=$(echo "$STATUS_RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
        CORRELATION_ID=$(echo "$STATUS_RESPONSE" | grep "CORRELATION_ID:" | cut -d: -f2)
        STATUS_BODY=$(echo "$STATUS_RESPONSE" | sed '/HTTP_CODE:/,$d')
        
        echo "   🔗 Correlation ID: $CORRELATION_ID"
        echo "   📊 Status: $(echo "$STATUS_BODY" | python3 -c "import json,sys; print(json.load(sys.stdin).get('status', 'unknown'))" 2>/dev/null)"
        
        # Check if processing is complete
        STATUS=$(echo "$STATUS_BODY" | python3 -c "import json,sys; print(json.load(sys.stdin).get('status', ''))" 2>/dev/null)
        
        if [[ "$STATUS" == "completed" || "$STATUS" == "error" ]]; then
            echo "   ✅ Processing finished with status: $STATUS"
            break
        fi
        
        sleep 2
    done
}

# Show correlation ID summary
show_correlation_summary() {
    log "📈 Correlation ID Summary"
    echo "========================"
    
    echo "🔗 Registration:  ${CORRELATION_ID_1:-'N/A'}"
    echo "🔗 Upload:        ${CORRELATION_ID_2:-'N/A'}"
    echo ""
    echo "📋 How to trace these requests in logs:"
    echo ""
    
    if [[ -n "$CORRELATION_ID_1" ]]; then
        echo "# Registration flow:"
        echo "grep '$CORRELATION_ID_1' /home/sagar/localfinance/logs/enhanced.log"
    fi
    
    if [[ -n "$CORRELATION_ID_2" ]]; then
        echo "# Upload flow:"  
        echo "grep '$CORRELATION_ID_2' /home/sagar/localfinance/logs/enhanced.log"
    fi
    
    echo ""
    echo "📊 Each correlation ID will show:"
    echo "   • Request initiation (Hermes)"
    echo "   • Business logic (Thesaurus)"  
    echo "   • Database operations"
    echo "   • Service calls between microservices"
    echo "   • Processing results (Logos)"
    echo "   • Response delivery"
}

# Main execution
main() {
    create_test_file
    
    if test_authentication; then
        if test_upload; then
            test_processing_status
        fi
    fi
    
    echo ""
    show_correlation_summary
    
    # Cleanup
    rm -f "$TEST_FILE"
    
    echo ""
    log "🎯 Correlation ID testing complete!"
    echo ""
    echo "💡 Next steps:"
    echo "   1. Check server logs with the correlation IDs above"
    echo "   2. See how each service logs the same request"
    echo "   3. Trace the complete request flow end-to-end"
    echo ""
    echo "🔗 Your LocalFinance now has full request traceability!"
}

# Check if server is running
if ! curl -s "$BASE_URL/health" > /dev/null; then
    error "LocalFinance server not responding at $BASE_URL"
    echo "Please ensure the enhanced LocalFinance is running on the Jetson"
    exit 1
fi

# Run main function
main "$@"