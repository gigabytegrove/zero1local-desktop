package main

import (
 "context"
 "errors"
 "fmt"
 "log"
 "os"
 "os/exec"
 "runtime"
 "time"
 "github.com/gigabytegrove/zero1local-desktop/internal/companion"
 "github.com/wailsapp/wails/v3/pkg/application"
 "github.com/wailsapp/wails/v3/pkg/events"
)

// External browser remains the secure fallback until NAS-origin WebView
// bridge isolation is verified on Windows, macOS and Linux.
func openExternal(address string) error {
 var name string
 var args []string
 switch runtime.GOOS {
 case "windows":
  name, args = "rundll32", []string{"url.dll,FileProtocolHandler", address}
 case "darwin":
  name, args = "open", []string{address}
 case "linux":
  name, args = "xdg-open", []string{address}
 default:
  return errors.New("unsupported desktop operating system")
 }
 cmd:=exec.Command(name,args...)
 if err:=cmd.Start();err!=nil{return fmt.Errorf("open system browser: %w",err)}
 return cmd.Process.Release()
}

func run() error {
 store,err:=companion.DefaultStore()
 if err!=nil{return fmt.Errorf("desktop settings: %w",err)}
 svc,err:=companion.New(store,openExternal)
 if err!=nil{return err}
 localURL,err:=svc.Start()
 if err!=nil{return err}
 defer func(){
  ctx,cancel:=context.WithTimeout(context.Background(),5*time.Second)
  defer cancel()
  if err:=svc.Shutdown(ctx);err!=nil{log.Print(err)}
 }()
 app:=application.New(application.Options{
  Name:"Zero1Local Desktop", Description:"Zero1Local multi-appliance desktop companion",
  Mac:application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed:false},
 })
 window:=app.Window.NewWithOptions(application.WebviewWindowOptions{
  Name:"desktop-workspace", Title:"Zero1Local Desktop",
  Width:1180,Height:790,MinWidth:660,MinHeight:480,
  BackgroundColour:application.NewRGB(242,245,248),
  URL:localURL,
 })
 window.Center()
 window.RegisterHook(events.Common.WindowClosing,func(e *application.WindowEvent){window.Hide();e.Cancel()})
 tray:=app.SystemTray.New()
 tray.SetTooltip("Zero1Local Desktop")
 menu:=app.Menu.New()
 menu.Add("Open Zero1Local Desktop").OnClick(func(_ *application.Context){window.Show();window.Focus()})
 menu.Add("Hide window").OnClick(func(_ *application.Context){window.Hide()})
 menu.AddSeparator()
 menu.Add("Quit Zero1Local Desktop").OnClick(func(_ *application.Context){app.Quit()})
 tray.SetMenu(menu)
 tray.OnClick(func(){window.Show();window.Focus()})
 window.Show()
 return app.Run()
}
func main(){if err:=run();err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}}
