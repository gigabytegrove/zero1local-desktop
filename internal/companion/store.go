package companion

import (
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "slices"
 "sync"
 "time"
 "github.com/gigabytegrove/zero1local-desktop/internal/appliance"
)

type InstalledDevice struct {
 ID string `json:"id"`
 URL string `json:"url"`
 Name string `json:"name"`
 Version string `json:"version"`
 AddedAt time.Time `json:"addedAt"`
}
type Mapping struct {
 ID string `json:"id"`
 DeviceID string `json:"deviceId"`
 Protocol string `json:"protocol"`
 Server string `json:"server"`
 Remote string `json:"remote"`
 Target string `json:"target"`
 AddedAt time.Time `json:"addedAt"`
}
type Config struct {
 Theme string `json:"theme"`
 Devices []InstalledDevice `json:"devices"`
 Mappings []Mapping `json:"mappings"`
}
type Store struct {
 mu sync.RWMutex
 path string
 data Config
}
func deviceID(address string) string {
 sum:=sha256.Sum256([]byte(address))
 return hex.EncodeToString(sum[:10])
}
func newStore(path string)(*Store,error){
 s:=&Store{path:path,data:Config{Theme:"system",Devices:[]InstalledDevice{},Mappings:[]Mapping{}}}
 b,err:=os.ReadFile(path)
 if errors.Is(err,os.ErrNotExist){return s,nil}
 if err!=nil{return nil,err}
 if len(b)>1024*1024{return nil,errors.New("settings file too large")}
 if err=json.Unmarshal(b,&s.data);err!=nil{return nil,fmt.Errorf("invalid settings: %w",err)}
 if s.data.Theme!="system" && s.data.Theme!="light" && s.data.Theme!="dark"{s.data.Theme="system"}
 if s.data.Devices==nil{s.data.Devices=[]InstalledDevice{}}
 if s.data.Mappings==nil{s.data.Mappings=[]Mapping{}}
 for _,d:=range s.data.Devices {
  normalized,e:=appliance.NormalizeEndpoint(d.URL)
  if e!=nil || normalized!=d.URL || deviceID(d.URL)!=d.ID{return nil,errors.New("stored appliance has invalid identity")}
 }
 return s,nil
}
func DefaultStore()(*Store,error){
 dir,err:=os.UserConfigDir();if err!=nil{return nil,err}
 return newStore(filepath.Join(dir,"Zero1Local Desktop","settings.json"))
}
func (s *Store) Snapshot() Config {
 s.mu.RLock();defer s.mu.RUnlock()
 out:=s.data
 out.Devices=slices.Clone(out.Devices)
 out.Mappings=slices.Clone(out.Mappings)
 return out
}
func (s *Store) writeLocked()error{
 dir:=filepath.Dir(s.path)
 if err:=os.MkdirAll(dir,0700);err!=nil{return err}
 b,err:=json.MarshalIndent(s.data,"","  ")
 if err!=nil{return err}
 f,err:=os.CreateTemp(dir,"settings-*.tmp");if err!=nil{return err}
 defer os.Remove(f.Name())
 defer f.Close()
 if err=f.Chmod(0600);err!=nil{return err}
 if _,err=f.Write(append(b,'
'));err!=nil{return err}
 if err=f.Sync();err!=nil{return err}
 if err=f.Close();err!=nil{return err}
 return os.Rename(f.Name(),s.path)
}
func (s *Store) Update(fn func(*Config) error)error{
 s.mu.Lock();defer s.mu.Unlock()
 previous:=s.data
 previous.Devices=slices.Clone(previous.Devices)
 previous.Mappings=slices.Clone(previous.Mappings)
 if err:=fn(&s.data);err!=nil{s.data=previous;return err}
 if err:=s.writeLocked();err!=nil{s.data=previous;return err}
 return nil
}
func (s *Store) Device(id string)(InstalledDevice,bool){
 s.mu.RLock();defer s.mu.RUnlock()
 for _,d:=range s.data.Devices{if d.ID==id{return d,true}}
 return InstalledDevice{},false
}
func addDevice(c *Config,d appliance.Device)(InstalledDevice,error){
 if len(c.Devices)>=64{return InstalledDevice{},errors.New("maximum 64 appliances")}
 v:=InstalledDevice{ID:deviceID(d.URL),URL:d.URL,Name:d.Name,Version:d.Version,AddedAt:time.Now().UTC()}
 for _,x:=range c.Devices{if x.ID==v.ID{return InstalledDevice{},errors.New("appliance already saved")}}
 c.Devices=append(c.Devices,v)
 return v,nil
}
