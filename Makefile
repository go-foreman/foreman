#! /usr/bin/make

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


docker-start:
	@docker-compose up -d
	@echo "---Docker compose started"

# shutting down docker components
docker-stop:
	@docker-compose down
	@echo "---Docker compose stopped"

docker-clean:
	@docker volume prune --force
	@echo "---Docker env cleaned"

testsuite-clean: docker-clean
	@echo "---Test suite cleaned"

testsuite-start: docker-start
	@echo "---Test suite started and ready"

testsuite-stop: docker-stop docker-clean
	@echo "---Testsuite stopped"

testsuite-clean: docker-clean

# this command will trigger integration test
# INTEGRATION_TEST_SUITE_PATH is used for run specific test in Golang, if it's not specified
# it will run all tests under ./testing directory
#test-integration:
#	$(ENV_LOCAL_TEST) \
#  	go test -tags=integration $(INTEGRATION_TEST_PATH) -count=1

test:
	go test ./...

lint:
	@if gofmt -l . | egrep -v ^vendor/ | grep .go; then \
	  echo "^- Repo contains improperly formatted go files; run gofmt -w *.go" && exit 1; \
	  else echo "All .go files formatted correctly"; fi
	#go tool vet -v -composites=false *.go
	#go tool vet -v -composites=false **/*.go
	for pkg in $$(go list ./... |grep -v /vendor/); do golint $$pkg; done