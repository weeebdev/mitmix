HUB_URL ?= ws://localhost:8090
TOKEN ?= test-agent-token
VENV = agent/.venv
PIDFILE = /tmp/mitmix-agent.pid

.PHONY: proxy-on proxy-off proxy-status agent-stop agent-logs build-hub build-dashboard

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

build-hub:
	cd internal/hub/site && npm run build && cd ../../..
	docker compose up --build -d hub

build-dashboard:
	cd internal/hub/site && npm run build

check-agent:
	@if [ ! -f $(VENV)/bin/python ]; then \
		echo "No venv found. Run: cd agent && python3 -m venv .venv && .venv/bin/pip install -e ."; \
		exit 1; \
	fi
