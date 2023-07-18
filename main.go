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
	s string
)

func init() {
	flag.StringVar(&i, "i", "geoip.db", "")
	flag.StringVar(&o, "o", "cnipset.conf", "")
	flag.StringVar(&s, "s", "", "")
	flag.Parse()
}

func main() {
	r, err := maxminddb.Open(i)
	if err != nil {
		panic(err)
	}

	network := r.Networks(maxminddb.SkipAliasedNetworks)
	var ipm map[string]*[]*net.IPNet

	if s == "" {
		var err error
		ipm, err = getLocIp(defaultFunc, *network)
		if err != nil {
			panic(err)
		}
	} else {
		b, err := os.ReadFile(s)
		if err != nil {
			panic(err)
		}
		f, err := NewJsFunc(string(b))
		if err != nil {
			panic(err)
		}
		ipm, err = getLocIp(f, *network)
		if err != nil {
			panic(err)
		}
	}

	f, err := os.Create(o)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	bw := bufio.NewWriter(f)
	defer bw.Flush()

	for k, v := range ipm {
		if len(*v) == 0 {
			continue
		}
		aip := (*v)[0]
		p, err := netip.ParsePrefix(aip.String())
		if err != nil {
			panic(err)
		}
		if p.Addr().Is4() {
			bw.WriteString(fmt.Sprintf("create %v hash:net\n", k))
		} else {
			bw.WriteString(fmt.Sprintf("create %v hash:net family inet6\n", k))
		}
		for _, v := range *v {
			bw.WriteString(fmt.Sprintf("add %v %v\n", k, v))
		}
	}
}

func getLocIp(need func(any, bool) (string, bool), network maxminddb.Networks) (m map[string]*[]*net.IPNet, err error) {
	m = map[string]*[]*net.IPNet{}
	for network.Next() {
		var r any
		ip, err := network.Network(&r)
		if err != nil {
			return nil, fmt.Errorf("getLocIp: %w", err)
		}
		pre, err := netip.ParsePrefix(ip.String())
		if err != nil {
			return nil, fmt.Errorf("getLocIp: %w", err)
		}

		tag, need := need(r, pre.Addr().Is4())
		if !need {
			continue
		}
		_, n, err := net.ParseCIDR(pre.String())
		if err != nil {
			panic(err)
		}
		l, ok := m[tag]
		if !ok {
			l = &[]*net.IPNet{}
			m[tag] = l
		}
		*l = append(*l, n)
	}
	for k, v := range m {
		new := sortNet(*v)
		m[k] = &new
	}
	return m, nil
}

func defaultFunc(a any, ip4 bool) (tag string, b bool) {
	c, ok := a.(map[string]any)
	if !ok {
		return "", false
	}
	country, ok := c["country"].(map[string]any)
	if !ok {
		return "", false
	}
	isocode, ok := country["iso_code"].(string)
	if !ok {
		return "", false
	}
	tag = "cn"
	if !ip4 {
		tag = "cn6"
	}
	return tag, isocode == "CN"
}

func sortNet(ipnet []*net.IPNet) []*net.IPNet {
	ipm := []merger.IRange{}
	for _, v := range ipnet {
		ipm = append(ipm, merger.IpNetWrapper{IPNet: v})
	}
	ipm = merger.SortAndMerge(ipm)

	newNet := make([]*net.IPNet, 0, len(ipnet))
	for _, v := range ipm {
		newNet = append(newNet, v.ToIpNets()...)
	}
	return newNet
}
