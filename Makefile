.PHONY: build
build:
		GOARM=6 GOARCH=arm GOOS=linux go build -v -o ./service
		scp service pi@192.168.88.28:/home/pi/carwashgo
		ssh -t pi@192.168.88.28 "sudo /home/pi/carwashgo/service"

.DEFAULT_GOAL := build
