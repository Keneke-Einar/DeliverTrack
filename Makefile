.PHONY: run test migrate test-integration test-coverage

run:
	# TODO: Add run command

test:
	go test ./...

migrate:
	# TODO: Add migration command

test-integration:
	# TODO: Add integration test command

test-coverage:
	go test -cover ./...
