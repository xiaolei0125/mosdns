//     Copyright (C) 2020-2021, IrineSistiana
//
//     This file is part of mosdns.
//
//     mosdns is free software: you can redistribute it and/or modify
//     it under the terms of the GNU General Public License as published by
//     the Free Software Foundation, either version 3 of the License, or
//     (at your option) any later version.
//
//     mosdns is distributed in the hope that it will be useful,
//     but WITHOUT ANY WARRANTY; without even the implied warranty of
//     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//     GNU General Public License for more details.
//
//     You should have received a copy of the GNU General Public License
//     along with this program.  If not, see <https://www.gnu.org/licenses/>.

package hosts

import (
	"context"
	"errors"
	"fmt"
	"github.com/IrineSistiana/mosdns/dispatcher/handler"
	"github.com/IrineSistiana/mosdns/dispatcher/matcher/domain"
	"github.com/miekg/dns"
	"net"
)

const PluginType = "hosts"

func init() {
	handler.RegInitFunc(PluginType, Init, func() interface{} { return new(Args) })
}

var _ handler.ESExecutablePlugin = (*hostsContainer)(nil)
var _ handler.ContextPlugin = (*hostsContainer)(nil)

type Args struct {
	Hosts []string `yaml:"hosts"`
}

type hostsContainer struct {
	*handler.BP

	matcher domain.Matcher
}

func Init(bp *handler.BP, args interface{}) (p handler.Plugin, err error) {
	return newHostsContainer(bp, args.(*Args))
}

var patternTypeMap = map[string]domain.MixMatcherPatternType{
	"domain":  domain.MixMatcherPatternTypeDomain,
	"keyword": domain.MixMatcherPatternTypeKeyword,
	"regexp":  domain.MixMatcherPatternTypeRegexp,
	"":        domain.MixMatcherPatternTypeFull,
	"full":    domain.MixMatcherPatternTypeFull,
}

func newHostsContainer(bp *handler.BP, args *Args) (*hostsContainer, error) {
	if len(args.Hosts) == 0 {
		return nil, errors.New("no hosts file is configured")
	}

	mixMatcher := domain.NewMixMatcher()
	mixMatcher.SetPattenTypeMap(patternTypeMap)
	err := domain.BatchLoadMatcher(mixMatcher, args.Hosts, parseIP)
	if err != nil {
		return nil, err
	}
	return &hostsContainer{
		BP:      bp,
		matcher: mixMatcher,
	}, nil
}

func (h *hostsContainer) ExecES(ctx context.Context, qCtx *handler.Context) (earlyStop bool, err error) {
	return h.matchAndSet(qCtx), nil
}

func (h *hostsContainer) Connect(ctx context.Context, qCtx *handler.Context, pipeCtx *handler.PipeContext) (err error) {
	if h.matchAndSet(qCtx) {
		return nil
	}
	return pipeCtx.ExecNextPlugin(ctx, qCtx)
}

func (h *hostsContainer) matchAndSet(qCtx *handler.Context) (matched bool) {
	if len(qCtx.Q().Question) != 1 {
		return false
	}

	typ := qCtx.Q().Question[0].Qtype
	fqdn := qCtx.Q().Question[0].Name
	v, ok := h.matcher.Match(fqdn)
	if !ok {
		return false
	}
	record := v.(*ipRecord)

	switch typ {
	case dns.TypeA:
		if len(record.ipv4) != 0 {
			r := new(dns.Msg)
			r.SetReply(qCtx.Q())
			for _, ip := range record.ipv4 {
				ipCopy := make(net.IP, len(ip))
				copy(ipCopy, ip)
				rr := &dns.A{
					Hdr: dns.RR_Header{
						Name:   fqdn,
						Rrtype: dns.TypeA,
						Class:  dns.ClassINET,
						Ttl:    3600,
					},
					A: ipCopy,
				}
				r.Answer = append(r.Answer, rr)
			}
			qCtx.SetResponse(r, handler.ContextStatusResponded)
			return true
		}

	case dns.TypeAAAA:
		if len(record.ipv6) != 0 {
			r := new(dns.Msg)
			r.SetReply(qCtx.Q())
			for _, ip := range record.ipv6 {
				ipCopy := make(net.IP, len(ip))
				copy(ipCopy, ip)
				rr := &dns.AAAA{
					Hdr: dns.RR_Header{
						Name:   fqdn,
						Rrtype: dns.TypeAAAA,
						Class:  dns.ClassINET,
						Ttl:    3600,
					},
					AAAA: ipCopy,
				}
				r.Answer = append(r.Answer, rr)
			}
			qCtx.SetResponse(r, handler.ContextStatusResponded)
			return true
		}
	}
	return false
}

type ipRecord struct {
	ipv4 []net.IP
	ipv6 []net.IP
}

func (r *ipRecord) Append(v interface{}) {
	n := v.(*ipRecord)
	r.ipv4 = append(r.ipv4, n.ipv4...)
	r.ipv6 = append(r.ipv6, n.ipv6...)
}

func parseIP(s []string) (v interface{}, accept bool, err error) {
	if len(s) == 0 {
		return nil, false, nil
	}

	record := new(ipRecord)
	for _, ipStr := range s {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return nil, false, fmt.Errorf("invalid ip addr %s", ipStr)
		}

		if ipv4 := ip.To4(); ipv4 != nil { // is ipv4
			record.ipv4 = append(record.ipv4, ipv4)
		} else if ipv6 := ip.To16(); ipv6 != nil { // is ipv6
			record.ipv6 = append(record.ipv6, ipv6)
		} else { // invalid
			return nil, false, fmt.Errorf("%s is not an ipv4 or ipv6 addr", ipStr)
		}
	}
	return record, true, nil
}
