APP_NAME=veet-code-frontend-event-reciever
LAMBDA_NAME=frontend_event_reciever
ENTRYPOINT=./main
ZIP_BASENAME=$(LAMBDA_NAME)
OUTPUT_DIR=../../../../veet-code-infra/lambdas/$(LAMBDA_NAME)

.PHONY: build package clean

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bootstrap $(ENTRYPOINT)

package: build
	mkdir -p $(OUTPUT_DIR)
	zip -j $(OUTPUT_DIR)/$(ZIP_BASENAME).zip bootstrap

clean:
	rm -f bootstrap
	rm -f $(OUTPUT_DIR)/$(ZIP_BASENAME).zip
