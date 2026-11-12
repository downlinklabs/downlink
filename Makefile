.PHONY: help \
  github-docker-login \
  operator-build operator-docker-build operator-docker-push operator-manifests operator-test \
  runner-docker-build runner-docker-push \
  chart-deps chart-lint chart-package \
  reporter-build reporter-test

GHCR_REGISTRY ?= ghcr.io/downlinklabs
RUNNER_IMAGE  ?= $(GHCR_REGISTRY)/playwright-runner
RUNNER_TAG    ?= $(shell git rev-parse --short HEAD)

help:
	@echo ""
	@echo "Auth:"
	@echo "  github-docker-login    Authenticate Docker to ghcr.io via gh CLI"
	@echo ""
	@echo "Operator:"
	@echo "  operator-build         Build the operator binary"
	@echo "  operator-docker-build  Build the operator Docker image"
	@echo "  operator-docker-push   Push the operator Docker image"
	@echo "  operator-manifests     Regenerate CRDs and RBAC manifests"
	@echo "  operator-test          Run operator unit tests"
	@echo ""
	@echo "Runner:"
	@echo "  runner-docker-build    Build the playwright runner image"
	@echo "  runner-docker-push     Build and push the playwright runner image"
	@echo ""
	@echo "Helm chart:"
	@echo "  chart-deps             Update chart dependencies"
	@echo "  chart-lint             Lint the Helm chart"
	@echo "  chart-package          Package the Helm chart"
	@echo ""
	@echo "Reporter:"
	@echo "  reporter-build         Build @downlink/synthetics"
	@echo "  reporter-test          Run @downlink/synthetics tests"

##@ Operator

operator-build:
	$(MAKE) -C operator build

operator-docker-build:
	$(MAKE) -C operator docker-build

operator-docker-push: github-docker-login
	$(MAKE) -C operator docker-push

operator-manifests:
	$(MAKE) -C operator manifests

operator-test:
	$(MAKE) -C operator test

##@ Runner

runner-docker-build:
	docker build -f playwright-runner/Dockerfile -t $(RUNNER_IMAGE):$(RUNNER_TAG) -t $(RUNNER_IMAGE):latest .

github-docker-login:
	gh auth token | docker login ghcr.io --username $$(gh api user --jq .login) --password-stdin

runner-docker-push: runner-docker-build github-docker-login
	docker push $(RUNNER_IMAGE):$(RUNNER_TAG)
	docker push $(RUNNER_IMAGE):latest

##@ Helm Chart

chart-deps:
	helm dependency update charts/downlink

chart-lint: chart-deps
	helm lint charts/downlink

chart-package: chart-deps
	helm package charts/downlink

##@ Reporter

reporter-build:
	cd packages/synthetics && npm run build

reporter-test:
	cd packages/synthetics && npm test
