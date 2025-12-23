build-GoFiberCoreEvery1minCron:
	mkdir -p $(ARTIFACTS_DIR)
	cp bootstrap $(ARTIFACTS_DIR)/bootstrap
	chmod +x $(ARTIFACTS_DIR)/bootstrap
	date -u +"%Y-%m-%dT%H:%M:%SZ" > $(ARTIFACTS_DIR)/build.txt
	cd $(ARTIFACTS_DIR) && zip -r function.zip bootstrap build.txt

.PHONY: build-GoFiberCoreEvery1minCron
