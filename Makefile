notifications: $(shell find . -name "*.go")
	go build -ldflags="-s -w" -o ./notifications

deploy: notifications
	ssh root@turgot 'systemctl stop notifications'
	scp notifications turgot:notifications/notifications
	ssh root@turgot 'systemctl start notifications'
