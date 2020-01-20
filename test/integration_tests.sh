#!/bin/bash

DATABASE_HOST=localhost \
DATABASE_NAME=depsets \
DATABASE_USER=depsets_robot \
DATABASE_PASSWORD="d3p53t5" \
../cmd/depset/depset &
SERVER_PID=$!

echo "Server Started"

sleep 1

echo "Sending command:"
tmpfile=$(mktemp /tmp/integration-test.XXXXXX)
echo curl -v -X POST -d @addsinglemodule.json -o $tmpfile http://localhost:8080/orgs/org1/apps/app1/sets/0
curl -s -X POST -d @addsinglemodule.json -o $tmpfile http://localhost:8080/orgs/org1/apps/app1/sets/0
echo
id=$(cat $tmpfile | sed 's/"//g')
echo "Output: >${id}<"

echo curl -o $tmpfile http://localhost:8080/orgs/org1/apps/app1/sets/$id
curl -s -o $tmpfile http://localhost:8080/orgs/org1/apps/app1/sets/$id
echo
echo "Output:"
cat $tmpfile
echo

sleep 1

rm $tmpfile
kill $SERVER_PID
