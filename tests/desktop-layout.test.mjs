import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {test} from "node:test";
import {runInNewContext} from "node:vm";

const root = new URL("../internal/companion/ui/", import.meta.url);
const source = readFileSync(new URL("app.js", root), "utf8");
const html = readFileSync(new URL("index.html", root), "utf8");
const css = readFileSync(new URL("app.css", root), "utf8");

class Element {
  constructor(tag="div") {
    this.tag=tag; this.children=[]; this.dataset={}; this.attributes={};
    this.listeners=new Map(); this.classList={toggle(){},add(){},remove(){}};
    this.textContent=""; this.value=""; this.open=false; this.hidden=false; this.disabled=false;
    this.className=""; this.id=""; this.files=[];
  }
  append(...children) {
    for(const child of children) this.children.push(child===null?"null":child===undefined?"undefined":child);
  }
  replaceChildren(...children){this.children=[];this.append(...children)}
  addEventListener(type,handler){this.listeners.set(type,handler)}
  setAttribute(name,value){this.attributes[name]=String(value);if(name==="id")this.id=String(value)}
  removeAttribute(name){delete this.attributes[name]}
  focus(){}
  showModal(){this.open=true}
  close(){this.open=false}
  querySelector(selector) {
    const walk=(n)=>{
      if(!(n instanceof Element))return null;
      if(selector==="input"&&n.tag==="input")return n;
      if(selector.startsWith("#")&&n.id===selector.slice(1))return n;
      for(const child of n.children){const found=walk(child);if(found)return found}
      return null;
    };
    return walk(this);
  }
  click(event={type:"click"}){const fn=this.listeners.get("click");if(fn)return fn(event)}
}

async function harness(devices=[],mappings=[],connected=true){
  const names=["main","device-rail","modal","modal-body","modal-title","modal-form",
    "modal-submit","toast","top-title","top-subtitle","btn-discover","btn-add",
    "appliance-switcher","topbar-presence","nav-management","device-navigation","workspace-switcher","device-nav-title","back-to-devices"];
  const els=new Map(names.map(name=>["#"+name,new Element(name)]));
  const navigation=["home","devices","files","shares","mounts","system","storage","network",
    "docker","hardware","tasks","users","services","backups","protection","activity","settings"]
    .map(name=>{const el=new Element("button");el.dataset.nav=name;return el});
  const settings={theme:"dark",devices,mappings};
  const state={settings,platform:"linux",sessions:Object.fromEntries((connected?devices:[]).map(d=>
    [d.id,{username:"root",role:"administrator"}])),updater:{supported:true,version:"0.1.0-beta.5"}};
  const document={
    documentElement:{dataset:{}},body:new Element("body"),
    querySelector:selector=>selector==='meta[name="csrf-token"]'?{content:"test-token"}:els.get(selector)||null,
    querySelectorAll:selector=>selector==="[data-nav]"?navigation:[],
    createElement:tag=>new Element(tag),
    createTextNode:text=>String(text)
  };
  const context={
    document, matchMedia:()=>({matches:true,addEventListener(){}}),
    fetch:async path=>{
      if(path==="/api/state")return {ok:true,json:async()=>state};
      if(path==="/api/nas/read")return {ok:true,json:async()=>({
        data:{hostname:"nas1",data_usage:{used:104857600,available:943718400,percent:10},
          root_usage:{used:20971520,available:104857600,percent:16.7},
          raid:[{name:"md127",health:"healthy",level:"raid1"}],mdstat:"md127 : active raid1"}
      })};
      throw new Error("Unexpected test request: "+path);
    },
    window:{location:{reload(){}},setTimeout(){},addEventListener(){},URL:{createObjectURL(){return ""}}},
    setTimeout(){},clearTimeout(){},URL,Event:class{constructor(type){this.type=type}}
  };
  runInNewContext(source,context,{filename:"app.js"});
  await new Promise(resolve=>setImmediate(resolve));
  return {els,navigation,document};
}

function allText(node){
 if(!(node instanceof Element))return String(node||"");
 return [node.textContent,...node.children.map(allText)].join(" ");
}

test("NAS-first navigation keeps the device switcher and groups the advanced sections",()=>{
 assert.match(html,/id="appliance-switcher"/);
 assert.match(html,/<details class="nav-management"/);
 assert.ok(html.indexOf('id="device-rail"')<html.indexOf('data-nav="home"'));
 assert.match(css,/\.nas-overview/);
 assert.match(css,/\.workspace-task/);
 assert.match(css,/\.capacity-bar/);
});

