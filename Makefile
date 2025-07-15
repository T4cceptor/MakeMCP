install:
	go install .

restart-mcp:
	pkill -f "localhost-mcp" || true
	sleep 1
	localhost-mcp &

config-test:
	makemcp openapi -s 'http://localhost:8081/openapi.json' -b "http://localhost:8081" --config-only true