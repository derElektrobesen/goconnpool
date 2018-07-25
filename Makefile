fmt:
	find . \
		-path ./vendor \
		-prune -o \
		-name '*.go' \
		-exec bash -c 'gofmt -w {}' \; \
		-exec bash -c 'goimports -w {}' \;

test:
