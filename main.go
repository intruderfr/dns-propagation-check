// dns-propagation-check queries multiple public DNS resolvers in parallel for
// a given domain/record type and reports whether they all agree.
//
// Exit codes:
//   0 — all successful resolvers returned identical answers
//   2 — resolvers disagreed (or watch mode timed out)
//   1 — invalid input / internal error
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// Options holds all CLI flag values plus the positional domain argument.
type Options struct {
	Domain     string
	RecordType string
	Timeout    time.Duration
	Watch      time.Duration
	MaxWait    time.Duration
	JSON       bool
	Resolvers  string
}

var supportedTypes = map[string]bool{
	"A": true, "AAAA": true, "CNAME": true, "MX": true,
	"TXT": true, "NS": true, "SRV": true,
}

func main() {
	var opts Options
	flag.StringVar(&opts.RecordType, "type", "A", "record type: A, AAAA, CNAME, MX, TXT, NS, SRV")
	flag.StringVar(&opts.RecordType, "t", "A", "short for --type")
	flag.DurationVar(&opts.Timeout, "timeout", 5*time.Second, "per-resolver query timeout")
	flag.DurationVar(&opts.Watch, "watch", 0, "poll interval; non-zero enables watch mode")
	flag.DurationVar(&opts.MaxWait, "max-wait", 10*time.Minute, "max watch duration before giving up")
	flag.BoolVar(&opts.JSON, "json", false, "emit JSON instead of a table")
	flag.StringVar(&opts.Resolvers, "resolvers", "", "comma-separated custom resolver IPs (overrides defaults)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: dns-propagation-check [flags] <domain>")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	opts.Domain = flag.Arg(0)

	rt := strings.ToUpper(opts.RecordType)
	if !supportedTypes[rt] {
		fmt.Fprintf(os.Stderr, "unsupported record type: %s (supported: A, AAAA, CNAME, MX, TXT, NS, SRV)\n", rt)
		os.Exit(1)
	}
	opts.RecordType = rt

	resolvers := defaultResolvers
	if opts.Resolvers != "" {
		resolvers = parseCustomResolvers(opts.Resolvers)
		if len(resolvers) == 0 {
			fmt.Fprintln(os.Stderr, "no valid resolvers parsed from --resolvers")
			os.Exit(1)
		}
	}

	if opts.Watch > 0 {
		os.Exit(runWatch(opts, resolvers))
	}
	os.Exit(runOnce(opts, resolvers))
}

func runOnce(opts Options, resolvers []Resolver) int {
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout+2*time.Second)
	defer cancel()

	results := queryAll(ctx, resolvers, opts.Domain, opts.RecordType, opts.Timeout)
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Resolver.Provider < results[j].Resolver.Provider
	})

	if opts.JSON {
		return renderJSON(os.Stdout, opts, results)
	}
	return renderTable(os.Stdout, opts, results)
}

func runWatch(opts Options, resolvers []Resolver) int {
	deadline := time.Now().Add(opts.MaxWait)
	iter := 0
	for {
		iter++
		ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout+2*time.Second)
		results := queryAll(ctx, resolvers, opts.Domain, opts.RecordType, opts.Timeout)
		cancel()

		consistent := isConsistent(results)
		if !opts.JSON {
			fmt.Printf("[iter %d] %s — %s\n",
				iter, time.Now().Format(time.RFC3339), consistencyLabel(consistent, results))
		}
		if consistent {
			if !opts.JSON {
				fmt.Println("Converged.")
			} else {
				renderJSON(os.Stdout, opts, results)
			}
			return 0
		}
		if time.Now().After(deadline) {
			if !opts.JSON {
				fmt.Fprintf(os.Stderr, "watch: exceeded max-wait (%s) without convergence\n", opts.MaxWait)
			} else {
				renderJSON(os.Stdout, opts, results)
			}
			return 2
		}
		time.Sleep(opts.Watch)
	}
}

func parseCustomResolvers(s string) []Resolver {
	out := []Resolver{}
	for _, ip := range strings.Split(s, ",") {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			continue
		}
		out = append(out, Resolver{IP: ip, Provider: "custom:" + ip})
	}
	return out
}
// dns-propagation-check queries multiple public DNS resolvers in parallel for
// a given domain/record type and reports whether they all agree.
//
// Exit codes:
//   0 — all successful resolvers returned identical answers
//   2 — resolvers disagreed (or watch mode timed out)
//   1 — invalid input / internal error
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// Options holds all CLI flag values plus the positional domain argument.
type Options struct {
	Domain     string
	RecordType string
	Timeout    time.Duration
	Watch      time.Duration
	MaxWait    time.Duration
	JSON       bool
	Resolvers  string
}

var supportedTypes = map[string]bool{
	"A": true, "AAAA": true, "CNAME": true, "MX": true,
	"TXT": true, "NS": true, "SRV": true,
}

func main() {
	var opts Options
	flag.StringVar(&opts.RecordType, "type", "A", "record type: A, AAAA, CNAME, MX, TXT, NS, SRV")
	flag.StringVar(&opts.RecordType, "t", "A", "short for --type")
	flag.DurationVar(&opts.Timeout, "timeout", 5*time.Second, "per-resolver query timeout")
	flag.DurationVar(&opts.Watch, "watch", 0, "poll interval; non-zero enables watch mode")
	flag.DurationVar(&opts.MaxWait, "max-wait", 10*time.Minute, "max watch duration before giving up")
	flag.BoolVar(&opts.JSON, "json", false, "emit JSON instead of a table")
	flag.StringVar(&opts.Resolvers, "resolvers", "", "comma-separated custom resolver IPs (overrides defaults)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: dns-propagation-check [flags] <domain>")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	opts.Domain = flag.Arg(0)

	rt := strings.ToUpper(opts.RecordType)
	if !supportedTypes[rt] {
		fmt.Fprintf(os.Stderr, "unsupported record type: %s (supported: A, AAAA, CNAME, MX, TXT, NS, SRV)\n", rt)
		os.Exit(1)
	}
	opts.RecordType = rt

	resolvers := defaultResolvers
	if opts.Resolvers != "" {
		resolvers = parseCustomResolvers(opts.Resolvers)
		if len(resolvers) == 0 {
			fmt.Fprintln(os.Stderr, "no valid resolvers parsed from --resolvers")
			os.Exit(1)
		}
	}

	if opts.Watch > 0 {
		os.Exit(runWatch(opts, resolvers))
	}
	os.Exit(runOnce(opts, resolvers))
}

func runOnce(opts Options, resolvers []Resolver) int {
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout+2*time.Second)
	defer cancel()

	results := queryAll(ctx, resolvers, opts.Domain, opts.RecordType, opts.Timeout)
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Resolver.Provider < results[j].Resolver.Provider
	})

	if opts.JSON {
		return renderJSON(os.Stdout, opts, results)
	}
	return renderTable(os.Stdout, opts, results)
}

func runWatch(opts Options, resolvers []Resolver) int {
	deadline := time.Now().Add(opts.MaxWait)
	iter := 0
	for {
		iter++
		ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout+2*time.Second)
		results := queryAll(ctx, resolvers, opts.Domain, opts.RecordType, opts.Timeout)
		cancel()
		consistent := isConsistent(results)
		if !opts.JSON {
			fmt.Printf("[iter %d] %s — %s\n",
				iter, time.Now().Format(time.RFC3339), consistencyLabel(consistent, results))
		}
		if consistent {
			if !opts.JSON {
				fmt.Println("Converged.")
			} else {
				renderJSON(os.Stdout, opts, results)
			}
			return 0
		}
		if time.Now().After(deadline) {
			if !opts.JSON {
				fmt.Fprintf(os.Stderr, "watch: exceeded max-wait (%s) without convergence\n", opts.MaxWait)
			} else {
				renderJSON(os.Stdout, opts, results)
			}
			return 2
		}
		time.Sleep(opts.Watch)
	}
}

func parseCustomResolvers(s string) []Resolver {
	out := []Resolver{}
	for _, ip := range strings.Split(s, ",") {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			continue
		}
		out = append(out, Resolver{IP: ip, Provider: "custom:" + ip})
	}
	return out
}
