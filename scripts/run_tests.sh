#!/bin/bash

docker-compose -f docker-compose.test.yml up --build -d

echo "Waiting for database initialization."
sleep 10

docker-compose -f docker-compose.test.yml run test-runner

docker-compose -f docker-compose.test.yml down