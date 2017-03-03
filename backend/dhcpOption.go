package backend

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"html/template"
	"net"
	"strconv"
	"strings"

	dhcp "github.com/krolaw/dhcp4"
)

func ConvertByteToOptionValue(code dhcp.OptionCode, b []byte) string {
	switch code {
	// Single IP-like address
	case dhcp.OptionSubnetMask,
		dhcp.OptionBroadcastAddress,
		dhcp.OptionSwapServer,
		dhcp.OptionRouterSolicitationAddress,
		dhcp.OptionRequestedIPAddress,
		dhcp.OptionServerIdentifier:
		return net.IP(b).To4().String()

	// Multiple IP-like address
	case dhcp.OptionRouter,
		dhcp.OptionTimeServer,
		dhcp.OptionNameServer,
		dhcp.OptionDomainNameServer,
		dhcp.OptionLogServer,
		dhcp.OptionCookieServer,
		dhcp.OptionLPRServer,
		dhcp.OptionImpressServer,
		dhcp.OptionResourceLocationServer,
		dhcp.OptionPolicyFilter, // This is special and could validate more (2Ips per)
		dhcp.OptionStaticRoute,  // This is special and could validate more (2IPs per)
		dhcp.OptionNetworkInformationServers,
		dhcp.OptionNetworkTimeProtocolServers,
		dhcp.OptionNetBIOSOverTCPIPNameServer,
		dhcp.OptionNetBIOSOverTCPIPDatagramDistributionServer,
		dhcp.OptionXWindowSystemFontServer,
		dhcp.OptionXWindowSystemDisplayManager,
		dhcp.OptionNetworkInformationServicePlusServers,
		dhcp.OptionMobileIPHomeAgent,
		dhcp.OptionSimpleMailTransportProtocol,
		dhcp.OptionPostOfficeProtocolServer,
		dhcp.OptionNetworkNewsTransportProtocol,
		dhcp.OptionDefaultWorldWideWebServer,
		dhcp.OptionDefaultFingerServer,
		dhcp.OptionDefaultInternetRelayChatServer,
		dhcp.OptionStreetTalkServer,
		dhcp.OptionStreetTalkDirectoryAssistance:

		addrs := make([]string, 0)
		for len(b) > 0 {
			addrs = append(addrs, net.IP(b[0:4]).To4().String())
			b = b[4:]
		}
		return strings.Join(addrs, ",")

	// String like value
	case dhcp.OptionHostName,
		dhcp.OptionMeritDumpFile,
		dhcp.OptionDomainName,
		dhcp.OptionRootPath,
		dhcp.OptionExtensionsPath,
		dhcp.OptionNetworkInformationServiceDomain,
		dhcp.OptionVendorSpecificInformation, // This is wrong, but ...
		dhcp.OptionNetBIOSOverTCPIPScope,
		dhcp.OptionNetworkInformationServicePlusDomain,
		dhcp.OptionTFTPServerName,
		dhcp.OptionBootFileName,
		dhcp.OptionMessage,
		dhcp.OptionVendorClassIdentifier,
		dhcp.OptionClientIdentifier,
		dhcp.OptionUserClass,
		dhcp.OptionTZPOSIXString,
		dhcp.OptionTZDatabaseString:
		return string(b[:len(b)])

	// 4 byte integer value
	case dhcp.OptionTimeOffset,
		dhcp.OptionPathMTUAgingTimeout,
		dhcp.OptionARPCacheTimeout,
		dhcp.OptionTCPKeepaliveInterval,
		dhcp.OptionIPAddressLeaseTime,
		dhcp.OptionRenewalTimeValue,
		dhcp.OptionRebindingTimeValue:
		return fmt.Sprint(binary.BigEndian.Uint32(b))

	// 2 byte integer value
	case dhcp.OptionBootFileSize,
		dhcp.OptionMaximumDatagramReassemblySize,
		dhcp.OptionInterfaceMTU,
		dhcp.OptionMaximumDHCPMessageSize,
		dhcp.OptionClientArchitecture:
		return fmt.Sprint(binary.BigEndian.Uint16(b))

	// 1 byte integer value
	case dhcp.OptionIPForwardingEnableDisable,
		dhcp.OptionNonLocalSourceRoutingEnableDisable,
		dhcp.OptionDefaultIPTimeToLive,
		dhcp.OptionAllSubnetsAreLocal,
		dhcp.OptionPerformMaskDiscovery,
		dhcp.OptionMaskSupplier,
		dhcp.OptionPerformRouterDiscovery,
		dhcp.OptionTrailerEncapsulation,
		dhcp.OptionEthernetEncapsulation,
		dhcp.OptionTCPDefaultTTL,
		dhcp.OptionTCPKeepaliveGarbage,
		dhcp.OptionNetBIOSOverTCPIPNodeType,
		dhcp.OptionOverload,
		dhcp.OptionDHCPMessageType:
		return fmt.Sprint(b[0])

		// Empty
	case dhcp.Pad, dhcp.End:
		return ""
	}

	return ""
}

