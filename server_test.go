package dhcp4server

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"github.com/d2g/dhcp4"
	"github.com/d2g/dhcp4client"
	"github.com/d2g/dhcp4server/leasepool"
	"github.com/d2g/dhcp4server/leasepool/memorypool"
	"github.com/d2g/hardwareaddr"
	"log"
	"net"
	"sync"
	"testing"
	"time"
)

/*
 * Example Server :D
 */
func ExampleServer() {
	//Configure the Server:
	myConfiguration := Configuration{}

	//You Should replace this with the IP of the Machine you are running the application on.
	myConfiguration.IP = net.IPv4(192, 168, 1, 201)

	//Replace this section with the subnet
	myConfiguration.SubnetMask = net.IPv4(255, 255, 255, 0)

	//Default Gateway
	myConfiguration.DefaultGateway = net.IPv4(192, 168, 1, 201)

	//Update these with your selected DNS Servers these belong to OpenDNS
	myConfiguration.DNSServers = []net.IP{net.IPv4(208, 67, 222, 222), net.IPv4(208, 67, 220, 220)}

	//Our Leases Are valid for 24 hours..
	myConfiguration.LeaseDuration = 24 * time.Hour

	//Create a Lease Pool We're going to use a memory pool
	//Remember the memory is cleared on restart so you will reissue the same IP Addresses.
	myMemoryLeasePool := memorypool.MemoryPool{}

	//Lets add a list of IPs to the pool these will be served to the clients so make sure they work for you.
	// So Create Array of IPs 192.168.1.1 to 192.168.1.30
	for i := 0; i < 30; i++ {
		err := myMemoryLeasePool.AddLease(leasepool.Lease{IP: dhcp4.IPAdd(net.IPv4(192, 168, 1, 1), i)})
		if err != nil {
			log.Fatalln("Error Adding IP to pool:" + err.Error())
		}
	}

	//Create The Server
	myServer := Server{}
	myServer.Configuration = &myConfiguration
	myServer.LeasePool = &myMemoryLeasePool

	//Start the Server...
	err := myServer.ListenAndServe()
	if err != nil {
		log.Fatalln("Error Starting Server:" + err.Error())
	}
}

/*
 * Dummy Test This allows us to run a server Independantly
 */
func TestDummyTest(test *testing.T){
	myServer := GetTestServerInstance()
	myServer.ListenAndServe()
}

/*
 * Test Server Used In Tests.
 */
func GetTestServerInstance() Server {
	//Configure the Server:
	myConfiguration := Configuration{}

	//You Should replace this with the IP of the Machine you are running the application on.
	myConfiguration.IP = net.IPv4(192, 168, 1, 201)

	//Replace this section with the subnet
	myConfiguration.SubnetMask = net.IPv4(255, 255, 255, 0)

	//Default Gateway
	myConfiguration.DefaultGateway = net.IPv4(192, 168, 1, 201)

	//Update these with your selected DNS Servers these belong to OpenDNS
	myConfiguration.DNSServers = []net.IP{net.IPv4(208, 67, 222, 222), net.IPv4(208, 67, 220, 220)}

	//Our Leases Are valid for 24 hours..
	myConfiguration.LeaseDuration = 24 * time.Hour

	//Create a Lease Pool We're going to use a memory pool
	//Remember the memory is cleared on restart so you will reissue the same IP Addresses.
	myMemoryLeasePool := memorypool.MemoryPool{}

	//Lets add a list of IPs to the pool these will be served to the clients so make sure they work for you.
	// So Create Array of IPs 192.168.1.1 to 192.168.1.30
	for i := 0; i < 30; i++ {
		err := myMemoryLeasePool.AddLease(leasepool.Lease{IP: dhcp4.IPAdd(net.IPv4(192, 168, 1, 1), i)})
		if err != nil {
			log.Fatalln("Error Adding IP to pool:" + err.Error())
		}
	}

	//Create The Server
	myServer := Server{}
	myServer.Configuration = &myConfiguration
	myServer.LeasePool = &myMemoryLeasePool

	return myServer
}

/*
 * Test Configuration Marshalling
 */
