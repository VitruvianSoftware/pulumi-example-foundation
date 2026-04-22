# Copyright 2026 Vitruvian Software
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

.PHONY: lint build test validate clean

STAGES := 0-bootstrap 1-org 2-environments 3-networks-hub-and-spoke 3-networks-svpc 4-projects 5-app-infra

# Lint all stages
lint:
	@for stage in $(STAGES); do \
		echo "=== Linting $$stage ===" ; \
		cd $$stage && go vet ./... && cd .. ; \
	done

# Build all stages
build:
	@for stage in $(STAGES); do \
		echo "=== Building $$stage ===" ; \
		cd $$stage && go build ./... && cd .. ; \
	done

# Test all stages
test:
	@for stage in $(STAGES); do \
		echo "=== Testing $$stage ===" ; \
		cd $$stage && go test ./... -race -count=1 && cd .. ; \
	done

# Pre-flight environment validation
validate:
	@echo "Usage: ./scripts/validate-requirements.sh -o <ORG_ID> -b <BILLING_ACCOUNT> -u <USER_EMAIL>"
	@./scripts/validate-requirements.sh $(ARGS)

# Tidy all modules
tidy:
	@for stage in $(STAGES); do \
		echo "=== Tidying $$stage ===" ; \
		cd $$stage && go mod tidy && cd .. ; \
	done

# Clean build artifacts
clean:
	@find . -name 'foundation-*' -type f -delete
	@find . -name '*.test' -type f -delete
	@echo "Cleaned build artifacts."
