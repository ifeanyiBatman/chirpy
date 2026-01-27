#!/bin/bash

# Base URL
URL="http://localhost:8080"

echo "🧪 Starting Chirpy API Tests..."
echo "--------------------------------"

# 1. Reset Database
echo "1. Resetting Database..."
curl -X POST $URL/admin/reset
echo -e "\n"

# 2. Create User
echo "2. Creating User..."
USER_RESP=$(curl -s -X POST $URL/api/users -d '{"email":"test@example.com"}')
echo $USER_RESP
USER_ID=$(echo $USER_RESP | grep -o '"id":"[^"]*' | cut -d'"' -f4)
echo "   -> User ID: $USER_ID"
echo ""

# 3. Create Valid Chirp
echo "3. Creating Valid Chirp..."
CHIRP_RESP=$(curl -s -X POST $URL/api/chirps -d "{\"body\": \"This is a clean chirp\", \"user_id\": \"$USER_ID\"}")
echo $CHIRP_RESP
CHIRP_ID=$(echo $CHIRP_RESP | grep -o '"id":"[^"]*' | cut -d'"' -f4)
echo "   -> Chirp ID: $CHIRP_ID"
echo -e "\n"

# 4. Create Profane Chirp
echo "4. Creating Profane Chirp (expect filter)..."
curl -X POST $URL/api/chirps -d "{\"body\": \"This is a kerfuffle chirp\", \"user_id\": \"$USER_ID\"}"
echo -e "\n"

# 5. Create Invalid Chirp (Too Long)
echo "5. Creating Too Long Chirp (expect error)..."
LONG_BODY=$(printf 'a%.0s' {1..141})
curl -X POST $URL/api/chirps -d "{\"body\": \"$LONG_BODY\", \"user_id\": \"$USER_ID\"}"
echo -e "\n"

# 6. Get All Chirps
echo "6. Getting All Chirps..."
curl -s -X GET $URL/api/chirps | python3 -m json.tool
echo -e "\n"

# 7. Get Single Chirp
echo "7. Getting Single Chirp..."
if [ -n "$CHIRP_ID" ]; then
    curl -s -X GET $URL/api/chirps/$CHIRP_ID | python3 -m json.tool
else
    echo "Skipping Get Single Chirp (No Chirp ID captured)"
fi
echo -e "\n"

# 8. Result Database
echo "8. Create Final Reset..."
curl -X POST $URL/admin/reset
echo -e "\n"

echo "--------------------------------"
echo "✅ Tests Completed!"
