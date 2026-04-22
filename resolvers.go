package main

// Resolver is a single public DNS resolver we can query over UDP/TCP port 53.
type Resolver struct {
	IP       string
	Provider string
}

// defaultResolvers is a geographically and organizationally diverse set of
// public DNS resolvers. Using a wide spread increases the chance of catching
// stale cached answers during propagation.
var defaultResolvers = []Resolver{
	{IP: "1.1.1.1", Provider: "Cloudflare"},
	{IP: "1.0.0.1", Provider: "Cloudflare (2)"},
	{IP: "8.8.8.8", Provider: "Google"},
	{IP: "8.8.4.4", Provider: "Google (2)"},
	{IP: "9.9.9.9", Provider: "Quad9"},
	{IP: "208.67.222.222", Provider: "OpenDNS"},
	{IP: "94.140.14.14", Provider: "AdGuard"},
	{IP: "77.88.8.8", Provider: "Yandex"},
	{IP: "185.228.168.9", Provider: "CleanBrowsing"},
	{IP: "8.26.56.26", Provider: "Comodo"},
	{IP: "4.2.2.2", Provider: "Level3"},
	{IP: "64.6.64.6", Provider: "Verisign"},
	{IP: "185.222.222.222", Provider: "DNS.SB"},
	{IP: "194.242.2.2", Provider: "Mullvad"},
}
