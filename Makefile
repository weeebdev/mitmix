HUB_URL ?= ws://localhost:8090
TOKEN ?= test-agent-token
VENV = agent/.venv
PIDFILE = /tmp/mitmix-agent.pid

.PHONY: proxy-on proxy-off proxy-status agent-stop agent-logs build-hub build-dashboard cert-install

proxy-on: check-agent
	@if [ -f $(PIDFILE) ] && kill -0 $$(cat $(PIDFILE)) 2>/dev/null; then \
		echo "Agent already running (PID $$(cat $(PIDFILE)))"; \
	else \
		echo "Starting mitmix agent..."; \
		nohup $(VENV)/bin/python agent/mitm_agent.py --hub $(HUB_URL) --token $(TOKEN) > /tmp/mitmix-agent.log 2>&1 & \
		echo $$! > $(PIDFILE); \
		sleep 2; \
		if ! kill -0 $$(cat $(PIDFILE)) 2>/dev/null; then \
			echo "Agent failed to start. Check /tmp/mitmix-agent.log"; \
			rm -f $(PIDFILE); \
			exit 1; \
		fi; \
		echo "Agent running (PID $$(cat $(PIDFILE)))"; \
	fi
	@echo "Setting system proxy..."
	sudo networksetup -setwebproxy Wi-Fi 127.0.0.1 8082
	sudo networksetup -setsecurewebproxy Wi-Fi 127.0.0.1 8082
	@echo ""
	@echo "Proxy: ON  -> 127.0.0.1:8082"
	@echo "Dashboard: http://localhost:8090/dashboard"
	@echo "Run 'make proxy-off' to stop."

proxy-off:
	@echo "Clearing system proxy..."
	sudo networksetup -setwebproxystate Wi-Fi off
	sudo networksetup -setsecurewebproxystate Wi-Fi off
	@if [ -f $(PIDFILE) ] && kill -0 $$(cat $(PIDFILE)) 2>/dev/null; then \
		kill $$(cat $(PIDFILE)) 2>/dev/null || true; \
		rm -f $(PIDFILE); \
		echo "Agent stopped."; \
	else \
		rm -f $(PIDFILE); \
		echo "No agent running."; \
	fi
	@echo "Proxy: OFF"

proxy-status:
	@echo "=== Agent ==="
	@if [ -f $(PIDFILE) ] && kill -0 $$(cat $(PIDFILE)) 2>/dev/null; then \
		echo "Status: running (PID $$(cat $(PIDFILE)))"; \
	else \
		rm -f $(PIDFILE); \
		echo "Status: stopped"; \
	fi
	@echo ""
	@echo "=== System Proxy ==="
	@networksetup -getwebproxy Wi-Fi 2>&1 | grep -E "Enabled|Server|Port" | sed 's/^/  HTTP: /'
	@networksetup -getsecurewebproxy Wi-Fi 2>&1 | grep -E "Enabled|Server|Port" | sed 's/^/  HTTPS: /'

agent-stop:
	@if [ -f $(PIDFILE) ] && kill -0 $$(cat $(PIDFILE)) 2>/dev/null; then \
		kill $$(cat $(PIDFILE)) 2>/dev/null || true; \
		rm -f $(PIDFILE); \
		echo "Agent stopped."; \
	else \
		rm -f $(PIDFILE); \
		echo "No agent running."; \
	fi

agent-logs:
	@tail -f /tmp/mitmix-agent.log

rebuild: build-dashboard
	docker compose up --build -d

build-hub:
	cd internal/hub/site && npm run build && cd ../../..
	docker compose up --build -d hub

build-dashboard:
	cd internal/hub/site && npm run build

cert-install-firefox:
	@echo "Installing mitmix CA cert into Firefox/Zen..."
	@if ! which certutil >/dev/null 2>&1; then \
		echo "certutil not found. Run: brew install nss"; \
		exit 1; \
	fi
	@PROFILE=$$(ls -d ~/.mozilla/firefox/*.default* ~/.mozilla/firefox/*.default-esr "$$HOME/Library/Application Support/zen/Profiles/"* 2>/dev/null | head -1); \
	if [ -z "$$PROFILE" ]; then \
		echo "No Firefox/Zen profile found"; \
		exit 1; \
	fi; \
	certutil -A -n "mitmix CA" -t "TCu,Cu,Tu" -i ~/.mitmproxy/mitmproxy-ca-cert.pem -d "sql:$$PROFILE" && \
	echo "Done. Restart Firefox/Zen." || echo "Failed."

cert-install:
	@echo "Installing mitmix CA cert..."
	@if [ -f ~/.mitmproxy/mitmproxy-ca-cert.pem ]; then \
		echo "Using local cert at ~/.mitmproxy/mitmproxy-ca-cert.pem"; \
		sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ~/.mitmproxy/mitmproxy-ca-cert.pem; \
		echo "Done."; \
	else \
		curl -sf http://localhost:8090/api/mitm/ca-cert -o /tmp/mitmix-ca.pem && \
		sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain /tmp/mitmix-ca.pem && \
		echo "Done." || \
		echo "No cert found. Start hub and agent first, or run 'mitmix-agent cert install'"; \
	fi

check-agent:
	@if [ ! -f $(VENV)/bin/python ]; then \
		echo "No venv found. Run: cd agent && python3 -m venv .venv && .venv/bin/pip install -e ."; \
		exit 1; \
	fi