func TestConfigurationJSONMarshalling(test *testing.T) {
	var err error

	startConfiguration := Configuration{}
	startConfiguration.IP = net.IPv4(192, 168, 0, 1)
	startConfiguration.DefaultGateway = net.IPv4(192, 168, 0, 254)
	startConfiguration.DNSServers = []net.IP{net.IPv4(208, 67, 222, 222), net.IPv4(208, 67, 220, 220)}
	startConfiguration.SubnetMask = net.IPv4(255, 255, 255, 0)
	startConfiguration.LeaseDuration = 24 * time.Hour
	startConfiguration.IgnoreIPs = []net.IP{net.IPv4(192, 168, 0, 100)}

	startConfiguration.IgnoreHardwareAddress = make([]net.HardwareAddr, 0)
	exampleMac, err := net.ParseMAC("00:00:00:00:00:00")
	if err != nil {
		test.Error("Error Parsing Mac Address:" + err.Error())
	}
	startConfiguration.IgnoreHardwareAddress = append(startConfiguration.IgnoreHardwareAddress, exampleMac)

	test.Logf("Configuration Object:%v\n", startConfiguration)

	bytestartConfiguration, err := json.Marshal(startConfiguration)
	if err != nil {
		test.Error("Error Marshaling to JSON:" + err.Error())
	}

	test.Log("As JSON:" + string(bytestartConfiguration))

	endConfiguration := Configuration{}
	err = json.Unmarshal(bytestartConfiguration, &endConfiguration)
	if err != nil {
		test.Error("Error Unmarshaling to JSON:" + err.Error())
	}

	test.Logf("Configuration Object:%v\n", endConfiguration)
}

/*
 * Test Discovering a Lease That's not Within Our Lease Range.
 * This Happens When a devce switches network.
 * Example: Mobile Phone on Mobile internet Has IP 100.123.123.123 Switch To Home Wifi
 * The device requests 100.123.123.123 on Home Wifi which is out of range...
 */
func TestDiscoverOutOfRangeLease(test *testing.T) {
	test.SkipNow()
	myServer := GetTestServerInstance()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		err := myServer.ListenAndServe()
		if err != nil {
			log.Fatalln("Error Starting Server:" + err.Error())
		}
	}()

	time.Sleep(time.Duration(5) * time.Second)

	//Generate Hardware Address
	HardwareMACAddress, err := hardwareaddr.GenerateEUI48()
	if err != nil {
		test.Error("Error: Can't Generate Valid MACAddress" + err.Error())
	}

	//Lets Be A Client
	client := dhcp4client.Client{
		MACAddress:    HardwareMACAddress,
		IgnoreServers: []net.IP{},
		Timeout:       (time.Second * 5),
	}

	err = client.Connect()
	defer client.Close()

	if err != nil {
		test.Error("Conection Error:" + err.Error())
	}

	discoveryPacket := client.DiscoverPacket()
	discoveryPacket.SetCIAddr(net.IPv4(100, 102, 96, 123))
	discoveryPacket.PadToMinSize()

	err = client.SendPacket(discoveryPacket)
	if err != nil {
		test.Error("Error: Sending Discover Packet" + err.Error())
	}

	test.Log("--Discovery Packet--")
	test.Logf("Client IP : %v\n", discoveryPacket.CIAddr().String())
	test.Logf("Your IP   : %v\n", discoveryPacket.YIAddr().String())
	test.Logf("Server IP : %v\n", discoveryPacket.SIAddr().String())
	test.Logf("Gateway IP: %v\n", discoveryPacket.GIAddr().String())
	test.Logf("Client Mac: %v\n", discoveryPacket.CHAddr().String())

	if !bytes.Equal(discoveryPacket.CHAddr(), HardwareMACAddress) {
		test.Error("MACAddresses Don't Match??")
	}

	offerPacket, err := client.GetOffer(&discoveryPacket)
	if err != nil {
		test.Error("Error Getting Offer:" + err.Error())
	}

	test.Log("--Offer Packet--")
	test.Logf("Client IP : %v\n", offerPacket.CIAddr().String())
	test.Logf("Your IP   : %v\n", offerPacket.YIAddr().String())
	test.Logf("Server IP : %v\n", offerPacket.SIAddr().String())
	test.Logf("Gateway IP: %v\n", offerPacket.GIAddr().String())
	test.Logf("Client Mac: %v\n", offerPacket.CHAddr().String())

	requestPacket, err := client.SendRequest(&offerPacket)
	if err != nil {
		test.Error("Error Sending Request:" + err.Error())
	}

	test.Log("--Request Packet--")
	test.Logf("Client IP : %v\n", requestPacket.CIAddr().String())
	test.Logf("Your IP   : %v\n", requestPacket.YIAddr().String())
	test.Logf("Server IP : %v\n", requestPacket.SIAddr().String())
	test.Logf("Gateway IP: %v\n", requestPacket.GIAddr().String())
	test.Logf("Client Mac: %v\n", requestPacket.CHAddr().String())

	acknowledgement, err := client.GetAcknowledgement(&requestPacket)
	if err != nil {
		test.Error("Error Getting Acknowledgement:" + err.Error())
	}

	test.Log("--Acknowledgement Packet--")
	test.Logf("Client IP : %v\n", acknowledgement.CIAddr().String())
	test.Logf("Your IP   : %v\n", acknowledgement.YIAddr().String())
	test.Logf("Server IP : %v\n", acknowledgement.SIAddr().String())
	test.Logf("Gateway IP: %v\n", acknowledgement.GIAddr().String())
	test.Logf("Client Mac: %v\n", acknowledgement.CHAddr().String())

	acknowledgementOptions := acknowledgement.ParseOptions()
	if dhcp4.MessageType(acknowledgementOptions[dhcp4.OptionDHCPMessageType][0]) != dhcp4.ACK {
		test.Error("Didn't get ACK?:" + err.Error())
	}

	test.Log("Shutting Down Server")
	myServer.Shutdown()
	wg.Wait()
}

