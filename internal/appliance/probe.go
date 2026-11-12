package appliance

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net"
    "net/http"
    "net/url"
    "sort"
    "strconv"
    "strings"
    "sync"
    "time"
)

// Device is an untrusted candidate found via Zero1Local's public identity API.
// A successful probe is NOT proof of authentication or device trust.
type Device struct {
    URL string `json:"url"`
    Name string `json:"name"`
    Version string `json:"version"`
    Setup bool `json:"setup"`
}

type health struct {
    OK bool `json:"ok"`
    Service string `json:"service"`
    Version string `json:"version"`
    Setup bool `json:"setup"`
}

type status struct { DeviceName string `json:"device_name"` }

// NormalizeEndpoint only accepts a management origin, without userinfo,
// query strings, fragments, or API paths.
func NormalizeEndpoint(raw string) (string, error) {
    raw = strings.TrimSpace(raw)
    if raw == "" { return "", errors.New("empty appliance address") }
    if !strings.Contains(raw, "://") { raw = "http://" + raw }
    u, err := url.Parse(raw)
    if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
        return "", errors.New("expected an http(s) appliance origin without credentials or a path")
    }
    if p := u.Port(); p != "" {
        n, err := strconv.Atoi(p)
        if err != nil || n < 1 || n > 65535 { return "", errors.New("invalid appliance port") }
    }
    return strings.TrimSuffix(u.String(), "/"), nil
}

func newClient(timeout time.Duration) *http.Client {
    if timeout <= 0 { timeout = 1500 * time.Millisecond }
    transport := http.DefaultTransport.(*http.Transport).Clone()
    transport.Proxy = nil // Never send requests for private appliances through HTTP proxies.
    transport.MaxIdleConnsPerHost = 2
    return &http.Client{
        Timeout: timeout,
        Transport: transport,
        CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
    }
}

// Probe checks the real /healthz marker and obtains the optional device name
// from /api/status; it never authenticates, saves credentials or mutates.
func Probe(ctx context.Context, endpoint string, timeout time.Duration) (Device, error) {
    origin, err := NormalizeEndpoint(endpoint)
    if err != nil { return Device{}, err }
    client := newClient(timeout)
    defer client.CloseIdleConnections()
    var h health
    if err = getJSON(ctx, client, origin+"/healthz", &h); err != nil { return Device{}, fmt.Errorf("health probe: %w", err) }
    if !h.OK || h.Service != "zero1-local" { return Device{}, errors.New("endpoint did not identify as Zero1Local") }
    d := Device{URL: origin, Version: h.Version, Setup: h.Setup}
    var s status
    if err := getJSON(ctx, client, origin+"/api/status", &s); err == nil { d.Name = strings.TrimSpace(s.DeviceName) }
    if d.Name == "" { d.Name = origin }
    return d, nil
}

func getJSON(ctx context.Context, client *http.Client, rawURL string, out any) error {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
    if err != nil { return err }
    resp, err := client.Do(req)
    if err != nil { return err }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK { return fmt.Errorf("unexpected HTTP status %d", resp.StatusCode) }
    if ct := resp.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(strings.ToLower(ct), "application/json") { return errors.New("unexpected content type") }
    dec := json.NewDecoder(io.LimitReader(resp.Body, 16*1024))
    if err := dec.Decode(out); err != nil { return err }
    return nil
}

// LANHosts enumerates bounded current /24s of active private IPv4 interfaces.
// Wide routed VLAN searches must be requested explicitly.
func LANHosts() []string {
    interfaces, err := net.Interfaces()
    if err != nil { return nil }
    unique := map[string]struct{}{}
    hosts := []string{}
    for _, iface := range interfaces {
        if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 { continue }
        name := strings.ToLower(iface.Name)
        skip := false
        for _, prefix := range []string{"docker", "br-", "veth", "virbr", "wg", "tun", "tap", "tailscale", "zt", "cni", "flannel"} {
            if strings.HasPrefix(name, prefix) { skip = true; break }
        }
        if skip { continue }
        addrs, err := iface.Addrs()
        if err != nil { continue }
        for _, addr := range addrs {
            ipnet, ok := addr.(*net.IPNet)
            if !ok { continue }
            ip := ipnet.IP.To4()
            if ip == nil || !privateIPv4(ip) { continue }
            for n := 1; n < 255; n++ {
                if byte(n) == ip[3] { continue }
                target := net.IPv4(ip[0], ip[1], ip[2], byte(n)).String()
                if _, ok := unique[target]; ok { continue }
                unique[target] = struct{}{}
                hosts = append(hosts, target)
            }
        }
    }
    return hosts
}

func privateIPv4(ip net.IP) bool {
    v := ip.To4()
    return v != nil && (v[0] == 10 || v[0] == 172 && v[1] >= 16 && v[1] <= 31 || v[0] == 192 && v[1] == 168)
}

// Discover runs up to 24 bounded workers and aborts without leaking workers
// when the caller cancels the context. It never alters an appliance.
func Discover(ctx context.Context, hosts []string, timeout time.Duration) []Device {
    if hosts == nil { hosts = LANHosts() }
    if timeout <= 0 { timeout = 450 * time.Millisecond }
    type response struct{ device Device; ok bool }
    jobs := make(chan string)
    results := make(chan response)
    var workers sync.WaitGroup
    const concurrency = 24
    for i := 0; i < concurrency; i++ {
        workers.Add(1)
        go func() {
            defer workers.Done()
            for host := range jobs {
                var item response
                for _, port := range []string{"80", "8088"} {
                    d, err := Probe(ctx, "http://"+net.JoinHostPort(host, port), timeout)
                    if err == nil { item = response{d, true}; break }
                    if ctx.Err() != nil { break }
                }
                select { case results <- item: case <-ctx.Done(): return }
            }
        }()
    }
    go func() {
        defer close(jobs)
        for _, host := range hosts {
            select { case jobs <- host: case <-ctx.Done(): return }
        }
    }()
    go func() { workers.Wait(); close(results) }()
    unique := map[string]Device{}
    for item := range results {
        if item.ok { unique[item.device.URL] = item.device }
        if ctx.Err() != nil { break }
    }
    return values(unique)
}

func values(m map[string]Device) []Device {
    list := make([]Device, 0, len(m))
    for _, d := range m { list = append(list, d) }
    sort.Slice(list, func(i, j int) bool { return list[i].URL < list[j].URL })
    return list
}
