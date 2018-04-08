SOURCE_FILES?=./pkg/...
TEST_PATTERN?=.
TEST_OPTIONS?=
OS=$(shell uname -s)

# Install all the build and lint dependencies
setup:
	go get -u golang.org/x/tools/cmd/stringer
	go get -u golang.org/x/tools/cmd/cover
	go get -u github.com/caarlos0/bandep
	go get -u gopkg.in/alecthomas/gometalinter.v2
ifeq ($(OS), Darwin)
	brew install dep
else
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
endif
	dep ensure
	gometalinter.v2 --install
	echo "make check" > .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
.PHONY: setup

check:
	bandep --ban github.com/tj/assert
.PHONY: check

# Run all the tests
test:
	go test $(TEST_OPTIONS) -failfast -race -coverpkg=./pkg/... -covermode=atomic -coverprofile=coverage.txt $(SOURCE_FILES) -run $(TEST_PATTERN) -timeout=2m
.PHONY: cover

# Run all the tests and opens the coverage report
cover: test
	go tool cover -html=coverage.txt
.PHONY: cover

# gofmt and goimports all go files
fmt:
	find . -name '*.go' -not -wholename './vendor/*' | while read -r file; do gofmt -w -s "$$file"; goimports -w "$$file"; done
	# find . -name '*.md' -not -wholename './vendor/*' | xargs prettier --write
.PHONY: fmt

# Run all the linters
lint:
	gometalinter.v2 --vendor ./...
	# find . -name '*.md' -not -wholename './vendor/*' | xargs prettier -l
.PHONY: lint

# Run all the tests and code checks
ci: build test lint
.PHONY: ci

# Build a beta version of goreleaser
build:
	#go generate ./...
	go build -o s3vn cmd/s3vn/main.go
.PHONY: build


# Show to-do items per file.
todo:
	@grep \
		--exclude-dir=vendor \
		--exclude-dir=node_modules \
		--exclude=Makefile \
		--text \
		--color \
		-nRo -E ' TODO:.*|SkipNow' .
.PHONY: todo

