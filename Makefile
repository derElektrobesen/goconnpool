fmt:
	find . \
		-path ./vendor \
		-prune -o \
		-name '*.go' \
		-exec bash -c 'gofmt -w {}' \; \
		-exec bash -c 'goimports -w {}' \;

test:
	go test -race -v ./...

gen:
	mockgen -source=server.go -package=goconnpool connectionProvider > server_mock_test.go
	mockgen -source=dialer.go -package=goconnpool Dialer > dialer_mock_test.go
	mockgen -source=server_test.go -package=goconnpool closer > closer_mock_test.go

lint:
	golangci-lint run
