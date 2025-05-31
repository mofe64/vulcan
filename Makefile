# -----------------------------------------------
# VARIABLES (override on CLI or env)
# -----------------------------------------------
REV        ?= $(shell git rev-parse --short HEAD)
IMAGE_REG  ?= ghcr.io/your-org/opa-bundles     # OCI registry repo
BUNDLE_TAR := bundle-$(REV).tar.gz
BUNDLE_REF := $(IMAGE_REG):$(REV)
SIGN_KEY   ?= keys/policy_private.pem

# -----------------------------------------------
# TARGETS
# -----------------------------------------------
.PHONY: test-policy build-policy push-policy-bundle

## opa fmt, vet, unit tests -----------------------------------------------
test-policy:
	@echo "🔍  Formatting, vetting and testing Rego..."
	opa fmt -w policy/
	opa vet policy/
	opa test policy/

## Build signed bundle.tar.gz from policy/ ----------------------------------
build-policy: test-policy
	@echo "📦  Building signed OPA bundle $(BUNDLE_TAR)"
	opa build -b policy -o $(BUNDLE_TAR) \
	          --revision=$(REV) \
	          --signing-key=$(SIGN_KEY)

## Push bundle to OCI registry via ORAS ------------------------------------
push-policy-bundle: build-policy
	@echo "🚀  Pushing $(BUNDLE_REF)"
	oras push $(BUNDLE_REF) \
		$(BUNDLE_TAR):application/vnd.opa.bundle.layer.v1+gzip

# Convenience: run make bundle to build+push
bundle: push-policy-bundle
	@echo "✅  Bundle available at $(BUNDLE_REF)"
