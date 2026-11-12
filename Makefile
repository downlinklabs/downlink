.PHONY: help \
  operator-build operator-docker-build operator-docker-push operator-manifests operator-test \
  runner-docker-build runner-docker-push \
  chart-deps chart-lint chart-package \
  reporter-build reporter-test

ECR_REGISTRY ?= 393657359434.dkr.ecr.us-east-2.amazonaws.com
RUNNER_IMAGE  ?= $(ECR_REGISTRY)/flynn/playwright-runner
RUNNER_TAG    ?= $(shell git rev-parse --short HEAD)

help:
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

operator-docker-push:
	$(MAKE) -C operator docker-push

operator-manifests:
	$(MAKE) -C operator manifests

operator-test:
	$(MAKE) -C operator test

##@ Runner

runner-docker-build:
	docker build -f playwright-runner/Dockerfile -t $(RUNNER_IMAGE):$(RUNNER_TAG) -t $(RUNNER_IMAGE):latest .

runner-docker-push: runner-docker-build
	aws ecr get-login-password --region us-east-2 | docker login --username AWS --password-stdin $(ECR_REGISTRY)
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
