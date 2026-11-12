package companion
import(
 "context"
 "time"
 "github.com/gigabytegrove/zero1local-desktop/internal/appliance"
)
func(s *Server)scan(ctx context.Context)[]appliance.Device{return appliance.Discover(ctx,nil,600*time.Millisecond)}
func(s *Server)probe(ctx context.Context,address string)(appliance.Device,error){return appliance.Probe(ctx,address,3*time.Second)}
