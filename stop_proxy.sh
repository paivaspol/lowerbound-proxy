#!/usr/bin/env bash

ps aux \
	| grep "go" \
	| awk '{ print $2 }' \
	| xargs kill -9

echo "Result: ps aux | grep \"go\""
ps aux | grep "go" \
