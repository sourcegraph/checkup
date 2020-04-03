.PHONY: all generate_cert

all: generate_cert

GOPATH ?= /root/go
GENERATE_TLS_CERT = $(GOPATH)/bin/generate-tls-cert

$(GENERATE_TLS_CERT):
	go get -u github.com/Shyp/generate-tls-cert

leaf.pem: | $(GENERATE_TLS_CERT)
	rm *.crt *.key *.pem -f
	$(GENERATE_TLS_CERT) --host=localhost,127.0.0.1 -duration 876000h

# Generate TLS certificates for local development.
generate_cert: leaf.pem | $(GENERATE_TLS_CERT)
