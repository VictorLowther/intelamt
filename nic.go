package intelamt

import (
	"errors"
	"fmt"
	"net"

	s "github.com/VictorLowther/simplexml/search"
)

const etherPath = "http://intel.com/wbem/wscim/1/amt-schema/1/AMT_EthernetPortSettings"

// NicState encapsulates the network adaptor state we want to show to
// the rest of the world.
type NicState struct {
	DHCPEnabled    bool
	DefaultGateway net.IP
	IPAddress      net.IP
	MACAddress     net.HardwareAddr
	PrimaryDNS     net.IP
	SecondaryDNS   net.IP
	SubnetMask     net.IPMask
}

func parseIP(c string) (res net.IP, err error) {
	res = net.ParseIP(c)
	if res == nil {
		err = fmt.Errorf("%s is not an IP address", c)
	}
	return res, err
}

func deviceString(device string) (string, error) {
	switch device {
	case "wired":
		return "Intel(r) AMT Ethernet Port Settings 0", nil
	case "wireless":
		return "Intel(r) AMT Ethernet Port Settings 1", nil
	default:
		return "", fmt.Errorf("Unknown nic device %s", device)
	}
}

// GetNicConfig gets the network configuration for a nic.
func (c *Client) GetNicConfig(device string) (res *NicState, err error) {
	instanceID, err := deviceString(device)
	if err != nil {
		return
	}

	msg := c.Get(etherPath).Selectors("InstanceID", instanceID)
	repl, err := msg.Send()
	if err != nil {
		return nil, fmt.Errorf("WSMAN failure: %v", err)
	}
	settings := s.FirstTag("AMT_EthernetPortSettings", etherPath, repl.AllBodyElements())
	if settings == nil {
		return nil, errors.New("WSMAN return did not include EthernetPortSettings")
	}
	res = &NicState{}
	for _, tag := range settings.Children() {
		content := string(tag.Content)
		switch tag.Name.Local {
		case "DHCPEnabled":
			res.DHCPEnabled = content == "true"
		case "DefaultGateway":
			res.DefaultGateway, err = parseIP(content)
			if err != nil {
				return
			}
		case "IPAddress":
			res.IPAddress, err = parseIP(content)
			if err != nil {
				return
			}
		case "SubnetMask":
			snm, err := parseIP(content)
			if err != nil {
				return res, err
			}
			res.SubnetMask = net.IPMask(snm)
		case "PrimaryDNS":
			res.PrimaryDNS, err = parseIP(content)
			if err != nil {
				return
			}
		case "SecondaryDNS":
			res.SecondaryDNS, err = parseIP(content)
			if err != nil {
				return
			}
		case "MACAddress":
			res.MACAddress, err = net.ParseMAC(content)
			if err != nil {
				return res, err
			}
		}
	}
	return
}

// SetNicConfig sets the new configuration for a nic.  Note that it
// can take up to 10 seconds to regain connectivity after changing the
// state of the nic.
func (c *Client) SetNicConfig(device string, settings *NicState) error {
	instanceID, err := deviceString(device)
	if err != nil {
		return err
	}
	msg := c.Put(etherPath)
	msg.Values("InstanceID", instanceID,
		"ElementName", "Intel(r) AMT Ethernet Port Settings")
	msg.Selectors("InstanceID", instanceID)
	// make sure the link stays up no matter the power state
	msg.Values("LinkPolicy", "1", "LinkPolicy", "14", "LinkPolicy", "16")
	msg.Values("LinkPreference", "1")
	if settings.DHCPEnabled {
		msg.Values("DHCPEnabled", "true")
	} else {
		msg.Values("DHCPEnabled", "false")
		if settings.IPAddress == nil || settings.SubnetMask == nil {
			return errors.New("Static config requires an IP address and a subnet mask")
		} else {
			msg.Values("IPAddress", settings.IPAddress.String(),
				"SubnetMask", net.IP(settings.SubnetMask).String())
		}
		if settings.DefaultGateway != nil {
			msg.Values("DefaultGateway", settings.DefaultGateway.String())
		}
		if settings.PrimaryDNS != nil {
			msg.Values("PrimaryDNS", settings.PrimaryDNS.String())
		}
		if settings.SecondaryDNS != nil {
			msg.Values("SecondaryDNS", settings.SecondaryDNS.String())
		}
	}
	_, err = msg.Send()
	return err
}
