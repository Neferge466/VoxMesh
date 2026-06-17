module github.com/voxmesh/command-handler

go 1.26.4

require (
	github.com/eclipse/paho.mqtt.golang v1.5.1
	github.com/voxmesh/pkg v0.0.0
)

require (
	github.com/gorilla/websocket v1.5.3 // indirect
	golang.org/x/net v0.44.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
)

replace github.com/voxmesh/pkg => ../pkg
