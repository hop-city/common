
test-all:
	go test **.*

test-auth:
	go test ./rest/auth/*.go