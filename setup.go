package vsphere

import (
	"errors"
	"strconv"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

const pluginName = "vsphere"

// init registers this plugin.
func init() { plugin.Register("vsphere", setup) }

func setup(c *caddy.Controller) error {
	url := ""
	user := ""
	pass := ""
	insecure := false
	var err error
	for c.Next() {
		if c.NextBlock() {
			for {
				switch c.Val() {
				case "url":
					if !c.NextArg() {
						return plugin.Error(pluginName, c.ArgErr())
					}
					url = c.Val()
				case "user":
					if !c.NextArg() {
						return plugin.Error(pluginName, c.ArgErr())
					}
					user = c.Val()
				case "pass":
					if !c.NextArg() {
						return plugin.Error(pluginName, c.ArgErr())
					}
					pass = c.Val()
				case "insecure":
					if !c.NextArg() {
						return plugin.Error(pluginName, c.ArgErr())
					}
					insecure, err = strconv.ParseBool(c.Val())
					if err != nil {
						return plugin.Error(pluginName, err)
					}
				}
				if !c.Next() {
					break
				}
			}
		}
	}
	if url == "" || user == "" || pass == "" {
		return plugin.Error(pluginName, errors.New("could not parse config"))
	}
	vs, err := NewVSphere(url, user, pass, insecure)
	if err != nil {
		return plugin.Error(pluginName, err)
	}
	// Add the Plugin to CoreDNS, so Servers can use it in their plugin chain.
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		vs.Next = next
		return vs
	})

	// All OK, return a nil error.
	return nil
}