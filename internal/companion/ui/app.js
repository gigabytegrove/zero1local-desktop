"use strict";
const csrf=document.querySelector('meta[name="csrf-token"]').content;
const main=document.querySelector("#main");
const rail=document.querySelector("#device-rail");
const modal=document.querySelector("#modal");
const modalBody=document.querySelector("#modal-body");
const navigation=[...document.querySelectorAll("[data-nav]")];
const S={settings:{theme:"system",devices:[],mappings:[]},platform:"linux",view:"home",selected:"",busy:false,scan:[],inventory:null};

const getDevice=id=>S.settings.devices.find(d=>d.id===id);
const selected=()=>getDevice(S.selected);
function node(tag,attrs={},...kids){
 const n=document.createElement(tag);
 for(const [key,value] of Object.entries(attrs)){
  if(key==="class")n.className=value;
  else if(key==="text")n.textContent=value;
  else if(key==="onClick")n.addEventListener("click",value);
  else if(key==="onChange")n.addEventListener("change",value);
  else if(key==="disabled")n.disabled=value;
  else if(key==="value")n.value=value;
  else n.setAttribute(key,value);
 }
 for(const k of kids.flat()){if(k!==undefined&&k!==null)n.append(typeof k==="string"?document.createTextNode(k):k)}
 return n;
}
function button(label,click,cls="btn btn-quiet",disabled=S.busy){return node("button",{text:label,type:"button",class:cls,disabled,onClick:click})}
function flash(msg){
 const x=document.querySelector("#toast");x.textContent=msg;x.classList.add("visible");
 clearTimeout(flash.timer);flash.timer=setTimeout(()=>x.classList.remove("visible"),5000);
}
async function request(path,payload){
 const opt=payload===undefined?{}:{method:"POST",headers:{"Content-Type":"application/json","X-Z1L-Token":csrf},body:JSON.stringify(payload)};
 const res=await fetch(path,opt);let body;
 try{body=await res.json()}catch{throw new Error("Unexpected response from local companion service")}
 if(!res.ok)throw new Error(body.error||"Operation failed ("+res.status+")");
 return body;
}
async function task(fn){
 if(S.busy)return;
 S.busy=true;render();
 try{await fn()}catch(e){flash(e.message||"Operation failed")}
 finally{S.busy=false;render()}
}
async function reload(){
 const result=await request("/api/state");
 S.settings=result.settings;S.platform=result.platform;
 if(S.selected&&!getDevice(S.selected))S.selected="";
 if(!S.selected&&S.settings.devices.length)S.selected=S.settings.devices[0].id;
 theme();render();
}
const systemTheme=matchMedia("(prefers-color-scheme: dark)");
function theme(){
 document.documentElement.dataset.theme=S.settings.theme==="system"?(systemTheme.matches?"dark":"light"):S.settings.theme;
}
systemTheme.addEventListener("change",()=>{if(S.settings.theme==="system")theme()});
function choose(id){S.selected=id;S.inventory=null;S.view="home";render()}
function heading(title,text){return node("div",{class:"page-heading"},node("div",{},node("h1",{text:title}),node("p",{text})))}
function section(title,content){return node("section",{class:"section"},node("div",{class:"section-title"},node("h2",{text:title})),content)}
function primary(title,subtitle){return node("div",{},node("strong",{text:title}),node("small",{text:subtitle}))}
function table(head,rows){
 return node("table",{class:"list-table"},
  node("thead",{},node("tr",{},...head.map(x=>node("th",{scope:"col",text:x})))),
  node("tbody",{},...rows.map(row=>node("tr",{},...row.map(v=>node("td",{},v))))
 );
}
function empty(symbol,title,description,action){
 return node("div",{class:"empty"},node("div",{class:"empty-symbol",text:symbol,"aria-hidden":"true"}),
  node("h2",{text:title}),node("p",{text:description}),action);
}
function field(label,id,placeholder,value="",type="text"){
 return node("label",{class:"field",for:id},
  node("span",{text:label}),node("input",{id,type,placeholder,value,autocomplete:"off"}));
}
function listRail(){
 rail.replaceChildren(...S.settings.devices.map(d=>node("button",{
  type:"button",class:"device-choice"+(d.id===S.selected?" selected":""),onClick:()=>choose(d.id)},
  node("span",{class:"connection-dot","aria-hidden":"true"}),
  node("span",{},node("strong",{text:d.name}),node("small",{text:d.url}))
 )));
 if(!S.settings.devices.length)rail.append(node("span",{class:"rail-empty",text:"No appliances saved"}));
}
function render(){
 listRail();
 navigation.forEach(b=>{
  const active=b.dataset.nav===S.view;b.classList.toggle("active",active);
  if(active)b.setAttribute("aria-current","page");else b.removeAttribute("aria-current");
 });
 const titles={home:"Overview",devices:"My appliances",shares:"Shares & access",mounts:"Mapped drives",settings:"Desktop settings"};
 document.querySelector("#top-title").textContent=titles[S.view];
 document.querySelector("#top-subtitle").textContent=selected()?.name||"Zero1Local on this computer";
 document.querySelector("#btn-discover").disabled=S.busy;
 document.querySelector("#btn-add").disabled=S.busy;
 switch(S.view){
 case "devices":main.replaceChildren(devicesPage());break;
 case "shares":main.replaceChildren(sharesPage());break;
 case "mounts":main.replaceChildren(mountsPage());break;
 case "settings":main.replaceChildren(settingsPage());break;
 default:main.replaceChildren(homePage());
 }
}
function homePage(){
 const d=selected(),maps=S.settings.mappings;
 const root=node("div",{},heading("Your Zero1Local workspace","All your NAS installations and connected shares, from one computer."),
 node("div",{class:"health-strip"},
  ...[[S.settings.devices.length,"Saved appliances"],[maps.length,"Saved mappings"],["Local","No cloud account"]].map(([value,name])=>
   node("div",{class:"health-item"},node("strong",{text:String(value)}),node("span",{text:name}))
 )));
 if(!d){
  root.append(empty("▣","Find your first Zero1Local","Discover local NAS devices or add an IP address or hostname.",button("Scan for Zero1Local",discover,"btn btn-primary")));
  return root;
 }
 root.append(section("Selected appliance",node("div",{class:"detail-grid"},
  node("div",{class:"detail-main"},node("h2",{text:d.name}),
   node("dl",{class:"facts"},
    node("dt",{text:"Management address"}),node("dd",{class:"path",text:d.url}),
    node("dt",{text:"Zero1Local version"}),node("dd",{text:d.version||"Not reported"}),
    node("dt",{text:"Connection"}),node("dd",{},node("span",{class:"state-label warn",text:"Saved; not continuously monitored"}))
   ),
   node("div",{class:"form-actions"},button("Open Zero1Local management",()=>openDevice(d.id),"btn btn-primary"),
    button("Connect shares",()=>{S.view="shares";render()}))
  ),
  node("div",{class:"detail-aside"},node("h2",{text:"Complete NAS management"}),
   node("p",{text:"The original Zero1Local console keeps its complete feature set, users, privileges, and MFA. Desktop share connections use the host operating system."}),
   node("p",{class:"hint",text:"The existing management UI currently opens in your system browser while remote WebView security isolation is qualified."}))
 )));
 root.append(section("Your saved mounts",maps.length?table(["Target","Share","Action"],
  maps.map(m=>[primary(m.target||"Desktop volume",m.protocol.toUpperCase()),node("span",{class:"path",text:m.server+"/"+m.remote}),
   button("Disconnect",()=>confirmUnmount(m))])
 ):empty("⌘","No mounts configured","Use Shares & access to connect a NAS folder.",button("Connect a share",()=>{S.view="shares";render()}))));
 return root;
}
function devicesPage(){
 const root=node("div",{},heading("My appliances","Zero1Local devices you chose to save. Discovery is read-only and does not automatically trust endpoints."));
 if(S.settings.devices.length)root.append(section("Saved installations",table(["Appliance","Version","Status","Actions"],
 S.settings.devices.map(d=>[
  primary(d.name,d.url),node("span",{text:d.version||"Unknown"}),
  node("span",{class:"state-label good",text:"Saved endpoint"}),
  node("div",{class:"actions"},
   button("Select",()=>choose(d.id)),button("Manage",()=>openDevice(d.id)),
   button("Forget",()=>forget(d),"btn btn-danger"))
 ]))));
 else root.append(empty("▣","No appliances yet","Scan your network or enter the NAS management address.",button("Scan network",discover,"btn btn-primary")));
 if(S.scan.length)root.append(section("Discovered candidates",table(["Appliance","Version","Address","Action"],
  S.scan.map(d=>[primary(d.name,d.url),node("span",{text:d.version}),
   node("span",{class:"path",text:d.url}),button("Save appliance",()=>task(async()=>{
    await request("/api/add",{url:d.url});await reload();flash("Appliance saved")
   }),"btn btn-primary")])
  )));
 return root;
}
function sharesPage(){
 const d=selected(),root=node("div",{},heading("Shares & access","Map shares using operating-system clients; the NAS always controls access."));
 if(!d){
  root.append(empty("▤","Choose an appliance","Save or select a NAS to connect its shares.",button("My appliances",()=>{S.view="devices";render()},"btn btn-primary")));
  return root;
 }
 const host=new URL(d.url).hostname;
 const targetLabel=S.platform==="windows"?"Drive letter (D:–Z:)":"Mount folder";
 const form=node("div",{},
  node("div",{class:"fields"},
   field("SMB server (LAN host or IPv4)","server","NAS hostname or LAN address",host),
   field("Share name or NFS export","remote","Example: Documents"),
   node("label",{class:"field"},node("span",{text:"Protocol"}),
    node("select",{id:"protocol"},
     node("option",{value:"smb",text:"SMB"}),
     node("option",{value:"nfs",text:"NFSv3 (Linux)",disabled:S.platform!=="linux"})
    )),
   field(targetLabel,"target",S.platform==="windows"?"Z:":"Managed by the operating system",
    S.platform==="windows"?"Z:":"")
  ),
  node("div",{class:"warning",text:"Use the NAS LAN address, not a reverse-proxy web address. No credentials are stored or sent to this desktop application. Some OS clients may require an existing saved login."}),
  node("div",{class:"form-actions"},button("Connect share",connectShare,"btn btn-primary"))
 );
 const p=form.querySelector("#protocol"),t=form.querySelector("#target");
 function updateMountTarget(){
  const nfs=p.value==="nfs";
  t.disabled=S.platform==="darwin"||(S.platform==="linux"&&!nfs);
  t.placeholder=nfs?"/home/you/nfs-share":S.platform==="windows"?"Z:":"Managed by OS";
 }
 p.addEventListener("change",updateMountTarget);updateMountTarget();
 root.append(section("Connect to "+d.name,node("div",{class:"detail-main"},form)));
 const inventory=node("div",{class:"detail-main"},
  node("h2",{text:"Existing NAS shares"}),
  node("p",{class:"hint",text:"For an HTTPS-connected NAS, enter a pre-existing read-only Zero1Local API key to list actual SMB and NFS exports. The key is used only for this request and is never saved."}),
  field("Zero1Local read-only API key","api-token","Paste key","","password"),
  node("div",{class:"form-actions"},button("Load existing shares",loadInventory,"btn btn-primary"))
 );
 if(!d.url.startsWith("https://"))
  inventory.append(node("p",{class:"warning",text:"Share listing is disabled for an HTTP management address. API keys are never transmitted without HTTPS."}));
 root.append(section("Read share inventory",inventory));
 if(S.inventory)root.append(section("Share results",inventoryPage()));
 return root;
}
function inventoryPage(){
 const root=node("div",{}),smb=S.inventory["/api/shares"],nfs=S.inventory["/api/nfs/exports"];
 if(smb?.available===false)root.append(node("p",{class:"warning",text:"SMB inventory returned HTTP "+smb.status+". Check API key permissions."}));
 else if(Array.isArray(smb?.active)&&smb.active.length){
  root.append(table(["SMB share","Folder","Action"],smb.active.map(x=>[
   primary(String(x.name||"Unnamed"),x.browseable?"Browseable":"Restricted"),
   node("span",{class:"path",text:String(x.path||"")}),
   button("Use",()=>{document.querySelector("#remote").value=String(x.name||"");document.querySelector("#server").focus()})
  ])));
 }else root.append(node("p",{class:"hint",text:"No SMB shares reported."}));
 if(nfs)root.append(node("p",{class:"hint",text:nfs.available===false?"NFS inventory unavailable on this installed NAS.":"NFS inventory was returned. Linux NFSv3 mounting requires an allowed server export and administrator-approved local mount target."}));
 return root;
}
function mountsPage(){
 const root=node("div",{},heading("Mapped drives","Recorded mount requests from this desktop companion."));
 if(!S.settings.mappings.length)root.append(empty("⌘","Nothing mounted yet","Use Shares & access to map a NAS share.",button("Connect a share",()=>{S.view="shares";render()},"btn btn-primary")));
 else root.append(section("Saved mappings",table(["Drive or folder","Remote share","Appliance","Action"],
  S.settings.mappings.map(m=>[
   primary(m.target||"Desktop volume",m.protocol.toUpperCase()),
   node("span",{class:"path",text:m.server+"/"+m.remote}),
   node("span",{text:getDevice(m.deviceId)?.name||"Unknown"}),
   button("Disconnect",()=>confirmUnmount(m),"btn btn-danger")
  ]))));
 root.append(node("p",{class:"hint",text:"Saved mappings describe OS requests. An entry does not prove that a volume is currently mounted. Use Finder, File Explorer, or your Linux file manager to verify."}));
 return root;
}
function settingsPage(){
 const choices=node("select",{id:"theme"},
  ...[["system","Follow operating system"],["light","Light"],["dark","Dark"]].map(([value,text])=>node("option",{value,text}))
 );
 choices.value=S.settings.theme;
 choices.addEventListener("change",()=>task(async()=>{await request("/api/theme",{theme:choices.value});await reload()}));
 return node("div",{},heading("Desktop settings","Preferences for this computer."),
  section("Appearance",node("div",{class:"detail-main"},
   node("div",{class:"settings-row"},node("div",{},node("strong",{text:"Color theme"}),
    node("div",{class:"hint",text:"Matches Zero1Local's original light and dark themes."})),choices)
  )),
  section("About",node("div",{class:"detail-main"},
   node("h2",{text:"Zero1Local Desktop"}),
   node("p",{class:"muted",text:"A Gigabyte Grove project. Native Go desktop companion for Windows, Linux, and macOS."}),
   node("p",{class:"hint",text:"Your NAS management sessions, passwords, API keys and MFA remain under the original Zero1Local and operating system security controls."})
  ))
 );
}
function dialog(title,body,onConfirm,buttonText="Continue"){
 document.querySelector("#modal-title").textContent=title;
 modalBody.replaceChildren(body);
 document.querySelector("#modal-submit").textContent=buttonText;
 modal.showModal();
 const form=document.querySelector("#modal-form");
 form.onsubmit=async e=>{
  if(e.submitter?.value!=="confirm")return;
  e.preventDefault();modal.close();await onConfirm();
 };
 body.querySelector("input")?.focus();
}
function addDialog(){
 const content=node("div",{},
  field("IP address, hostname or management URL","add-url","192.168.0.87"),
  node("p",{class:"hint",text:"Zero1Local is identified through its existing read-only health API before this address is saved. For secure management use HTTPS when configured."})
 );
 dialog("Add Zero1Local appliance",content,()=>task(async()=>{
  await request("/api/add",{url:content.querySelector("#add-url").value});
  await reload();flash("Appliance saved");
 }),"Add appliance");
}
function forget(d){
 dialog("Forget this appliance?",node("div",{},node("p",{text:"Forget "+d.name+" ("+d.url+") from this computer? No NAS files or settings will be changed."})),
 ()=>task(async()=>{await request("/api/remove",{id:d.id});await reload();flash("Appliance forgotten")}),"Forget appliance");
}
function confirmUnmount(m){
 dialog("Disconnect share?",node("div",{},node("p",{text:"Disconnect "+m.server+"/"+m.remote+" from "+(m.target||"this computer")+"? Any open files may be affected. This will not delete NAS data."})),
 ()=>task(async()=>{const result=await request("/api/unmount",{id:m.id});await reload();flash(result.message)}),"Disconnect");
}
async function discover(){await task(async()=>{
 flash("Scanning the current private LAN...");
 const result=await request("/api/discover",{});
 S.scan=result.devices||[];S.view="devices";
 flash("Scan complete: "+S.scan.length+" candidates found");
})}
async function openDevice(id){await task(async()=>{
 await request("/api/open",{id});
 flash("Opening the existing Zero1Local management console");
})}
async function connectShare(){
 const d=selected();if(!d)return;
 const req={deviceId:d.id,protocol:document.querySelector("#protocol").value,
  server:document.querySelector("#server").value,remote:document.querySelector("#remote").value,
  target:document.querySelector("#target").value};
 await task(async()=>{const v=await request("/api/mount",req);await reload();S.view="mounts";flash(v.message)});
}
async function loadInventory(){
 const token=document.querySelector("#api-token")?.value||"";
 await task(async()=>{S.inventory=await request("/api/shares",{id:S.selected,token});flash("Share inventory received")});
}
navigation.forEach(b=>b.addEventListener("click",()=>{S.view=b.dataset.nav;render();main.focus({preventScroll:true})}));
document.querySelector("#btn-add").addEventListener("click",addDialog);
document.querySelector("#btn-discover").addEventListener("click",discover);
reload().catch(e=>{main.replaceChildren(empty("!","Desktop service unavailable",e.message));flash(e.message)});
