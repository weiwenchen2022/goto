#!/bin/sh

make

./goto -http=:8080 -rpc=true &
masterPid=$!

sleep 1

./goto -master=:8080 -http=:8081 -host=localhost:8081 &
slavePid=$!

echo "Running master on :8080, slave on :8081."
echo "Visit: http://localhost:8081/add"
echo "Press enter to shutdown"

read
kill $masterPid
kill $slavePid
