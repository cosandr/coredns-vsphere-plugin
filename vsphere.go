package vsphere

import (
	"context"
	"net"
	"net/url"
	"strings"

	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/session/cache"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
)

// Define log to be a logger with the plugin name in it. This way we can just use log.Info and
// friends to log.
var log = clog.NewWithPlugin("vsphere")

func NewClient(ctx context.Context, clientURL string, user string, pass string, insecure bool) (*vim25.Client, error) {
	// Parse URL from string
	u, err := soap.ParseURL(clientURL)
	if err != nil {
		return nil, err
	}

	u.User = url.UserPassword(user, pass)
	// Share govc's session cache
	s := &cache.Session{
		URL:      u,
		Insecure: insecure,
	}

	c := new(vim25.Client)
	err = s.Login(ctx, c, nil)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func NewVSphere(url string, user string, pass string, insecure bool) (VSphere, error) {
	ctx := context.Background()
	c, err := NewClient(ctx, url, user, pass, insecure)
	if err != nil {
		return VSphere{}, plugin.Error(pluginName, err)
	}
	return VSphere{Client: c}, nil
}

// VSphere is an vsphere plugin to show how to write a plugin.
type VSphere struct {
	Next plugin.Handler
	Client *vim25.Client
}
// Name implements the Handler interface.
func (v VSphere) Name() string { return pluginName }

func (v VSphere) Ready() bool { return v.Client != nil }

// ServeDNS implements the plugin.Handler interface. This method gets called when vsphere is used
// in a Server.
func (v VSphere) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	log.Debug("Received response")
	pw := NewResponsePrinter(w)

	m := view.NewManager(v.Client)

	cv, err := m.CreateContainerView(ctx, v.Client.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		log.Error(err)
		return plugin.NextOrFailure(v.Name(), v.Next, ctx, pw, r)
	}

	defer cv.Destroy(ctx)

	var vms []mo.VirtualMachine
	// Get only running VMs
	// Fetch all to cache and also check both vSphere name and hostname
	filters := property.Filter{
		"summary.config.template": "false",
		"summary.runtime.powerState": "poweredOn",
	}
	objs, err := cv.Find(ctx, []string{"VirtualMachine"}, filters)
	pc := property.DefaultCollector(cv.Client())

	fields := []string{"summary.guest.hostName", "summary.guest.ipAddress", "summary.config.name"}
	log.Debug("fetching VMs from API")
	err = pc.Retrieve(ctx, objs, fields, &vms)

	state := request.Request{W: w, Req: r}
	search := strings.TrimRight(state.QName(), ".")
	ipAddress := ""
	log.Debugf("got %d VMs, searching for %s", len(vms), search)
	for _, vm := range vms {
		if vm.Summary.Config.Name == search || vm.Summary.Guest.HostName == search {
			ipAddress = vm.Summary.Guest.IpAddress
			break
		}
	}
	if ipAddress == "" {
		log.Debugf("did not find %s", search)
		return plugin.NextOrFailure(v.Name(), v.Next, ctx, pw, r)
	}
	log.Debugf("found %s: %s", search, ipAddress)
	rec := new(dns.A)
	rec.Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 3600}
	rec.A = net.ParseIP(ipAddress)
	var answers []dns.RR
	answers = append(answers, rec)
	man := new(dns.Msg)
	man.Answer = answers
	man.SetReply(r)
	err = w.WriteMsg(man)

	if err != nil {
		log.Error(err)
		return plugin.NextOrFailure(v.Name(), v.Next, ctx, pw, r)
	}

	return dns.RcodeSuccess, nil
}


// ResponsePrinter wrap a dns.ResponseWriter and will write vsphere to standard output when WriteMsg is called.
type ResponsePrinter struct {
	dns.ResponseWriter
}

// NewResponsePrinter returns ResponseWriter.
func NewResponsePrinter(w dns.ResponseWriter) *ResponsePrinter {
	return &ResponsePrinter{ResponseWriter: w}
}

// WriteMsg calls the underlying ResponseWriter's WriteMsg method and prints "vsphere" to standard output.
func (r *ResponsePrinter) WriteMsg(res *dns.Msg) error {
	log.Info("vsphere")
	return r.ResponseWriter.WriteMsg(res)
}