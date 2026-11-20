.PHONY: help \
  release release-runner \
  github-docker-login \
  operator-build operator-docker-build operator-docker-push operator-manifests operator-test \
  runner-docker-build runner-docker-push \
  chart-deps chart-lint chart-package \
  reporter-build reporter-test

GHCR_REGISTRY ?= ghcr.io/downlinklabs
RUNNER_IMAGE  ?= $(GHCR_REGISTRY)/playwright-runner
RUNNER_TAG    ?= $(shell git rev-parse --short HEAD)

BUMP ?= minor

help:
	@echo ""
	@echo "Release:"
	@echo "  release                Bump chart/operator version, tag, and push (BUMP=major|minor|patch, default: minor)"
	@echo "  release-runner         Bump runner version, tag, and push (BUMP=major|minor|patch, default: minor)"
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

##@ Release

release:
	@current=$$(grep '^version:' charts/downlink/Chart.yaml | awk '{print $$2}'); \
	major=$$(echo $$current | cut -d. -f1); \
	minor=$$(echo $$current | cut -d. -f2); \
	patch=$$(echo $$current | cut -d. -f3); \
	case "$(BUMP)" in \
	  major) major=$$((major+1)); minor=0; patch=0 ;; \
	  minor) minor=$$((minor+1)); patch=0 ;; \
	  patch) patch=$$((patch+1)) ;; \
	  *) echo "Usage: make release BUMP=major|minor|patch"; exit 1 ;; \
	esac; \
	next="$$major.$$minor.$$patch"; \
	perl -pi -e "s/^version: .*/version: $$next/" charts/downlink/Chart.yaml; \
	git add charts/downlink/Chart.yaml; \
	git commit -m "Release v$$next"; \
	git tag "v$$next"; \
	git push && git push --tags; \
	echo "Released v$$next"

release-runner:
	@current=$$(node -p "require('./playwright-runner/package.json').version"); \
	major=$$(echo $$current | cut -d. -f1); \
	minor=$$(echo $$current | cut -d. -f2); \
	patch=$$(echo $$current | cut -d. -f3); \
	case "$(BUMP)" in \
	  major) major=$$((major+1)); minor=0; patch=0 ;; \
	  minor) minor=$$((minor+1)); patch=0 ;; \
	  patch) patch=$$((patch+1)) ;; \
	  *) echo "Usage: make release-runner BUMP=major|minor|patch"; exit 1 ;; \
	esac; \
	next="$$major.$$minor.$$patch"; \
	npm version $$next --prefix playwright-runner --no-git-tag-version; \
	git add playwright-runner/package.json; \
	git commit -m "Release runner v$$next"; \
	git tag "runner-v$$next"; \
	git push && git push --tags; \
	echo "Released runner v$$next"

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
