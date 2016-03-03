package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	amt "github.com/VictorLowther/intelamt"
)

var endpoint, username, password string
var debug, useDHCP bool
var op string
var address, netmask, dns1, dns2, gateway, nic string
var client *amt.Client

func init() {
	flag.StringVar(&endpoint, "e", "", `The AMT management endpoint to communicate with.
        Must be configured to use digest auth.`)
	flag.StringVar(&username, "u", "admin", "The username for the AMT endpoint")
	flag.StringVar(&password, "p", "", "The password for the AMT endpoint")
	flag.BoolVar(&debug, "debug", false, "Debug will dump all WSMAN communication with the AMT endpoint")
	flag.StringVar(&op, "op", "", `The action to perform.  Can be one of:
          powerOn
          powerOff
          powerCycle
          powerState
          netState
          netConfig`)
	flag.StringVar(&nic, "nic", "wired", "Interface to manage")
	flag.BoolVar(&useDHCP, "dhcp", false, `Whether to use DHCP to configure the interface.
        If true, no other network flags will be honored`)
	flag.StringVar(&address, "ip", "", "The IPv4 to assign to the interface.")
	flag.StringVar(&netmask, "mask", "", "The netmask to use with the interface")
	flag.StringVar(&gateway, "gw", "", "The gateway address to use.  Optional")
	flag.StringVar(&dns1, "dns1", "", "The first DNS server to use.  Optional")
	flag.StringVar(&dns2, "dns2", "", "The second DNS server to use.  Optional")
}

func main() {
	flag.Parse()
	if endpoint == "" {
		log.Fatalf("-e is required")
	}
	if password == "" {
		log.Fatalf("-p is required")
	}
	client = amt.NewClient(endpoint, username, password)
	client.Debug = debug
	if err := client.Identify(); err != nil {
		log.Fatalf("Failed to ID AMT endpoint: %v", err)
	}
	switch op {
	case "powerOn":
		client.SetChassisPower("on")
	case "powerOff":
		client.SetChassisPower("off")
	case "powerCycle":
		client.SetChassisPower("off")
		client.SetChassisPower("on")
	case "powerState":
		state, err := client.GetChassisPower()
		if err != nil {
			log.Fatalf("Error getting chassis power state: %v", err)
		}
		fmt.Println(state)
	case "netState":
		state, err := client.GetNicConfig(nic)
		if err != nil {
			log.Fatalf("Error getting nic state: %v", err)
		}
		fmt.Printf("nic:\t%s\ndhcp:\t%v\nip:\t%s\nmask:\t%s\nmac:\t%s\n", nic, state.DHCPEnabled,
			state.IPAddress, net.IP(state.SubnetMask), state.MACAddress)
		if state.DefaultGateway != nil {
			fmt.Printf("gw:\t%s\n", state.DefaultGateway)
		}
		if state.PrimaryDNS != nil {
			fmt.Printf("dns1:\t%s\n", state.PrimaryDNS)
		}
		if state.SecondaryDNS != nil {
			fmt.Printf("dns2:\t%s\n", state.SecondaryDNS)
		}

	case "netConfig":
		config := &amt.NicState{DHCPEnabled: useDHCP}
		if !useDHCP {
			config.IPAddress = net.ParseIP(address)
			config.SubnetMask = net.IPMask(net.ParseIP(netmask))
			config.DefaultGateway = net.ParseIP(gateway)
			config.PrimaryDNS = net.ParseIP(dns1)
			config.SecondaryDNS = net.ParseIP(dns2)
			if config.IPAddress == nil {
				log.Fatalf("%s is not a valid ip address", address)
			}
			if config.SubnetMask == nil {
				log.Fatalf("%s is not a valid subnet mask", netmask)
			}
		}
		if err := client.SetNicConfig(nic, config); err != nil {
			log.Fatalf("Failed to set %s config: %v", nic, err)
		}
	default:
		log.Fatalf("Unknown op %s", op)
	}
	os.Exit(0)
}
