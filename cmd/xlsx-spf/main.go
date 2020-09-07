package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strings"

	"github.com/miekg/dns"
	"github.com/tealeg/xlsx"
)

var suppliers map[string]*regexp.Regexp

func init() {
	suppliers = make(map[string]*regexp.Regexp)
	suppliers["zendesk"] = regexp.MustCompile(`(?m)zendesk`)
	suppliers["freshdesk"] = regexp.MustCompile(`(?m)freshdesk`)
	suppliers["freshsales"] = regexp.MustCompile(`(?m)freshsales`)
}

func main() {
	file := flag.String("input", "", "file path. eg. ./fixtures/domain-test.xlsx")
	o := flag.String("output", "./output.xlsx", "eg. ./output.xlsx (defaults to output.xlsx)")
	c := flag.Int("col", 6, "describe what column to read domain from")
	flag.Parse()

	if *file == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *o == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	wb, err := xlsx.OpenFile(*file)
	if err != nil {
		panic(err)
	}

	sh := wb.Sheets[0]
	for _, r := range sh.Rows {
		e := r.Cells[*c]
		host, err := lookup(e.String())
		if err == nil {
			r.AddCell().SetString(host)
		}
	}

	fmt.Println(*o)

	wb.Save(*o)
}

func lookup(domain string) (string, error) {
	var res string
	config, _ := dns.ClientConfigFromFile("/etc/resolv.conf")
	c := new(dns.Client)

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(domain), dns.TypeTXT)
	m.RecursionDesired = true

	r, _, err := c.Exchange(m, net.JoinHostPort(config.Servers[0], config.Port))
	if r == nil {
		return "", err
	}

	if r.Rcode != dns.RcodeSuccess {
		log.Printf(" *** invalid answer name %s after MX query for %s\n", domain, domain)
	}
	// Stuff must be in the answer section
	for _, a := range r.Answer {
		res = res + a.String() + ","
	}

	for key, t := range suppliers {
		if t.MatchString(strings.ToLower(res)) {
			return key, nil
		}
	}

	return res, nil
}
