TEST_TIMEOUT?=5m
GO_LICENSER_EXCLUDE=stacktrace/testdata
GO_LANGUAGE_VERSION=1.19

.PHONY: check
check: precheck check-modules test

.PHONY: precheck
precheck: check-goimports check-vanity-import check-vet check-dockerfile-testing check-licenses model/marshal_fastjson.go scripts/Dockerfile-testing

.PHONY: check-goimports
check-goimports:
	sh scripts/check_goimports.sh

.PHONY: check-dockerfile-testing
check-dockerfile-testing:
	go run ./scripts/gendockerfile.go -d

.PHONY: check-licenses
check-licenses:
	go run -modfile=tools/go.mod github.com/elastic/go-licenser -d $(patsubst %,-exclude %,$(GO_LICENSER_EXCLUDE)) .

.PHONY: check-modules
check-modules: update-modules
	git diff --exit-code

.PHONY: check-vanity-import
check-vanity-import:
	sh scripts/check_vanity.sh

.PHONY: check-vet
check-vet:
	@for dir in $(shell scripts/moduledirs.sh); do (cd $$dir && go vet ./...) || exit $$?; done

.PHONY: docker-test
docker-test:
	scripts/docker-compose-testing run -T --rm go-agent-tests make test

.PHONY: test
test:
	@for dir in $(shell scripts/moduledirs.sh); do (cd $$dir && go test -race -v -timeout=$(TEST_TIMEOUT) ./...) || exit $$?; done

.PHONY: coverage
coverage:
	@bash scripts/test_coverage.sh

.PHONY: fmt
fmt:
	@GOIMPORTSFLAGS=-w sh scripts/goimports.sh

.PHONY: clean
clean:
	rm -fr docs/html

.PHONY: update-modules
update-modules:
	cd scripts/genmod && go run main.go -go=$(GO_LANGUAGE_VERSION) ../..

.PHONY: docs
docs:
ifdef ELASTIC_DOCS
	$(ELASTIC_DOCS)/build_docs --direct_html --chunk=1 $(BUILD_DOCS_ARGS) --doc docs/index.asciidoc --out docs/html
else
	@echo "\nELASTIC_DOCS is not defined.\n"
	@exit 1
endif

.PHONY: update-licenses
update-licenses:
	go run -modfile=tools/go.mod github.com/elastic/go-licenser $(patsubst %, -exclude %, $(GO_LICENSER_EXCLUDE)) .

model/marshal_fastjson.go: model/model.go
	go generate ./model

module/apmgrpc/internal/testservice/testservice.pb.go:
	./scripts/install-protobuf.sh
	./scripts/generate-testservice.sh

scripts/Dockerfile-testing: $(wildcard module/*)
	go generate ./scripts
