package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"net/netip"
	"os"

	"github.com/oschwald/maxminddb-golang"
	"github.com/xmdhs/ipsetcn/merger"
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

	bw.WriteString("create cn hash:net\n")
	bw.WriteString("create cn6 hash:net family inet6\n")

	for _, v := range ip4 {
		bw.WriteString("add cn " + v.String() + "\n")
	}
	for _, v := range ip6 {
		bw.WriteString("add cn6 " + v.String() + "\n")
	}
}

type record struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
}

func getLocIp(isoCode string, network maxminddb.Networks) (ip4 []*net.IPNet, ip6 []*net.IPNet, err error) {
	ip4 = make([]*net.IPNet, 0)
	ip6 = make([]*net.IPNet, 0)
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
		_, n, err := net.ParseCIDR(pre.String())
		if err != nil {
			panic(err)
		}
		if pre.Addr().Is4() {
			ip4 = append(ip4, n)
			continue
		}
		if pre.Addr().Is6() {
			ip6 = append(ip6, n)
			continue
		}
	}
	ip4m := []merger.IRange{}
	ip6m := []merger.IRange{}

	for _, v := range ip4 {
		ip4m = append(ip4m, merger.IpNetWrapper{IPNet: v})
	}
	for _, v := range ip6 {
		ip6m = append(ip6m, merger.IpNetWrapper{IPNet: v})
	}

	ip4m = merger.SortAndMerge(ip4m)
	ip6m = merger.SortAndMerge(ip6m)

	ip4 = make([]*net.IPNet, 0)
	ip6 = make([]*net.IPNet, 0)

	for _, v := range ip4m {
		ip4 = append(ip4, v.ToIpNets()...)
	}
	for _, v := range ip6m {
		ip6 = append(ip6, v.ToIpNets()...)
	}

	return ip4, ip6, nil
}
