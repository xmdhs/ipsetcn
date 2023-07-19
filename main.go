package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net"
	"net/netip"
	"os"
	"runtime"
	"sync"

	"github.com/oschwald/maxminddb-golang"
	"github.com/xmdhs/ipsetcn/merger"
	"golang.org/x/sync/errgroup"
)

var (
	i   string
	o   string
	s   string
	asn string
)

func init() {
	flag.StringVar(&i, "i", "GeoLite2-Country.mmdb", "")
	flag.StringVar(&asn, "asn", "GeoLite2-ASN.mmdb", "")
	flag.StringVar(&o, "o", "cnipset.conf", "")
	flag.StringVar(&s, "s", "", "")
	flag.Parse()
}

func main() {
	r, err := maxminddb.Open(i)
	if err != nil {
		panic(err)
	}
	asnr, err := maxminddb.Open(asn)
	if err != nil {
		panic(err)
	}

	network := r.Networks(maxminddb.SkipAliasedNetworks)
	var ipm map[string]*[]*net.IPNet

	ctx := context.Background()

	if s == "" {
		var err error
		ipm, err = getLocIp(ctx, func() func(any, string, uint, bool) (string, bool) {
			return defaultFunc
		}, *network, asnr)
		if err != nil {
			panic(err)
		}
	} else {
		b, err := os.ReadFile(s)
		if err != nil {
			panic(err)
		}
		ipm, err = getLocIp(ctx, func() func(any, string, uint, bool) (string, bool) {
			f, err := NewJsFunc(string(b))
			if err != nil {
				panic(err)
			}
			return f
		}, *network, asnr)
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

type ASN struct {
	AutonomousSystemNumber uint `maxminddb:"autonomous_system_number"`
}

func getLocIp(ctx context.Context, needF func() func(any, string, uint, bool) (string, bool), network maxminddb.Networks, asnr *maxminddb.Reader) (m map[string]*[]*net.IPNet, err error) {
	m = map[string]*[]*net.IPNet{}
	ml := sync.Mutex{}
	g, _ := errgroup.WithContext(ctx)
	g.SetLimit(runtime.GOMAXPROCS(0))

	needPool := sync.Pool{}
	needPool.New = func() any {
		return needF()
	}

	for network.Next() {
		var r any
		ip, err := network.Network(&r)
		if err != nil {
			return nil, fmt.Errorf("getLocIp: %w", err)
		}
		g.Go(func() error {
			pre, err := netip.ParsePrefix(ip.String())
			if err != nil {
				return err
			}
			var asn ASN
			nip := net.ParseIP(pre.Addr().String())
			err = asnr.Lookup(nip, &asn)
			if err != nil {
				return err
			}
			need := needPool.Get().(func(any, string, uint, bool) (string, bool))
			tag, is := need(r, ip.String(), asn.AutonomousSystemNumber, pre.Addr().Is4())
			if !is {
				return nil
			}
			needPool.Put(need)
			_, n, err := net.ParseCIDR(pre.String())
			if err != nil {
				return err
			}
			ml.Lock()
			defer ml.Unlock()
			l, ok := m[tag]
			if !ok {
				l = &[]*net.IPNet{}
				m[tag] = l
			}
			*l = append(*l, n)
			return nil
		})
	}
	err = g.Wait()
	if err != nil {
		return nil, fmt.Errorf("getLocIp: %w", err)
	}
	for k, v := range m {
		new := sortNet(*v)
		m[k] = &new
	}
	return m, nil
}

func defaultFunc(a any, ipnet string, _ uint, ip4 bool) (tag string, b bool) {
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
