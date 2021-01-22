notifications: $(shell find . -name "*.go")
	go build -ldflags="-s -w" -o ./notifications

deploy: notifications
	ssh root@nusakan-58 'systemctl stop notifications'
	scp notifications nusakan-58:notifications/notifications
	ssh root@nusakan-58 'systemctl start notifications'