func ConvertOptionValueToByte(code dhcp.OptionCode, value string) ([]byte, error) {
	switch code {
	// Single IP-like address
	case dhcp.OptionSubnetMask,
		dhcp.OptionBroadcastAddress,
		dhcp.OptionSwapServer,
		dhcp.OptionRouterSolicitationAddress,
		dhcp.OptionRequestedIPAddress,
		dhcp.OptionServerIdentifier:
		return []byte(net.ParseIP(value).To4()), nil

	// Multiple IP-like address
	case dhcp.OptionRouter,
		dhcp.OptionTimeServer,
		dhcp.OptionNameServer,
		dhcp.OptionDomainNameServer,
		dhcp.OptionLogServer,
		dhcp.OptionCookieServer,
		dhcp.OptionLPRServer,
		dhcp.OptionImpressServer,
		dhcp.OptionResourceLocationServer,
		dhcp.OptionPolicyFilter, // This is special and could validate more (2Ips per)
		dhcp.OptionStaticRoute,  // This is special and could validate more (2IPs per)
		dhcp.OptionNetworkInformationServers,
		dhcp.OptionNetworkTimeProtocolServers,
		dhcp.OptionNetBIOSOverTCPIPNameServer,
		dhcp.OptionNetBIOSOverTCPIPDatagramDistributionServer,
		dhcp.OptionXWindowSystemFontServer,
		dhcp.OptionXWindowSystemDisplayManager,
		dhcp.OptionNetworkInformationServicePlusServers,
		dhcp.OptionMobileIPHomeAgent,
		dhcp.OptionSimpleMailTransportProtocol,
		dhcp.OptionPostOfficeProtocolServer,
		dhcp.OptionNetworkNewsTransportProtocol,
		dhcp.OptionDefaultWorldWideWebServer,
		dhcp.OptionDefaultFingerServer,
		dhcp.OptionDefaultInternetRelayChatServer,
		dhcp.OptionStreetTalkServer,
		dhcp.OptionStreetTalkDirectoryAssistance:

		addrs := make([]net.IP, 0)
		alist := strings.Split(value, ",")
		for i := range alist {
			addrs = append(addrs, net.ParseIP(alist[i]).To4())
		}
		return dhcp.JoinIPs(addrs), nil

	// String like value
	case dhcp.OptionHostName,
		dhcp.OptionMeritDumpFile,
		dhcp.OptionDomainName,
		dhcp.OptionRootPath,
		dhcp.OptionExtensionsPath,
		dhcp.OptionNetworkInformationServiceDomain,
		dhcp.OptionVendorSpecificInformation, // This is wrong, but ...
		dhcp.OptionNetBIOSOverTCPIPScope,
		dhcp.OptionNetworkInformationServicePlusDomain,
		dhcp.OptionTFTPServerName,
		dhcp.OptionBootFileName,
		dhcp.OptionMessage,
		dhcp.OptionVendorClassIdentifier,
		dhcp.OptionClientIdentifier,
		dhcp.OptionUserClass,
		dhcp.OptionTZPOSIXString,
		dhcp.OptionTZDatabaseString:
		return []byte(value), nil

	// 4 byte integer value
	case dhcp.OptionTimeOffset,
		dhcp.OptionPathMTUAgingTimeout,
		dhcp.OptionARPCacheTimeout,
		dhcp.OptionTCPKeepaliveInterval,
		dhcp.OptionIPAddressLeaseTime,
		dhcp.OptionRenewalTimeValue,
		dhcp.OptionRebindingTimeValue:
		answer := make([]byte, 4)
		ival, err := strconv.Atoi(value)
		if err != nil {
			return nil, err
		}
		binary.BigEndian.PutUint32(answer, uint32(ival))
		return answer, nil

	// 2 byte integer value
	case dhcp.OptionBootFileSize,
		dhcp.OptionMaximumDatagramReassemblySize,
		dhcp.OptionInterfaceMTU,
		dhcp.OptionMaximumDHCPMessageSize:
		answer := make([]byte, 2)
		ival, err := strconv.Atoi(value)
		if err != nil {
			return nil, err
		}
		binary.BigEndian.PutUint16(answer, uint16(ival))
		return answer, nil

	// 1 byte integer value
	case dhcp.OptionIPForwardingEnableDisable,
		dhcp.OptionNonLocalSourceRoutingEnableDisable,
		dhcp.OptionDefaultIPTimeToLive,
		dhcp.OptionAllSubnetsAreLocal,
		dhcp.OptionPerformMaskDiscovery,
		dhcp.OptionMaskSupplier,
		dhcp.OptionPerformRouterDiscovery,
		dhcp.OptionTrailerEncapsulation,
		dhcp.OptionEthernetEncapsulation,
		dhcp.OptionTCPDefaultTTL,
		dhcp.OptionTCPKeepaliveGarbage,
		dhcp.OptionNetBIOSOverTCPIPNodeType,
		dhcp.OptionOverload,
		dhcp.OptionDHCPMessageType:
		answer := make([]byte, 1)
		ival, err := strconv.Atoi(value)
		if err != nil {
			return nil, err
		}
		answer[0] = byte(ival)
		return answer, nil

		// Empty
	case dhcp.Pad, dhcp.End:
		return make([]byte, 0), nil
	}

	return nil, errors.New("Invalid Option: " + code.String() + " " + value)
}

// DhcpOption is a representation of a specific DHCP option.
// swagger:model
type DhcpOption struct {
	// Code is a DHCP Option Code.
	//
	// required: true
	Code dhcp.OptionCode
	// Value is a text/template that will be expanded
	// and then converted into the proper format
	// for the option code
	//
	// required: true
	Value string
}

func (o *DhcpOption) RenderToDHCP(srcOpts map[int]string) (code dhcp.OptionCode, val []byte, err error) {
	code = o.Code
	tmpl, err := template.New("dhcp_option").Parse(o.Value)
	if err != nil {
		return code, nil, err
	}
	buf := &bytes.Buffer{}
	if err := tmpl.Execute(buf, srcOpts); err != nil {
		return code, nil, err
	}
	val, err = ConvertOptionValueToByte(code, buf.String())
	return code, val, err
}
