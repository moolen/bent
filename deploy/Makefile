# Validate for all environments to ensure .tfvars files are complete.
.PHONY: validate
validate:
	terraform init -input=false -backend=false
	terraform validate
	terraform fmt -check=true -diff=true

init:
	terraform init -input=false -reconfigure

.PHONY: plan
plan: init
	terraform plan -input=false -var keypair=$(keypair)

.PHONY: apply
apply: init
	terraform apply -input=false -auto-approve -var keypair=$(keypair)

.PHONY: destroy
destroy: init
	terraform destroy -input=false -auto-approve -var keypair=$(keypair)
