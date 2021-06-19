# CoreDNS vSphere plugin

This plugin queries the vSphere API and looks for VM a matching name/hostname.

All VMs are fetched at once and cached, the cache is updated whenever a request is not found.

## Usage

To activate the plugin you need to compile CoreDNS with the plugin added
to `plugin.cfg`

```
vsphere:github.com/cosandr/coredns-vsphere-plugin
```

Then add it to Corefile:

```
. {
    vsphere {
        url "<URL to vSphere>"
        user "<username>"
        pass "<password>"
        insecure "<true/false, defaults to false>"
    }
}
```
