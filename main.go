package main

import (
	"servis/pkg/api"
	"servis/pkg/ethernet"
	"servis/pkg/rtc"
)

func main() {
	rtc.ConfigureRTC()
	ethernet.ConfigureEthernet()
	api.StartServer()
}
