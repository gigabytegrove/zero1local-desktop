// z1l-discover is a diagnostic entry point for the portable discovery core.
// It does not register, authenticate to, or modify discovered NAS appliances.
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "io"
    "os"
    "time"

    "github.com/gigabytegrove/zero1local-desktop/internal/appliance"
)

func run(ctx context.Context, args []string, out io.Writer) error {
    fs := flag.NewFlagSet("z1l-discover", flag.ContinueOnError)
    fs.SetOutput(io.Discard)
    scan := fs.Bool("scan", false, "scan current private LAN /24 networks")
    probe := fs.String("probe", "", "probe one Zero1Local endpoint URL or IP")
    timeout := fs.Duration("timeout", 750*time.Millisecond, "per-endpoint HTTP timeout")
    if err := fs.Parse(args); err != nil { return fmt.Errorf("invalid options: %w", err) }
    if (*scan && *probe != "") || (!*scan && *probe == "") || fs.NArg() != 0 { return fmt.Errorf("specify exactly one of -scan or -probe ADDRESS") }
    if *timeout < 100*time.Millisecond || *timeout > 30*time.Second { return fmt.Errorf("timeout must be between 100ms and 30s") }
    var devices []appliance.Device
    if *probe != "" {
        d, err := appliance.Probe(ctx, *probe, *timeout)
        if err != nil { return err }
        devices = []appliance.Device{d}
    } else {
        devices = appliance.Discover(ctx, nil, *timeout)
    }
    enc := json.NewEncoder(out)
    enc.SetIndent("", "  ")
    return enc.Encode(devices)
}

func main() {
    if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(2)
    }
}
