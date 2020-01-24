#!/bin/bash
SERVER="http://localhost:8080"
if [ "$#" -gt 0 ]
then
  echo "Server assumed to be running on $1"
  SERVER="$1"
else
  DATABASE_HOST=localhost \
  DATABASE_NAME=depsets \
  DATABASE_USER=depsets_robot \
  DATABASE_PASSWORD="d3p53t5" \
  ../cmd/depsets/depsets &
  SERVER_PID=$!
  echo "Server Started"
  sleep 1
fi

echo "Sending command:"
tmpfile=$(mktemp /tmp/integration-test.XXXXXX)
echo curl -v -X POST -H "Accept: application/json" -H "Content-Type: application/json" -d @addsinglemodule.json -o $tmpfile ${SERVER}/orgs/org1/apps/app1/sets/0
curl -s -X POST -H "Accept: application/json" -H "Content-Type: application/json" -d @addsinglemodule.json -o $tmpfile ${SERVER}/orgs/org1/apps/app1/sets/0
echo
id=$(cat $tmpfile | sed 's/"//g')
echo "Output: >${id}<"

echo curl -o $tmpfile -H "Accept: application/json"  ${SERVER}/orgs/org1/apps/app1/sets/$id
curl -s -o $tmpfile -H "Accept: application/json" ${SERVER}/orgs/org1/apps/app1/sets/$id
echo
echo "Output:"
cat $tmpfile
echo

sleep 1s

rm $tmpfile
if [ ! -z "$SERVER_PID" ]
then
  kill $SERVER_PID
fi
