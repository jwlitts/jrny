#!/bin/sh
set -e #exit on error
set -u #fail on use of unset variable
set -x #print commands as they are executed
GOOS=linux go build .
GOOS=windows go build .
