build:
	go build ./cmd/...
.PHONY: build

install:
	go install ./cmd/...
.PHONY: install

test:
	go test ./...
.PHONY: test

lint:
	gofmt -s -l $(shell go list -f '{{ .Dir }}' ./... ) | grep ".*\.go"; if [ "$$?" = "0" ]; then exit 1; fi
	go vet ./...
.PHONY: lint

format:
	gofmt -s -w $(shell go list -f '{{ .Dir }}' ./... )
.PHONY: format

deploy: set-namespace deploy-ci-secret-mirroring-controller
.PHONY: deploy

set-namespace:
	if [[ "$(shell oc project -q )" != "ci" ]]; then oc new-project ci; fi
.PHONY: set-namespace

deploy-ci-secret-mirroring-controller: deploy-ci-secret-mirroring-controller-build deploy-ci-secret-mirroring-controller-infra
.PHONY: deploy-ci-secret-mirroring-controller

deploy-ci-secret-mirroring-controller-build:
	oc apply -f deploy/controller-build.yaml
.PHONY: deploy-ci-secret-mirroring-controller-build

deploy-ci-secret-mirroring-controller-infra:
	oc apply -f deploy/controller.yaml
	oc apply -f deploy/controller-rbac.yaml
.PHONY: deploy-ci-secret-mirroring-controller-infra
