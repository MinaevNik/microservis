package main

import (
	"servis/pkg/api"
	"servis/pkg/ethernet"
	"servis/pkg/rtc"
	"servis/pkg/systemd"
	//"servis/pkg/update"
)

func main() {
	systemd.Manage()
	rtc.ConfigureRTC()
	ethernet.ConfigureEthernet()
	//update.StartUpdate()
	api.StartServer()
}
