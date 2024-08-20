package main

import (
	"servis/pkg/api"
	"servis/pkg/ethernet"
	"servis/pkg/rtc"
	"servis/pkg/device"
	//"servis/pkg/update"
)

func main() {
	go device.Start()
	rtc.ConfigureRTC()
	ethernet.ConfigureEthernet()
	
	api.StartServer()
}
