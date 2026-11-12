package main

import (
 "bytes"
 "context"
 "image"
 "image/png"
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
 var window *application.WebviewWindow
 app:=application.New(application.Options{
  SingleInstance:&application.SingleInstanceOptions{
   UniqueID:"net.gigabytegrove.zero1local.desktop",
   OnSecondInstanceLaunch:func(_ application.SecondInstanceData){
    if window!=nil{window.Show();window.Focus()}
   },
  },
  Name:"Zero1Local Desktop", Description:"Zero1Local multi-appliance desktop companion",
  Mac:application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed:false},
 })
 checkForUpdates, unavailable, err := configureUpdater(app)
 if err != nil { return fmt.Errorf("configure native updater: %w", err) }
 svc.ConfigureUpdater(currentVersion, updateChannel, checkForUpdates, unavailable)
 window=app.Window.NewWithOptions(application.WebviewWindowOptions{
  Name:"desktop-workspace", Title:"Zero1Local Desktop",
  Width:1180,Height:790,MinWidth:660,MinHeight:480,
  BackgroundColour:application.NewRGB(242,245,248),
  URL:localURL,
 })
 window.Center()
 window.RegisterHook(events.Common.WindowClosing,func(e *application.WindowEvent){window.Hide();e.Cancel()})
 tray:=app.SystemTray.New()
 if art,err:=companion.BrandLightPNG();err==nil {
  if icon,err:=squareIcon(art);err==nil{tray.SetIcon(icon)}
 }
 tray.SetTooltip("Zero1Local Desktop")
 menu:=app.Menu.New()
 menu.Add("Open Zero1Local Desktop").OnClick(func(_ *application.Context){window.Show();window.Focus()})
 menu.Add("Check for updates…").OnClick(func(_ *application.Context){
  if err := svc.StartUpdateCheck(); err != nil { log.Printf("update request: %v", err) }
 })
 menu.Add("Hide window").OnClick(func(_ *application.Context){window.Hide()})
 menu.AddSeparator()
 menu.Add("Quit Zero1Local Desktop").OnClick(func(_ *application.Context){app.Quit()})
 tray.SetMenu(menu)
 tray.OnClick(func(){window.Show();window.Focus()})
 window.Show()
 return app.Run()
}
func main(){if err:=run();err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}}


// squareIcon makes a small system-tray crop of the official first-party logo.
// Source artwork stays byte-for-byte unchanged inside the embedded assets.
func squareIcon(data []byte)([]byte,error){
 src,err:=png.Decode(bytes.NewReader(data))
 if err!=nil{return nil,err}
 b:=src.Bounds()
 side:=b.Dy()
 if b.Dx()<side{side=b.Dx()}
 if side<1{return nil,errors.New("empty source brand image")}
 icon:=image.NewRGBA(image.Rect(0,0,64,64))
 for y:=0;y<64;y++{
  for x:=0;x<64;x++{
   sx:=b.Min.X+x*side/64
   sy:=b.Min.Y+y*side/64
   icon.Set(x,y,src.At(sx,sy))
  }
 }
 var out bytes.Buffer
 if err:=png.Encode(&out,icon);err!=nil{return nil,err}
 return out.Bytes(),nil
}
