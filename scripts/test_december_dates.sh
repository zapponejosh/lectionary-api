#!/bin/bash
# Test all December dates with the readings API

BASE_URL="http://localhost:8080/api/v1/readings/date"
YEAR=2025

echo "Testing all December dates for $YEAR..."
echo "=========================================="
echo ""

success_count=0
error_count=0
errors=()

for day in {1..31}; do
    date=$(printf "%d-12-%02d" $YEAR $day)
    url="$BASE_URL/$date"
    
    echo -n "Testing $date... "
    
    response=$(curl -s -w "\n%{http_code}" "$url")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "200" ]; then
        period=$(echo "$body" | jq -r '.data.period // "N/A"' 2>/dev/null)
        if [ "$period" != "null" ] && [ "$period" != "" ]; then
            echo "✓ $period"
            ((success_count++))
        else
            echo "✗ No period found"
            ((error_count++))
            errors+=("$date: No period in response")
        fi
    else
        error_msg=$(echo "$body" | jq -r '.error.message // "Unknown error"' 2>/dev/null)
        echo "✗ HTTP $http_code: $error_msg"
        ((error_count++))
        errors+=("$date: HTTP $http_code - $error_msg")
    fi
done

echo ""
echo "=========================================="
echo "Summary:"
echo "  Success: $success_count"
echo "  Errors:  $error_count"
echo ""

if [ $error_count -gt 0 ]; then
    echo "Errors:"
    for error in "${errors[@]}"; do
        echo "  - $error"
    done
fi

