APP_NAME=veet-code-read-metrics-from-users
LAMBDA_NAME=lambda_read_metrics_from_users
ENTRYPOINT=./main
ZIP_BASENAME=$(LAMBDA_NAME)
OUTPUT_DIR=../../../../veet-code-infra/lambdas/$(LAMBDA_NAME)

.PHONY: build package clean

## Builda a Lambda em Go para Amazon Linux 2 (provided.al2)
build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bootstrap $(ENTRYPOINT)

## Empacota o binário em .zip e move para o diretório de infra
package: build
	mkdir -p $(OUTPUT_DIR)
	zip -j $(OUTPUT_DIR)/$(ZIP_BASENAME).zip bootstrap

## Limpa artefatos locais
clean:
	rm -f bootstrap
	rm -f $(OUTPUT_DIR)/$(ZIP_BASENAME).zip
