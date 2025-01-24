#!/bin/bash

# Sestavení a spuštění testovacího prostředí
docker-compose -f docker-compose.test.yml up --build -d

# Čekání na inicializaci databází
echo "Čekání na inicializaci databází..."
sleep 10

# Spuštění testů
docker-compose -f docker-compose.test.yml run test-runner

# Úklid
docker-compose -f docker-compose.test.yml down 