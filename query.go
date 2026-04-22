package main

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"
)

// Result is one resolver's answer (or error) for a single query.
type Result struct {
	Resolver Resolver
	Answers  []string
	Latency  time.Duration
	Err      error
}

// queryAll fans out a query across every resolver in parallel and collects
// the results in a slice aligned with the input order.
func queryAll(ctx context.Context, resolvers []Resolver, domain, recordType string, timeout time.Duration) []Result {
	results := make([]Result, len(resolvers))
	var wg sync.WaitGroup
	for i := range resolvers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = queryOne(ctx, resolvers[i], domain, recordType, timeout)
		}(i)
	}
	wg.Wait()
	return results
}

// queryOne asks a single resolver for a single record type.
func queryOne(ctx context.Context, r Resolver, domain, recordType string, timeout time.Duration) Result {
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			d := net.Dialer{Timeout: timeout}
			return d.DialContext(ctx, network, net.JoinHostPort(r.IP, "53"))
		},
	}
	qctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	answers, err := doLookup(qctx, resolver, recordType, domain)
	lat := time.Since(start)
	sort.Strings(answers)
	return Result{Resolver: r, Answers: answers, Latency: lat, Err: err}
}

// doLookup dispatches to the right net.Resolver method based on record type.
func doLookup(ctx context.Context, resolver *net.Resolver, recordType, domain string) ([]string, error) {
	switch recordType {
	case "A":
		ips, err := resolver.LookupIP(ctx, "ip4", domain)
		if err != nil {
			return nil, err
		}
		out := make([]string, 0, len(ips))
		for _, ip := range ips {
			out = append(out, ip.String())
		}
		return out, nil
	case "AAAA":
		ips, err := resolver.LookupIP(ctx, "ip6", domain)
		if err != nil {
			return nil, err
		}
		out := make([]string, 0, len(ips))
		for _, ip := range ips {
			out = append(out, ip.String())
		}
		return out, nil
	case "CNAME":
		cname, err := resolver.LookupCNAME(ctx, domain)
		if err != nil {
			return nil, err
		}
		return []string{strings.TrimSuffix(cname, ".")}, nil
	case "MX":
		mxs, err := resolver.LookupMX(ctx, domain)
		if err != nil {
			return nil, err
		}
		out := make([]string, 0, len(mxs))
		for _, mx := range mxs {
			out = append(out, fmt.Sprintf("%d %s", mx.Pref, strings.TrimSuffix(mx.Host, ".")))
		}
		return out, nil
	case "TXT":
		txts, err := resolver.LookupTXT(ctx, domain)
		if err != nil {
			return nil, err
		}
		return txts, nil
	case "NS":
		nss, err := resolver.LookupNS(ctx, domain)
		if err != nil {
			return nil, err
		}
		out := make([]string, 0, len(nss))
		for _, ns := range nss {
			out = append(out, strings.TrimSuffix(ns.Host, "."))
		}
		return out, nil
	case "SRV":
		_, srvs, err := resolver.LookupSRV(ctx, "", "", domain)
		if err != nil {
			return nil, err
		}
		out := make([]string, 0, len(srvs))
		for _, s := range srvs {
			out = append(out, fmt.Sprintf("%d %d %d %s",
				s.Priority, s.Weight, s.Port, strings.TrimSuffix(s.Target, ".")))
		}
		return out, nil
	}
	return nil, fmt.Errorf("unsupported type: %s", recordType)
}

// isConsistent returns true if every successful resolver returned the exact
// same set of answers. Errored resolvers are skipped.
func isConsistent(results []Result) bool {
	var ref string
	first := true
	for _, r := range results {
		if r.Err != nil {
			continue
		}
		sig := strings.Join(r.Answers, "|")
		if first {
			ref = sig
			first = false
		} else if sig != ref {
			return false
		}
	}
	return !first // need at least one successful result
}

// groupByAnswer buckets successful results by their answer signature.
func groupByAnswer(results []Result) map[string][]Result {
	groups := map[string][]Result{}
	for _, r := range results {
		if r.Err != nil {
			continue
		}
		sig := strings.Join(r.Answers, "|")
		groups[sig] = append(groups[sig], r)
	}
	return groups
}
