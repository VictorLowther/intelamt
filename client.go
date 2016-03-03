package intelamt

import (
	"bytes"
	"errors"
	"fmt"

	s "github.com/VictorLowther/simplexml/search"
	"github.com/VictorLowther/wsman"
)

// Client is a thin wrapper around the wsman client.
// Pretty much all of the rest of the AMT methods hang off it for now.
type Client struct {
	*wsman.Client
}

// NewClient creates a new initialized client
func NewClient(endpoint, username, password string) *Client {
	return &Client{Client: wsman.NewClient(endpoint, username, password, true)}
}

// Identify checks that the endpoint we are talking to is a WSMAN
// endpoint for an AMT firmware
func (c *Client) Identify() error {
	reply, err := c.Client.Identify()
	if err != nil {
		return err
	}
	vendor := s.FirstTag("ProductVendor", "*", reply.AllBodyElements())
	version := s.FirstTag("ProductVersion", "*", reply.AllBodyElements())
	if vendor == nil || version == nil {
		return errors.New("Failed to get vendor and version from endpoint")
	}
	if !(bytes.HasPrefix(vendor.Content, []byte("Intel")) &&
		bytes.HasPrefix(version.Content, []byte("AMT"))) {
		return fmt.Errorf("Endpoint is '%s %s',not and Intel AMT endpoint",
			string(vendor.Content), string(version.Content))
	}
	return nil
}