test("Add NAS click does not insert a PointerEvent into its address field",async()=>{
 const {els}=await harness();
 const click={type:"click",toString(){return "[object PointerEvent]"}};
 els.get("#btn-add").click(click);
 const address=els.get("#modal-body").querySelector("#add-url");
 assert.ok(address,"address field must be present");
 assert.equal(address.value,"","event object leaked into the address input");
 assert.equal(els.get("#modal").open,true,"NAS sign-in dialog did not open");
});

test("connected NAS renders task-first overview without raw nested JSON",async()=>{
 const {els}=await harness([{id:"nas1",name:"nas1",url:"http://192.168.0.145",version:"1.4.7"}]);
 assert.match(allText(els.get("#main")),/nas1/);
 assert.equal(els.get("#device-navigation").hidden,true);
 els.get("#device-rail").children[0].click();
 const content=allText(els.get("#main"));
 assert.equal(els.get("#device-navigation").hidden,false);
 assert.match(content,/Browse files/);
 assert.match(content,/Check storage/);
 assert.match(content,/Connect shared folders/);
 assert.doesNotMatch(content,/\[object (PointerEvent|Object)\]/);
 assert.equal(els.get("#appliance-switcher").value,"nas1");
 assert.match(els.get("#topbar-presence").textContent,/Signed in/);
});

test("storage view shows readable capacity and never appends literal null",async()=>{
 const {els,navigation}=await harness([{id:"nas1",name:"nas1",url:"http://192.168.0.145",version:"1.4.7"}]);
 els.get("#device-rail").children[0].click();
 const storage=navigation.find(n=>n.dataset.nav==="storage");
 storage.click();
 await new Promise(resolve=>setImmediate(resolve));
 const text=allText(els.get("#main"));
 assert.match(text,/Storage capacity/);
 assert.match(text,/Main storage/);
 assert.match(text,/RAID arrays/);
 assert.doesNotMatch(text,/(^|\s)null(\s|$)/);
});


test("every management view belongs to a selected NAS, not a mixed default workspace",async()=>{
 const nas1={id:"nas1",name:"Primary NAS",url:"http://192.168.0.145",version:"1.4.7"};
 const nas2={id:"nas2",name:"Archive NAS",url:"http://192.168.0.146",version:"1.4.7"};
 const mappings=[
  {id:"drive1",deviceId:"nas1",protocol:"smb",server:"primary",remote:"PrimaryDocs",target:"X:",reconnect:true},
  {id:"drive2",deviceId:"nas2",protocol:"smb",server:"archive",remote:"ArchiveOnly",target:"Y:",reconnect:false}
 ];
 const {els,navigation}=await harness([nas1,nas2],mappings);
 assert.equal(els.get("#device-navigation").hidden,true,"global landing must not show NAS navigation");
 assert.match(allText(els.get("#main")),/Primary NAS/);
 assert.match(allText(els.get("#main")),/Archive NAS/);
 const rail=els.get("#device-rail");
 rail.children[0].click();
 assert.equal(els.get("#device-navigation").hidden,false);
 assert.match(allText(els.get("#main")),/Primary NAS/);
 assert.doesNotMatch(allText(els.get("#main")),/ArchiveOnly/);
 navigation.find(n=>n.dataset.nav==="mounts").click();
 let body=allText(els.get("#main"));
 assert.match(body,/PrimaryDocs/);
 assert.doesNotMatch(body,/ArchiveOnly/);
 navigation.find(n=>n.dataset.nav==="devices").click();
 assert.equal(els.get("#device-navigation").hidden,true);
 els.get("#device-rail").children[1].click();
 navigation.find(n=>n.dataset.nav==="mounts").click();
 body=allText(els.get("#main"));
 assert.match(body,/ArchiveOnly/);
 assert.doesNotMatch(body,/PrimaryDocs/);
});

test("remembrance is an explicit per-device choice; preview cannot silently persist a password",async()=>{
 const {els}=await harness();
 els.get("#btn-add").click({type:"click"});
 const checkbox=els.get("#modal-body").querySelector("#remember-nas");
 assert.ok(checkbox,"NAS sign-in must present a 'Keep me signed in' preference");
 assert.equal(checkbox.disabled,true,"no native OS keyring means this option must be disabled");
 assert.match(allText(els.get("#modal-body")),/securely save|securely saves|secure password storage/i);
});
