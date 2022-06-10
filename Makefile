#! /usr/bin/make


UNIT_TEST_PKGS=`go list ./... | grep -v -E './testing'`
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
	go install github.com/vektra/mockery/v2@latest
	go install github.com/sonatype-nexus-community/nancy@latest
	go install github.com/golang/mock/mockgen@v1.6.0
	## using wget because go get is not working for 1.40.1
	wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | BINDIR=${GOPATH}/bin sh -s v1.46.2

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
	go test ./... -cover

.PHONY: test-report
test-report:
	#go test -coverprofile=$(CI_REPORTS_DIR)/coverage.txt -covermode=atomic -json $(UNIT_TEST_PKGS) > $(CI_REPORTS_DIR)/report.json
	go test -coverprofile=coverage.txt -covermode=atomic $(UNIT_TEST_PKGS)

integration-test:
	go test $(INTEGRATION_TEST_PKGS)

.PHONY: lint
lint:
	golangci-lint run -v

.PHONY: lint-report
lint-report:
	golangci-lintgolangci-lint run -v --issues-exit-code=0 --out-format checkstyle > lint-report.xml

.PHONY: create_reports_dir
create_reports_dir:
	mkdir -p $(CI_REPORTS_DIR)

.PHONY: mod-download
mod-download:
	go mod download

.PHONY: check-mods
check-mods:
	go list -json -m all | nancy sleuth

generate:
	go generate ./...
