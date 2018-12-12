NS=credentials-test

VAULT_PORT=$(shell kubectl --namespace $(NS) get service/vault -o='jsonpath={.spec.ports[0].nodePort}')

VAULT_TOKEN=vault_root_token

VAULT_ADDR=http://localhost:$(VAULT_PORT)

CURL=curl --header "X-Vault-Token: $(VAULT_TOKEN)" -i
VAULT=$(VAULT_ADDR)/v1

KUBECTL=kubectl --namespace $(NS)

TERRAFORM=VAULT_ADDR=$(VAULT_ADDR) VAULT_TOKEN=$(VAULT_TOKEN) terraform

clean:
	rm -f main terraform.tfvars

########################################
# DOCKER IMAGE

main: main.go
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

dockerimage: main Dockerfile
	docker build -t appkit-credentials-example-app .

########################################
# VAULT ON K8S

teardown:
	$(TERRAFORM) destroy -auto-approve || echo "did my best..."
	rm -f terraform.tfstate*
	kubectl delete namespace $(NS)

bootstrap:
	kubectl create namespace $(NS) || echo "ignoring existing namespace"
	$(KUBECTL) delete configmap vault-config || echo "skipping deletion"
	$(KUBECTL) create configmap vault-config --from-literal=VAULT_DEV_ROOT_TOKEN_ID=$(VAULT_TOKEN)
	$(KUBECTL) apply -f vault.yaml
	sleep 8
	$(KUBECTL) get service vault
	$(TERRAFORM) init

vaultlogs:
	$(KUBECTL) logs deployment/vault

shell:
	$(KUBECTL) run my-shell --rm -i --tty --image ububntu -- bash

########################################
# TERRAFORM

vaultconfig: terraform.tfstate
	$(CURL) $(VAULT)/$(shell terraform output vault-aws-path)

terraform.tfvars: Makefile
	echo 'kubernetes_namespace = "$(NS)"' > $@
	$(eval K8S_DEFAULT_SECRET=$(shell $(KUBECTL) get serviceaccounts default -o jsonpath='{.secrets[0].name}'))
	echo "kubernetes_ca_crt = <<CERT" >> $@
	$(KUBECTL) get secret $(K8S_DEFAULT_SECRET) -o="jsonpath={.data['ca\.crt']}" | base64 -D >> $@
	echo "CERT" >> $@

terraform.tfstate: example.tf terraform.tfvars
	$(TERRAFORM) apply -auto-approve
	sleep 8

########################################
# APP ON K8S

logs:
	$(KUBECTL) logs job/app

logs-failing:
	$(KUBECTL) logs job/app-failing

run: dockerimage
	$(KUBECTL) delete job app || echo "skipping deletion"
	$(KUBECTL) delete job app-failing || echo "skipping deletion"
	$(KUBECTL) delete configmap app || echo "skipping deletion"
	$(KUBECTL) create configmap app --from-literal=VAULT_AUTHN_ROLE=$(shell terraform output vault-authn-role) --from-literal=VAULT_AWS_PATH=$(shell terraform output vault-aws-path)
	$(KUBECTL) apply -f app.yaml
	sleep 5

########################################
# ACCEPTANCE TEST :E

test-up: clean
	make bootstrap
	make vaultconfig
	make run
	make logs
	@echo "================= LOOK FOR assumed-role ARN ^^^ ================"
	make logs-failing
	@echo "================= LOOK FOR FAILURE TO ASSUME ROLE ^^^ ================"

test: test-up teardown
