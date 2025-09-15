export MAIN_BRANCH ?= main

ifndef ignore-not-found
  ignore-not-found = false
endif

.DEFAULT_GOAL := help

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

KUSTOMIZE ?= $(LOCALBIN)/kustomize
KUSTOMIZE_VERSION ?= v4.5.5
KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"

GIT_BRANCH := $(shell git symbolic-ref --short HEAD)
WORKTREE_CLEAN := $(shell git status --porcelain 1>/dev/null 2>&1; echo $$?)
SCRIPTS_DIR := $(CURDIR)/scripts

versionFile = $(CURDIR)/.VERSION
curVersion := $(shell cat $(versionFile) | sed 's/^v//')

IMG_TAG ?= 1password/kubernetes-secrets-injector:latest

.PHONY: test
test:	## Run test suite
	go test ./...

KIND ?= kind
KIND_CLUSTER ?= kubernetes-secrets-injector-test-e2e

.PHONY: setup-test-e2e
setup-test-e2e: ## Set up a Kind cluster for e2e tests if it does not exist
	@command -v $(KIND) >/dev/null 2>&1 || { \
		echo "Kind is not installed. Please install Kind manually."; \
		exit 1; \
	}
	@case "$$($(KIND) get clusters)" in \
		*"$(KIND_CLUSTER)"*) \
			echo "Kind cluster '$(KIND_CLUSTER)' already exists. Skipping creation." ;; \
		*) \
			echo "Creating Kind cluster '$(KIND_CLUSTER)'..."; \
			$(KIND) create cluster --name $(KIND_CLUSTER) ;; \
	esac

.PHONY: test/coverage
test/coverage:	## Run test suite with coverage report
	go test -v ./... -cover

.PHONY: test-e2e
test-e2e: setup-test-e2e ## Run the e2e tests. Expected an isolated environment using Kind.
	KIND_CLUSTER=$(KIND_CLUSTER) go test ./test/e2e/ -v -ginkgo.v
	$(MAKE) cleanup-test-e2e

.PHONY: cleanup-test-e2e
cleanup-test-e2e: ## Tear down the Kind cluster used for e2e tests
	@$(KIND) delete cluster --name $(KIND_CLUSTER)

.PHONY: docker-build
docker-build:	## Build secrets-injector Docker image
	@docker build -f Dockerfile --build-arg secret_injector_version=$(curVersion) -t $(IMG_TAG) .
	@echo "Successfully built and tagged image."
	@echo "Tag: $(IMG_TAG)"

.PHONY: build/secrets-injector/binary
build/secrets-injector/binary: clean	## Build secrets-injector binary
	@mkdir -p dist
	@go build -a -o manager ./cmd
	@mv manager ./dist

.PHONY: clean
clean:
	rm -rf ./dist

.PHONY: help
help:	## Prints this help message
	@grep -E '^[\/a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || { curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: set-namespace
set-namespace: kustomize
	cd deploy && $(KUSTOMIZE) edit set namespace $(shell kubectl config view --minify -o jsonpath={..namespace})

.PHONY: deploy
deploy: set-namespace
	$(KUSTOMIZE) build deploy | kubectl apply -f -

.PHONY: undeploy
undeploy:
	$(KUSTOMIZE) build deploy --reorder none | kubectl delete --ignore-not-found=$(ignore-not-found) -f -


## Release functions =====================

.PHONY: release/prepare
release/prepare: .check_git_clean	## Updates changelog and creates release branch (call with 'release/prepare version=<new_version_number>')

	@test $(version) || (echo "[ERROR] version argument not set."; exit 1)
	@git fetch --quiet origin $(MAIN_BRANCH)

	@echo $(version) | tr -d '\n' | tee $(versionFile) &>/dev/null

	@NEW_VERSION=$(version) $(SCRIPTS_DIR)/prepare-release.sh

.PHONY: release/tag
release/tag: .check_git_clean	## Creates git tag
	@git pull --ff-only
	@echo "Applying tag 'v$(curVersion)' to HEAD..."
	@git tag --sign "v$(curVersion)" -m "Release v$(curVersion)"
	@echo "[OK] Success!"
	@echo "Remember to call 'git push --tags' to persist the tag."

## Helper functions =====================

.PHONY: .check_git_clean
.check_git_clean:
ifneq ($(GIT_BRANCH), $(MAIN_BRANCH))
	@echo "[ERROR] Please checkout default branch '$(MAIN_BRANCH)' and re-run this command."; exit 1;
endif
ifneq ($(WORKTREE_CLEAN), 0)
	@echo "[ERROR] Uncommitted changes found in worktree. Address them and try again."; exit 1;
endif
