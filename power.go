package intelamt

import (
	"errors"
	"fmt"
	"log"

	"github.com/VictorLowther/simplexml/dom"
	s "github.com/VictorLowther/simplexml/search"
)

func (c *Client) findSystemEPR() *dom.Element {
	msg := c.EnumerateEPR("http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_ComputerSystem")
	repl, err := msg.Send()
	if err != nil {
		log.Fatalf("Error enumerating EPR for ComputerSystem: %v", err)
	}
	return s.MustFirstTag("EndpointReference", "*", repl.AllBodyElements())
}

// GetChassisPower gets the current power state of the chassis that
// the AMT firmware is managing.
func (c *Client) GetChassisPower() (string, error) {
	epr := c.findSystemEPR()
	msg := c.Get("http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_AssociatedPowerManagementService")
	userSel := msg.MakeSelector("UserOfService")
	userSel.AddChildren(epr.Children()...)
	repl, err := msg.Send()
	if err != nil {
		return "", fmt.Errorf("Error getting power state: %v", err)
	}
	powerState := s.FirstTag("PowerState", "*", repl.AllBodyElements())
	if powerState == nil {
		return "", errors.New("WSMAN did not return power state")
	}

	switch string(powerState.Content) {
	case "1":
		return "unknown", nil
	case "2":
		return "on", nil
	case "3", "4", "7":
		return "sleep", nil
	default:
		return "off", nil
	}
}

// SetChassisPower sets the managed chassis to a particular state.
// Currently we only handle "on" and "off"
func (c *Client) SetChassisPower(state string) error {
	var param string
	switch state {
	case "on":
		param = "2"
	case "off":
		param = "8"
	default:
		return fmt.Errorf("Unknown power state %s", state)
	}
	epr := c.findSystemEPR()
	msg := c.Invoke("http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_PowerManagementService", "RequestPowerStateChange")
	msg.Parameters("PowerState", param)
	managedSystem := msg.MakeParameter("ManagedElement")
	managedSystem.AddChildren(epr.Children()...)
	msg.AddParameter(managedSystem)
	repl, err := msg.Send()
	if err != nil {
		return fmt.Errorf("Error setting power state: %v", err)
	}
	retVal := s.FirstTag("ReturnValue", "*", repl.AllBodyElements())
	if retVal == nil {
		return errors.New("WSMAN return did not include ReturnValue")
	}
	if string(retVal.Content) != "0" {
		return fmt.Errorf("Failed to change power state to %s (error %s)",
			state,
			string(retVal.Content))
	}
	return nil
}