/*
 * Try Renewing A Lease From A Different Network.
 */
func TestRequestOutOfRangeLease(test *testing.T) {
	test.SkipNow()
	myServer := GetTestServerInstance()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		err := myServer.ListenAndServe()
		if err != nil {
			log.Fatalln("Error Starting Server:" + err.Error())
		}
	}()

	//Sleep some so the server starts....
	time.Sleep(time.Duration(5) * time.Second)

	//Generate Hardware Address
	HardwareMACAddress, err := hardwareaddr.GenerateEUI48()
	if err != nil {
		test.Error("Error: Can't Generate Valid MACAddress" + err.Error())
	}

	HardwareMACAddress, err = net.ParseMAC("58-94-6B-73-57-0C")
	if err != nil {
		log.Printf("MAC Error:%v\n", err)
	}

	//Lets Be A Client
	client := dhcp4client.Client{
		MACAddress:    HardwareMACAddress,
		IgnoreServers: []net.IP{},
		Timeout:       (time.Second * 5),
	}

	err = client.Connect()
	defer client.Close()

	if err != nil {
		test.Error("Conection Error:" + err.Error())
	}

	//Create a dummy offer packet
	offerPacket := client.DiscoverPacket()

	offerPacket.SetCIAddr(net.IPv4(100, 102, 96, 123))
	offerPacket.SetSIAddr(net.IPv4(192, 168, 1, 201))
	offerPacket.SetYIAddr(net.IPv4(100, 102, 96, 123))
	offerPacket.AddOption(dhcp4.OptionDHCPMessageType, []byte{byte(dhcp4.Offer)})

	requestPacket, err := client.SendRequest(&offerPacket)
	if err != nil {
		test.Error("Error Sending Request:" + err.Error())
	}

	test.Log("--Request Packet--")
	test.Logf("Client IP : %v\n", requestPacket.CIAddr().String())
	test.Logf("Your IP   : %v\n", requestPacket.YIAddr().String())
	test.Logf("Server IP : %v\n", requestPacket.SIAddr().String())
	test.Logf("Gateway IP: %v\n", requestPacket.GIAddr().String())
	test.Logf("Client Mac: %v\n", requestPacket.CHAddr().String())

	acknowledgement, err := client.GetAcknowledgement(&requestPacket)
	if err != nil {
		test.Error("Error Getting Acknowledgement:" + err.Error())
	}

	test.Log("--Acknowledgement Packet--")
	test.Logf("Client IP : %v\n", acknowledgement.CIAddr().String())
	test.Logf("Your IP   : %v\n", acknowledgement.YIAddr().String())
	test.Logf("Server IP : %v\n", acknowledgement.SIAddr().String())
	test.Logf("Gateway IP: %v\n", acknowledgement.GIAddr().String())
	test.Logf("Client Mac: %v\n", acknowledgement.CHAddr().String())

	acknowledgementOptions := acknowledgement.ParseOptions()
	if len(acknowledgementOptions[dhcp4.OptionDHCPMessageType]) <= 0 || dhcp4.MessageType(acknowledgementOptions[dhcp4.OptionDHCPMessageType][0]) != dhcp4.NAK {
		test.Errorf("Didn't get NAK got DHCP4 Message Type:%v\n", dhcp4.MessageType(acknowledgementOptions[dhcp4.OptionDHCPMessageType][0]))
	}

	test.Log("Shutting Down Server")
	myServer.Shutdown()
	wg.Wait()
}

