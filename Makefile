SRC := $(shell find . -name "*.go" -not -path "./vendor/*" -not -name "*_test.go")
TESTS := $(shell find . -name "*_test.go" -not -path "./vendor/*")

BINARY=pit

format: $(SRC) $(TESTS)
	@goimports -w $?
	@gofmt -s -w $?

test: build $(TESTS)
	@./bin/pit

build: bin/$(BINARY)
bin/$(BINARY): $(SRC)
	@go build -o $@ ./cmd/$(BINARY)
	
deps:
	@dep ensure
	
clean:
	@rm -v bin/*; trap

.PHONY: format test build deps clean