package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/netip"
	"os"

	"github.com/oschwald/maxminddb-golang"
)

var (
	i string
	o string
)

func init() {
	flag.StringVar(&i, "i", "geoip.db", "")
	flag.StringVar(&o, "o", "cnipset.conf", "")
	flag.Parse()
}

func main() {
	r, err := maxminddb.Open(i)
	if err != nil {
		panic(err)
	}

	network := r.Networks(maxminddb.SkipAliasedNetworks)

	ip4, ip6, err := getLocIp("CN", *network)
	if err != nil {
		panic(err)
	}

	f, err := os.Create(o)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	bw := bufio.NewWriter(f)
	defer bw.Flush()

	bw.WriteString("create cn hash:net maxelem\n")
	bw.WriteString("create cn6 hash:net family inet6\n")

	for k := range ip4 {
		bw.WriteString("add cn " + k.String() + "\n")
	}
	for k := range ip6 {
		bw.WriteString("add cn6 " + k.String() + "\n")
	}
}

type record struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
}

func getLocIp(isoCode string, network maxminddb.Networks) (ip4 map[netip.Prefix]struct{}, ip6 map[netip.Prefix]struct{}, err error) {
	ip4 = make(map[netip.Prefix]struct{})
	ip6 = make(map[netip.Prefix]struct{})
	for network.Next() {
		var r record
		ip, err := network.Network(&r)
		if err != nil {
			return nil, nil, fmt.Errorf("getLocIp: %w", err)
		}
		if r.Country.ISOCode != isoCode {
			continue
		}
		pre, err := netip.ParsePrefix(ip.String())
		if err != nil {
			return nil, nil, fmt.Errorf("getLocIp: %w", err)
		}
		if pre.Addr().Is4() {
			ip4[pre] = struct{}{}
			continue
		}
		if pre.Addr().Is6() {
			ip6[pre] = struct{}{}
			continue
		}
	}
	return ip4, ip6, nil
}
