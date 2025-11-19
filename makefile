include .env


dev:
	sh -c 'set -a; . ./.env; set +a; gow run cmd/server/main.go'