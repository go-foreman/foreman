#! /usr/bin/make


UNIT_TEST_PKGS=`go list ./... | grep -v -E './testing'`
COVERAGE_PKGS=`go list  ./... | grep -v -E './testing' | tr "\n" ',' | rev | cut -c 2- | rev`
INTEGRATION_TEST_PKGS=`go list ./... | grep "testing/integration"`

INTEGRATION_TEST_PATH?=./.../testing

# set of env variables that you need for testing
ENV_LOCAL_TEST=\
  POSTGRES_PASSWORD=foreman \
  POSTGRES_DB=foreman \
  POSTGRES_HOST=postgres \
  POSTGRES_USER=foreman \
  MYSQL_ADDRESS=127.0.0.1:3306\
  MYSQL_DB=foreman \
  MYSQL_USER=foreman \
  MYSQL_PASSWORD=foreman

#CI_REPORTS_DIR ?= reports

.PHONY: tools
tools:
	go install github.com/sonatype-nexus-community/nancy@latest
	go install github.com/golang/mock/mockgen@v1.6.0
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.6

.PHONY: docker-start
docker-start:
	@docker compose up -d
	@echo "---Docker compose started"

# shutting down docker components
.PHONY: docker-stop
docker-stop:
	@docker compose down
	@echo "---Docker compose stopped"

.PHONY: docker-clean
docker-clean:
	@docker volume prune --force
	@echo "---Docker env cleaned"

.PHONY: testsuite-clean
testsuite-clean: docker-clean
	@echo "---Test suite cleaned"

.PHONY: testsuite-start
testsuite-start: docker-start
	@echo "---Test suite started and ready"

.PHONY: testsuite-stop
testsuite-stop: docker-stop docker-clean
	@echo "---Testsuite stopped"

.PHONY: testsuite-clean
testsuite-clean: docker-clean

.PHONY: test
test:
	go test ./... -cover -race

.PHONY: unit-test
unit-test:
	go test -race -coverprofile=coverage.txt -covermode=atomic -coverpkg=$(COVERAGE_PKGS) $(UNIT_TEST_PKGS)

integration-test:
	go test -race $(INTEGRATION_TEST_PKGS)

.PHONY: lint
lint:
	golangci-lint run -v

.PHONY: lint-report
lint-report:
	golangci-lint run -v --issues-exit-code=0 --out-format checkstyle > lint-report.xml

.PHONY: create_reports_dir
create_reports_dir:
	mkdir -p $(CI_REPORTS_DIR)

.PHONY: mod-download
mod-download:
	go mod download

.PHONY: check-mods
check-mods:
	go list -json -m all | nancy sleuth

.PHONY: generate
generate:
	go generate ./...
