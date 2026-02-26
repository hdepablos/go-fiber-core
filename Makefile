
.PHONY: terraform-deploy-eks
terraform-deploy-eks: ## 🚀☸️ Despliega la infraestructura en modo EKS (usando Helm + Terraform). Uso: make terraform-deploy-eks
	@echo "$(INFO)🚀 Desplegando infraestructura en modo EKS (Híbrido)...$(RESET)"
	@cd $(TF_DIR) && \
		tflocal init && \
		tflocal apply -var-file=$(TF_VARS) -var="deploy_mode=eks" -auto-approve
	@echo "$(SUCCESS)✅ Despliegue en modo EKS completado.$(RESET)"

.PHONY: terraform-deploy-lambda
terraform-deploy-lambda: ## 🚀⚡ Despliega la infraestructura en modo Lambda (Clásico). Uso: make terraform-deploy-lambda
	@echo "$(INFO)🚀 Desplegando infraestructura en modo Lambda...$(RESET)"
	@cd $(TF_DIR) && \
		tflocal init && \
		tflocal apply -var-file=$(TF_VARS) -var="deploy_mode=lambda" -auto-approve
	@echo "$(SUCCESS)✅ Despliegue en modo Lambda completado.$(RESET)"
