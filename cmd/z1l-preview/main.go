// z1l-preview runs the same local-only Go UI and APIs as the native app.
// The preview runs in a browser and does not provide Wails tray integration.
package main
import(
 "context"
 "fmt"
 "log"
 "os"
 "os/signal"
 "syscall"
 "time"
 "github.com/gigabytegrove/zero1local-desktop/internal/companion"
)
func main(){
 store,err:=companion.DefaultStore();if err!=nil{log.Fatal(err)}
 svc,err:=companion.New(store,nil);if err!=nil{log.Fatal(err)}
 address,err:=svc.Start();if err!=nil{log.Fatal(err)}
 fmt.Println("Zero1Local Desktop local preview:",address)
 signals:=make(chan os.Signal,1)
 signal.Notify(signals,os.Interrupt,syscall.SIGTERM)
 <-signals
 ctx,cancel:=context.WithTimeout(context.Background(),5*time.Second);defer cancel()
 _=svc.Shutdown(ctx)
}
