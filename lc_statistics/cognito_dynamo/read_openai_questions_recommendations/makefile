APP_NAME=veet-code-openai-questions-recomendations
LAMBDA_NAME=lambda_read_openai_questions_recomendations
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
