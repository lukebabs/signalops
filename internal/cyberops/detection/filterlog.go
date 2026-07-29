package detection

import (
	"net/netip"
	"strconv"
	"strings"
)

type FirewallEvent struct {
	Action          string
	Protocol        string
	SourceIP        string
	DestinationIP   string
	DestinationPort int
}

func ParseOPNsenseFilterlog(message string) (FirewallEvent, bool) {
	parts := strings.Split(message, ",")
	actionIndex := -1
	action := ""
	for i, value := range parts {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "pass" || value == "allow" {
			actionIndex = i
			action = value
			break
		}
	}
	if actionIndex < 0 {
		return FirewallEvent{}, false
	}
	protocolIndex := -1
	for i := actionIndex + 1; i < len(parts); i++ {
		value := strings.ToLower(strings.TrimSpace(parts[i]))
		if value == "tcp" || value == "udp" {
			protocolIndex = i
			break
		}
	}
	if protocolIndex < 0 || protocolIndex+5 >= len(parts) {
		return FirewallEvent{}, false
	}
	source := strings.TrimSpace(parts[protocolIndex+2])
	destination := strings.TrimSpace(parts[protocolIndex+3])
	port, err := strconv.Atoi(strings.TrimSpace(parts[protocolIndex+5]))
	if err != nil || port < 1 || port > 65535 {
		return FirewallEvent{}, false
	}
	if _, err := netip.ParseAddr(source); err != nil {
		return FirewallEvent{}, false
	}
	if _, err := netip.ParseAddr(destination); err != nil {
		return FirewallEvent{}, false
	}
	return FirewallEvent{Action: action, Protocol: strings.ToLower(strings.TrimSpace(parts[protocolIndex])), SourceIP: source, DestinationIP: destination, DestinationPort: port}, true
}

func IsPublicRoutable(value string) bool {
	address, err := netip.ParseAddr(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	return !address.IsPrivate() && !address.IsLoopback() && !address.IsLinkLocalUnicast() && !address.IsLinkLocalMulticast() && !address.IsMulticast() && !address.IsUnspecified()
}
