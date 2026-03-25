BINARY     := expose
MODULE     := github.com/kevin-cantwell/expose
REMOTE_BIN := /usr/local/bin/expose
SERVICE    := expose

# Load personal deploy settings (gitignored). Copy local.mk.example to local.mk.
-include local.mk

.PHONY: build build-linux install deploy restart logs clean

build:
	go build -o $(BINARY) .

build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY)-linux .

install: build
	cp $(BINARY) ~/.local/bin/$(BINARY)

deploy: build-linux
ifndef DROPLET_IP
	$(error DROPLET_IP not set — copy local.mk.example to local.mk and configure it)
endif
	scp -i $(SSH_KEY) $(BINARY)-linux $(DROPLET_USER)@$(DROPLET_IP):/tmp/$(BINARY)-new
	ssh -i $(SSH_KEY) $(DROPLET_USER)@$(DROPLET_IP) "mv /tmp/$(BINARY)-new $(REMOTE_BIN) && systemctl restart $(SERVICE)"
	rm $(BINARY)-linux
	@echo "Deployed and restarted $(SERVICE) on $(DROPLET_IP)"

restart:
ifndef DROPLET_IP
	$(error DROPLET_IP not set — copy local.mk.example to local.mk and configure it)
endif
	ssh -i $(SSH_KEY) $(DROPLET_USER)@$(DROPLET_IP) "systemctl restart $(SERVICE)"

logs:
ifndef DROPLET_IP
	$(error DROPLET_IP not set — copy local.mk.example to local.mk and configure it)
endif
	ssh -i $(SSH_KEY) $(DROPLET_USER)@$(DROPLET_IP) "journalctl -u $(SERVICE) -f"

clean:
	rm -f $(BINARY) $(BINARY)-linux
