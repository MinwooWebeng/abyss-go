package atype

import (
	"bytes"
	"strconv"
	"strings"
)

type AbyssAddress struct {
	Pubkey_hash string
	IP          string
	Port        uint16
	Path        string //always start with '/' or empty

	Text string
}

func IsValidIpv4Address(ip string) bool {
	return true //TODO
}

// TODO: ipv6 support
func MakeAbyssAddress(pubkey_hash string, ip string, port uint16, path string) (AbyssAddress, bool) {
	var address AbyssAddress
	if len(pubkey_hash) < 32 {
		return address, false
	}
	if !IsValidIpv4Address(ip) {
		return address, false
	}
	if port == 0 {
		return address, false
	}
	if path != "" && path[0] != '/' { //path can be empty
		return address, false
	}

	address.Pubkey_hash = pubkey_hash
	address.IP = ip
	address.Port = port
	address.Path = path

	var sb strings.Builder
	sb.WriteString("abyss:")
	sb.WriteString(pubkey_hash)
	sb.WriteString(":")
	sb.WriteString(ip)
	sb.WriteString(":")
	sb.WriteString(strconv.Itoa(int(port)))
	sb.WriteString(path)

	address.Text = sb.String()
	return address, true
}

func MakeAbyssAddress2(pubkey_hash string, ip_port string, path string) (AbyssAddress, bool) {
	split := bytes.LastIndexByte([]byte(ip_port), ':')
	port, err := strconv.Atoi(ip_port[split+1:])
	if err != nil || int(uint16(port)) != port {
		return AbyssAddress{}, false
	}
	return MakeAbyssAddress(pubkey_hash, ip_port[:split], uint16(port), path)
}

func ParseAbyssAddress(addr_str string) (AbyssAddress, bool) {
	var address AbyssAddress
	if !strings.HasPrefix(addr_str, "abyss:") {
		return address, false
	}
	remainder := addr_str[6:]

	//publickey
	next_pos := strings.Index(remainder, ":")
	if next_pos == -1 {
		return address, false
	}
	address.Pubkey_hash = remainder[:next_pos]
	if len(address.Pubkey_hash) < 32 {
		return address, false
	}
	remainder = remainder[next_pos+1:]

	//ip
	next_pos = strings.Index(remainder, ":")
	if next_pos == -1 {
		return address, false
	}
	address.IP = remainder[:next_pos]
	if !IsValidIpv4Address(address.IP) {
		return address, false
	}
	remainder = remainder[next_pos+1:]

	//port
	next_pos = strings.Index(remainder, "/")
	if next_pos == -1 {
		next_pos = len(remainder)
	}
	port_num, err := strconv.ParseInt(remainder[:next_pos], 10, 32)
	if err != nil || port_num <= 0 || port_num > 65535 {
		return address, false
	}
	address.Port = uint16(port_num)
	remainder = remainder[next_pos:]

	//path
	address.Path = remainder //can be empty

	address.Text = addr_str

	return address, true
}

func (a *AbyssAddress) MakeOtherPath(new_path string) (AbyssAddress, bool) {
	if new_path != "" && new_path[0] != '/' {
		return AbyssAddress{}, false
	}
	var sb strings.Builder
	sb.WriteString("abyss:")
	sb.WriteString(a.Pubkey_hash)
	sb.WriteString(":")
	sb.WriteString(a.IP)
	sb.WriteString(":")
	sb.WriteString(strconv.Itoa(int(a.Port)))
	sb.WriteString(new_path)
	return AbyssAddress{a.Pubkey_hash, a.IP, a.Port, a.Path, sb.String()}, true
}
