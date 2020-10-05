PLUGIN := protect

cf:
	go build .
	cf uninstall-plugin $(PLUGIN) || true
	yes | cf install-plugin cf-$(PLUGIN)

.PHONY: cf
