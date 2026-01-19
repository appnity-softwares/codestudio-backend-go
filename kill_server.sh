#!/bin/bash
lsof -ti :8080 | xargs kill -9
echo "Killed process on port 8080"
