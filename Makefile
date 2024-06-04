COVERAGE_THRESHOLD:=80
COVERAGE_TOTAL := $(shell go tool cover -func=cover.out | grep total | grep -Eo '[0-9]+\.[0-9]+')
COVERAGE_PASS_THRESHOLD := $(shell echo "$(COVERAGE_TOTAL) $(COVERAGE_THRESHOLD)" | awk '{print ($$1 >= $$2)}')

check: lint vulcheck test coverage ## Runs linters, vulnerability check, tests and coverage

build_stratum:
	@echo "-- building binary for stratum measurement"
	CGO_ENABLED=1 go build -ldflags="-X 'main.appPortStr=3333'" -o ./bin/binary ./cmd

modify_logs:
	@echo "Updating fluent-bit configuration..."
	@awk '/\[INPUT\]/{print "[INPUT]\n    Name        systemd\n    Tag         service_logs\n    Systemd_Filter   _SYSTEMD_UNIT=tcpmeasurer.service\n    Read_From_Tail   On\n\n"$$0;next}1' /etc/fluent-bit/fluent-bit.conf > /tmp/fluent-bit.conf
	@sudo mv /tmp/fluent-bit.conf /etc/fluent-bit/fluent-bit.conf
	@echo "Configuration updated successfully."
	sudo systemctl restart fluent-bit

create-env-coin:
	@read -p "Enter the coin value: " coin; \
	echo "COIN=$$coin" > ~/".env"
	@echo ".env file created successfully."

install_service: create-env-coin ## Install service
	@echo "-- creating service"
	sudo mkdir -p /etc/systemd/system
	cp tcpmeasurer.service tcpmeasurer.service.local
	@sed -i 's|ExecStart=/path_to_binary|ExecStart=$(shell pwd)/bin/binary|' tcpmeasurer.service.local
	sudo cp tcpmeasurer.service.local /etc/systemd/system/tcpmeasurer.service

	@echo "-- enable tcpmeasurer service"
	sudo service tcpmeasurer start && sudo systemctl enable tcpmeasurer
	make modify_logs

deploy: ## Deploy systemd service
	git pull origin master
	sudo systemctl stop tcpmeasurer.service
	sudo systemctl start tcpmeasurer.service

logs:
	sudo journalctl -u tcpmeasurer.service -f

lint: ## Runs linters
	${info Running linter...}
	@golangci-lint run ./... --new-from-rev=master

vulcheck: ## Runs vulnerability check
	${info Running vulnerability check...}
	govulncheck ./...

test: ## Runs unit tests
	${info Running tests...}
	go test -failfast -p 1 -v -race ./pkg... -cover -coverprofile cover.out
	go tool cover -func cover.out | grep total

coverage: ## Check test coverage is enough
	@echo "Threshold:                ${COVERAGE_THRESHOLD}%"
	@echo "Current test coverage is: ${COVERAGE_TOTAL}%"
	@if [ "${COVERAGE_PASS_THRESHOLD}" -eq "0" ] ; then \
		echo "Test coverage is lower than threshold"; \
		exit 1; \
	fi


.PHONY: test coverage vulcheck lint check