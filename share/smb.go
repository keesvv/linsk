package share

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/AlexSSD7/linsk/osspecifics"
	"github.com/AlexSSD7/linsk/vm"
	"github.com/pkg/errors"
)

const smbPort = 445

type SMBBackend struct {
	listenIP  net.IP
	sharePort *uint16
}

func NewSMBBackend(uc *UserConfiguration) (Backend, *VMShareOptions, error) {
	var ports []vm.PortForwardingRule
	var sharePortPtr *uint16
	if !uc.smbExtMode {
		sharePort, err := getNetworkSharePort(0)
		if err != nil {
			return nil, nil, errors.Wrap(err, "get network share port")
		}

		sharePortPtr = &sharePort

		ports = append(ports, vm.PortForwardingRule{
			HostIP:   uc.listenIP,
			HostPort: sharePort,
			VMPort:   smbPort,
		})
	}

	return &SMBBackend{
			listenIP:  uc.listenIP,
			sharePort: sharePortPtr,
		}, &VMShareOptions{
			Ports:     ports,
			EnableTap: uc.smbExtMode,
		}, nil
}

func (b *SMBBackend) Apply(ctx context.Context, sharePWD string, vc *VMShareContext) (string, error) {
	if b.sharePort != nil && vc.NetTapCtx != nil {
		return "", fmt.Errorf("conflict: configured to use a forwarded port but a net tap configuration was detected")
	}

	if b.sharePort == nil && vc.NetTapCtx == nil {
		return "", fmt.Errorf("no net tap configuration found")
	}

	err := vc.FileManager.StartSMB(sharePWD)
	if err != nil {
		return "", errors.Wrap(err, "start smb server")
	}

	var shareURL string
	switch {
	case b.sharePort != nil:
		shareURL = "smb://" + net.JoinHostPort(b.listenIP.String(), fmt.Sprint(*b.sharePort)) + "/linsk"
	case vc.NetTapCtx != nil:
		if osspecifics.IsWindows() {
			shareURL = `\\` + strings.ReplaceAll(vc.NetTapCtx.Net.GuestIP.String(), ":", "-") + ".ipv6-literal.net" + `\linsk`
		} else {
			shareURL = "smb://" + net.JoinHostPort(vc.NetTapCtx.Net.GuestIP.String(), fmt.Sprint(smbPort)) + "/linsk"
		}
	default:
		return "", fmt.Errorf("no port forwarding and net tap configured")
	}

	return shareURL, nil
}