/*
 * 
 */
func TestConsumeLeases(test *testing.T) {
	test.SkipNow()
	
	//Setup the Server
	myServer := GetTestServerInstance()

	//Setup A Client
	// Although We Won't send the packets over the network we'll use the client to create the requests.
	client := dhcp4client.Client{
		IgnoreServers: []net.IP{},
		Timeout:       (time.Second * 5),
	}

	for i := 0; i < 30; i++ {
		//Generate Hardware Address
		HardwareMACAddress, err := hardwareaddr.GenerateEUI48()
		if err != nil {
			test.Error("Error: Can't Generate Valid MACAddress" + err.Error())
		}

		client.MACAddress = HardwareMACAddress

		test.Log("MAC:" + client.MACAddress.String())

		discovery := client.DiscoverPacket()

		//Run the Discovery On the Server
		offer, err := myServer.ServeDHCP(discovery)
		_, err = myServer.ServeDHCP(discovery)
		if err != nil {
			test.Error("Discovery Error:" + err.Error())
		}

		request := client.RequestPacket(&offer)
		acknowledgement, err := myServer.ServeDHCP(request)
		if err != nil {
			test.Error("Acknowledge Error:" + err.Error())
		}

		test.Logf("Received Lease:%v\n", acknowledgement.YIAddr().String())
		if !dhcp4.IPAdd(net.IPv4(192, 168, 1, 1), i).Equal(acknowledgement.YIAddr()) {
			test.Error("Expected IP:" + dhcp4.IPAdd(net.IPv4(192, 168, 1, 1), i).String() + " Received:" + acknowledgement.YIAddr().String())
		}

		//How long the lease is for?
		acknowledgementOptions := acknowledgement.ParseOptions()
		if len(acknowledgementOptions) > 0 {
			test.Logf("Lease Options:%v\n", acknowledgementOptions)
			if acknowledgementOptions[dhcp4.OptionIPAddressLeaseTime] != nil {
				var result uint32
				buf := bytes.NewBuffer(acknowledgementOptions[dhcp4.OptionIPAddressLeaseTime])
				binary.Read(buf, binary.BigEndian, &result)
				test.Logf("Lease Time (Seconds):%d\n", result)
			}
		} else {
			test.Errorf("Lease:\"%v\" Has No Options\n", acknowledgement.YIAddr())
		}
	}
}

/*
 * Benchmark the ServeDHCP Function
 */
func BenchmarkServeDHCP(test *testing.B) {
	//Setup the Server
	myServer := GetTestServerInstance()

	//Load the additional Leases We'll need
	for i := 30; i < test.N; i++ {
		myServer.LeasePool.AddLease(leasepool.Lease{IP: dhcp4.IPAdd(net.IPv4(192, 168, 1, 1), i)})
	}

	//Setup A Client
	// Although We Won't send the packets over the network we'll use the client to create the requests.
	client := dhcp4client.Client{
		IgnoreServers: []net.IP{},
		Timeout:       (time.Second * 5),
	}

	test.ResetTimer()

	for i := 0; i < test.N; i++ {
		test.StopTimer()
		//Generate Hardware Address
		HardwareMACAddress, err := hardwareaddr.GenerateEUI48()
		if err != nil {
			test.Error("Error: Can't Generate Valid MACAddress" + err.Error())
		}

		client.MACAddress = HardwareMACAddress
		discovery := client.DiscoverPacket()

		//Run the Discovery On the Server
		test.StartTimer()
		offer, err := myServer.ServeDHCP(discovery)
		if err != nil {
			test.Error("Discovery Error:" + err.Error())
		}

		if len(offer) == 0 {
			test.Error("No Valid Offer")
		} else {
			request := client.RequestPacket(&offer)
			_, err := myServer.ServeDHCP(request)
			if err != nil {
				test.Error("Acknowledge Error:" + err.Error())
			}

		}
	}
}
