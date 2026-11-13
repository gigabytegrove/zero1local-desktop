"use strict";
const csrf=document.querySelector('meta[name="csrf-token"]').content;
const main=document.querySelector("#main");
const rail=document.querySelector("#device-rail");
const modal=document.querySelector("#modal");
const modalBody=document.querySelector("#modal-body");
const navigation=[...document.querySelectorAll("[data-nav]")];
const S={settings:{theme:"system",devices:[],mappings:[]},platform:"linux",view:"home",selected:"",busy:false,scan:[],inventory:null,sessions:{},live:{},liveLoading:"",liveErrors:{},fileLocation:{id:"",share:"",rel:""},updater:{supported:false,version:"development preview",channel:"beta",checking:false}};

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
 S.settings=result.settings;S.platform=result.platform;S.updater=result.updater||S.updater;S.sessions=result.sessions||{};
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
 const headingRow=node("tr",{},...head.map(label=>node("th",{scope:"col",text:label})));
 const bodyRows=rows.map(row=>node("tr",{},...row.map(cell=>node("td",{},cell))));
 return node("table",{class:"list-table"},node("thead",{},headingRow),node("tbody",{},...bodyRows));
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
  node("span",{class:"connection-dot"+(S.sessions[d.id]?" connected":" disconnected"),"aria-hidden":"true"}),
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
 const titles={home:"Overview",devices:"My appliances",files:"File manager",shares:"Shared folders",mounts:"Mapped drives",storage:"Storage",tasks:"Tasks",users:"Users & groups",services:"Services",backups:"Backups",activity:"Notifications",protection:"Protection",settings:"Desktop settings"};
 document.querySelector("#top-title").textContent=titles[S.view];
 document.querySelector("#top-subtitle").textContent=selected()?.name||"Zero1Local on this computer";
 document.querySelector("#btn-discover").disabled=S.busy;
 document.querySelector("#btn-add").disabled=S.busy;
 switch(S.view){
 case "devices":main.replaceChildren(devicesPage());break;
 case "files":main.replaceChildren(fileManagerPage());break;
 case "storage":case "tasks":case "users":case "services":case "backups":case "activity":case "protection":main.replaceChildren(managementPage(S.view));break;
 case "shares":main.replaceChildren(sharesPage());break;
 case "mounts":main.replaceChildren(mountsPage());break;
 case "settings":main.replaceChildren(settingsPage());break;
 default:main.replaceChildren(homePage());
 }
}
function homePage(){
 const d=selected(),maps=S.settings.mappings;
 const root=node("div",{},heading("Your Zero1Local workspace","Manage multiple Zero1Local appliances, files, shares and health from one computer."),
 node("div",{class:"health-strip"},
  ...[[S.settings.devices.length,"Saved appliances"],[maps.length,"Saved mappings"],["Local","No cloud account"]].map(([value,name])=>
   node("div",{class:"health-item"},node("strong",{text:String(value)}),node("span",{text:name}))
 )));
 if(!d){
  root.append(empty("▣","Find your first Zero1Local","Discover local NAS devices or add an IP address or hostname.",button("Scan for Zero1Local",discover,"btn btn-primary")));
  return root;
 }
 root.append(connectionBanner(d));
 root.append(section("Selected appliance",node("div",{class:"detail-grid"},
  node("div",{class:"detail-main"},node("h2",{text:d.name}),
   node("dl",{class:"facts"},
    node("dt",{text:"Management address"}),node("dd",{class:"path",text:d.url}),
    node("dt",{text:"Zero1Local version"}),node("dd",{text:d.version||"Not reported"}),
    node("dt",{text:"Connection"}),node("dd",{},node("span",{class:"state-label warn",text:"Saved; not continuously monitored"}))
   ),
   node("div",{class:"form-actions"},button("Open complete NAS console",()=>openDevice(d.id),"btn btn-primary"),
    button("File manager",()=>switchView("files")),
    button("Connect shares",()=>{S.view="shares";render()}))
  ),
  node("div",{class:"detail-aside"},node("h2",{text:"Complete NAS management"}),
   node("p",{text:"The original Zero1Local console keeps its complete feature set, users, privileges, and MFA. Desktop share connections use the host operating system."}),
   node("p",{class:"hint",text:"Advanced configuration opens the original Zero1Local console in your system browser. That browser has its own separate NAS login session; desktop sessions cannot be shared across the security boundary."}))
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
   node("span",{class:"path",text:d.url}),button("Sign in & save",()=>addDialog(d.url),"btn btn-primary")])
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
  node("div",{class:"warning",text:"Drive mapping uses your operating system’s SMB/NFS credentials, separate from the Zero1Local administrator login. Your NAS password is never saved by the desktop app."}),
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
  node("h2",{text:"Verified shared folders"}),
  node("p",{class:"hint",text:"Read actual SMB shares and NFS exports using your authenticated NAS session. No separate API key is needed."}),
  S.sessions[d.id]?
   button("Refresh NAS share inventory",loadInventory,"btn btn-primary",S.busy):
   button("Sign in to view NAS shares",()=>credentialDialog(d.id),"btn btn-primary",S.busy)
 );
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
  section("Software updates",node("div",{class:"detail-main"},
   node("div",{class:"settings-row"},
    node("div",{},node("strong",{text:"Zero1Local Desktop " + (S.updater.version||"development preview")}),
     node("div",{class:"hint",text:"Channel: "+(S.updater.channel||"beta")+" · Source: GitHub Releases"})
    ),
    button(S.updater.checking?"Checking for updates…":"Check for updates",checkUpdates,
     "btn btn-primary",S.busy||S.updater.checking||!S.updater.supported)
   ),
   S.updater.reason?node("p",{class:"hint",text:S.updater.reason}):node("p",{class:"hint",text:"Checks for newer beta releases. Only explicitly selected updates are downloaded, verified against SHA256SUMS, and installed with restart confirmation. No automatic NAS changes."}),
   S.updater.lastError?node("p",{class:"warning",text:S.updater.lastError}):null,
   S.updater.packageDownload?node("p",{class:"hint",text:"This Linux installation is managed by Debian packages. Download a newer SHA-256-verified .deb and install it with your distribution’s package manager; no unattended root changes."}):null,
   S.downloadedPackage?node("p",{class:"warning",text:S.downloadedPackage}):null,
   node("div",{class:"form-actions"},
    S.updater.packageDownload?button("Download verified .deb",downloadDebianUpdate,
      "btn btn-primary",S.busy||S.updater.checking):null,
    button("View releases",openReleases))
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
  e.preventDefault();
  const submit=document.querySelector("#modal-submit");
  submit.disabled=true;
  try{await onConfirm();modal.close()}
  catch(err){
   const inline=modalBody.querySelector("#signin-error");
   if(inline)inline.textContent=err.message||"Unable to complete sign-in";
   else flash(err.message||"Unable to complete operation");
  }finally{submit.disabled=false}
 };
 body.querySelector("input")?.focus();
}
function addDialog(address=""){credentialDialog("",address)}
function credentialDialog(id="",address=""){
 const d=id?getDevice(id):null;
 const content=node("div",{class:"signin-form"},
  !d?field("NAS address or hostname","add-url","192.168.0.87",address):node("p",{class:"hint",text:"Signing in to "+d.name+" at "+d.url}),
  field("Administrator username","admin-username","root","root"),
  field("Administrator password","admin-password","", "", "password"),
  field("Two-factor authentication code (if enabled)","admin-totp","6-digit code", "", "text"),
  node("label",{class:"http-consent"},
   node("input",{type:"checkbox",id:"allow-http",value:"yes"}),
   node("span",{text:"I understand that connecting over HTTP sends my administrator password without encryption on the network. I want to proceed anyway."})
  ),
  node("p",{class:"hint",text:"HTTPS certificates are checked. Credentials are used once to obtain a NAS session; passwords are never saved to disk. You will sign in again after restarting the desktop app."}),
  node("p",{class:"signin-error",id:"signin-error","aria-live":"assertive"})
 );
 const username=content.querySelector("#admin-username"),password=content.querySelector("#admin-password");
 username.autocomplete="username";password.autocomplete="current-password";password.required=true;
 dialog(d?"Sign in to "+d.name:"Add Zero1Local appliance",content,async()=>{
  const url=d?d.url:content.querySelector("#add-url").value.trim();
  if(!url)throw new Error("Enter the NAS management address.");
  const payload={username:username.value.trim(),password:password.value,
   totp:content.querySelector("#admin-totp").value.trim(),
   allowHTTP:content.querySelector("#allow-http").checked};
  if(!payload.username||!payload.password)throw new Error("Enter both administrator username and password.");
  if(d)payload.id=d.id;else payload.url=url;
  try{
   const result=await request(d?"/api/nas/login":"/api/add",payload);
   password.value="";content.querySelector("#admin-totp").value="";
   await reload();
   S.selected=result.id;S.view="home";render();
   flash("Connected to "+(d?d.name:result.name)+" as "+result.username);
  }finally{password.value="";}
 },d?"Sign in":"Verify & add NAS");
}
function forget(d){
 dialog("Forget this appliance?",node("div",{},node("p",{text:"Forget "+d.name+" ("+d.url+") from this computer? No NAS files or settings will be changed."})),
 ()=>task(async()=>{await request("/api/remove",{id:d.id});await reload();flash("Appliance forgotten")}),"Forget appliance");
}
function confirmUnmount(m){
 dialog("Disconnect share?",node("div",{},node("p",{text:"Disconnect "+m.server+"/"+m.remote+" from "+(m.target||"this computer")+"? Any open files may be affected. This will not delete NAS data."})),
 ()=>task(async()=>{const result=await request("/api/unmount",{id:m.id});await reload();flash(result.message)}),"Disconnect");
}
async function checkUpdates(){
 await task(async()=>{
  await request("/api/updates/check",{});
  await reload();
  flash("The native update window is checking GitHub Releases.");
  pollUpdater();
 });
}
async function pollUpdater(){
 if(!S.updater.checking)return;
 await new Promise(resolve=>setTimeout(resolve,1500));
 try{
  const result=await request("/api/state");
  S.updater=result.updater||S.updater;
  if(S.view==="settings")render();
  if(S.updater.checking)pollUpdater();
 }catch(e){flash("Could not refresh update state: "+e.message)}
}
async function downloadDebianUpdate(){
 const body=node("div",{},
  node("p",{text:"Download the newer Zero1Local Desktop Debian package from the official GitHub Release, verify SHA-256, and save it to your Downloads folder?"}),
  node("p",{class:"hint",text:"This will not install anything or request administrative privileges. You choose when to install the verified package using your distribution's package manager."})
 );
 dialog("Download desktop update",body,()=>task(async()=>{
  const result=await request("/api/updates/download-deb",{});
  await reload();
  S.downloadedPackage=result.status==="downloaded"?
    "Verified "+result.version+" package saved: "+result.path:"No newer Debian package is published.";
  flash(result.message);
 }),"Download .deb");
}
async function openReleases(){
 await task(async()=>{
  await request("/api/updates/releases",{});
  flash("Opening Zero1Local Desktop release downloads");
 });
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
 const d=selected();if(!d)return;
 await task(async()=>{
  const [smb,nfs]=await Promise.all([
   request("/api/nas/read",{id:d.id,resource:"shares"}),
   request("/api/nas/read",{id:d.id,resource:"nfs"})
  ]);
  S.inventory={"/api/shares":smb.data,"/api/nfs/exports":nfs.data};
  flash("Verified SMB and NFS inventory loaded");
 });
}

// All NAS management data comes from a scoped, authenticated local API.
// Remote HTML and JavaScript are never loaded into this privileged webview.
const managedViews={
 storage:{title:"Storage & disks",description:"RAID health, mounted volumes, capacity and SMART information.",resource:"storage"},
 tasks:{title:"Tasks & jobs",description:"Background jobs and ongoing work reported by your NAS.",resource:"tasks"},
 users:{title:"Users & groups",description:"Administrator-visible NAS accounts and group memberships.",resource:"users"},
 services:{title:"Services",description:"Current service inventory and operation status.",resource:"services"},
 backups:{title:"Backups",description:"Existing backup jobs, snapshots and recovery coverage.",resource:"backups"},
 activity:{title:"Notifications",description:"Alerts, operations and recent NAS notifications.",resource:"notifications"},
 protection:{title:"Protection plans",description:"Existing retention, backup and protection plans.",resource:"protection"},
};
function sessionFor(d=selected()){return d?S.sessions[d.id]:null}
function switchView(view){
 if(!["home","devices","files","shares","mounts","storage","tasks","users","services","backups","activity","protection","settings"].includes(view))return;
 S.view=view;render();main.focus({preventScroll:true});
 if(view==="files"||managedViews[view])refreshLive(false);
}
function sessionState(d){
 const session=sessionFor(d);
 if(!session)return "Sign in required";
 return "Signed in as "+session.username+" · "+(session.role||"administrator");
}
function connectionBanner(d){
 const session=sessionFor(d);
 return node("div",{class:"connection-banner"},
  node("span",{class:"connection-dot "+(session?"connected":"disconnected"),"aria-hidden":"true"}),
  node("div",{class:"connection-description"},
   node("strong",{text:session?"Connected to "+d.name:"Administrator sign-in needed"}),
   node("span",{text:session?sessionState(d):"Sign in to access files, storage, tasks and user management on this appliance."})
  ),
  session?button("Sign out",()=>disconnectDevice(d.id),"btn btn-quiet",S.busy):
   button("Sign in",()=>credentialDialog(d.id),"btn btn-primary",S.busy),
  button("NAS console",()=>openDevice(d.id),"btn btn-quiet",S.busy)
 );
}
async function disconnectDevice(id){
 await task(async()=>{await request("/api/nas/logout",{id});await reload();flash("Disconnected the NAS administrator session")});
}
function liveKey(d,resource){return d.id+":"+resource+(resource==="files"?":"+S.fileLocation.share+":"+S.fileLocation.rel:"")}
async function refreshLive(force=true){
 const d=selected();if(!d||!sessionFor(d))return;
 const resource=S.view==="files"?"files":managedViews[S.view]?.resource;
 if(!resource)return;
 if(resource==="files"&&S.fileLocation.id!==d.id)S.fileLocation={id:d.id,share:"",rel:""};
 const key=liveKey(d,resource);
 if(!force&&S.live[key])return;
 const args={id:d.id,resource};
 if(resource==="files"){args.share=S.fileLocation.share;args.rel=S.fileLocation.rel}
 S.liveLoading=key;delete S.liveErrors[key];render();
 try{
  const result=await request("/api/nas/read",args);
  S.live[key]=result.data;
 }catch(e){
  S.liveErrors[key]=e.message||"The NAS could not provide this information.";
  if(e.message?.includes("session expired")){await reload()}
 }finally{
  if(S.liveLoading===key)S.liveLoading="";
  if(selected()?.id===d.id)render();
 }
}
function liveEmpty(d,message="Sign in to load management information"){
 return node("div",{},connectionBanner(d),empty("▣","NAS connection required",message,
  button("Sign in to NAS",()=>credentialDialog(d.id),"btn btn-primary")));
}
function liveToolbar(d,key,action=()=>refreshLive()){
 return node("div",{class:"live-toolbar"},
  node("div",{class:"hint",text:sessionState(d)}),
  button("Refresh",action,"btn btn-quiet",S.liveLoading===key||S.busy),
  button("Full NAS console",()=>openDevice(d.id),"btn btn-quiet",S.busy)
 );
}
function liveErrorNode(key){
 const error=S.liveErrors[key];
 return error?node("div",{class:"warning",role:"alert"},
  node("strong",{text:"Unable to load NAS data: "}),node("span",{text:error}),
  button("Retry",()=>refreshLive(),"btn btn-quiet")):null;
}
function dataValue(value){
 if(value===null||value===undefined||value==="")return "—";
 if(Array.isArray(value))return value.slice(0,5).map(dataValue).join(", ")+(value.length>5?" …":"");
 if(typeof value==="object"){
  const scalars=Object.entries(value).filter(([,v])=>v!==null&&typeof v!=="object");
  return scalars.slice(0,4).map(([k,v])=>humanize(k)+": "+String(v)).join(" · ")||"Details available in NAS console";
 }
 return String(value);
}
function humanize(key){return String(key).replace(/[_-]/g," ").replace(/\b\w/g,c=>c.toUpperCase())}
function rowsFromData(data,resource){
 const names={
  users:["users"],tasks:["tasks","jobs","active"],services:["services","units"],
  backups:["backups","jobs","schedules"],notifications:["notifications","items"],
  protection:["plans"],storage:["raid","block_devices","smart"]
 };
 const result=[];
 if(Array.isArray(data))return data;
 for(const k of names[resource]||[]){
  if(Array.isArray(data?.[k]))return data[k];
  if(data?.[k]&&typeof data[k]==="object"&&Array.isArray(data[k].items))return data[k].items;
 }
 for(const [k,value] of Object.entries(data||{})){
  if(Array.isArray(value)&&value.length){result.push(...value.slice(0,80).map(item=>({source:k,...(typeof item==="object"&&item?item:{value:item})})));break}
 }
 return result;
}
function recordsTable(items){
 if(!items.length)return node("p",{class:"hint",text:"No records reported by this NAS."});
 const first=items.find(x=>x&&typeof x==="object")||{};
 const preferred=["name","username","title","id","status","state","health","level","type","group","service_active","size","started_at","updated_at"];
 const keys=preferred.filter(k=>Object.hasOwn(first,k));
 const remaining=Object.keys(first).filter(k=>!keys.includes(k)&&!["password","token","secret","key","private_key"].some(secret=>k.toLowerCase().includes(secret)));
 keys.push(...remaining.slice(0,Math.max(0,5-keys.length)));
 return table(keys.slice(0,6).map(humanize),items.slice(0,100).map(x=>keys.slice(0,6).map(k=>
  node("span",{class:"cell-value",text:dataValue(x?.[k])}))));
}
function managementPage(view){
 const d=selected(),meta=managedViews[view],root=node("div",{},heading(meta.title,meta.description));
 if(!d){root.append(empty("▣","Select an appliance","Add or select a NAS to see this information.",button("My appliances",()=>switchView("devices"),"btn btn-primary")));return root}
 if(!sessionFor(d)){root.append(liveEmpty(d));return root}
 const key=liveKey(d,meta.resource),data=S.live[key];
 root.append(liveToolbar(d,key),liveErrorNode(key));
 if(S.liveLoading===key&&!data)root.append(node("p",{class:"loading",role:"status",text:"Loading verified NAS data…"}));
 if(!data&&!S.liveErrors[key])root.append(button("Load "+meta.title,()=>refreshLive(),"btn btn-primary"));
 if(data){
  if(view==="storage"){
   root.append(section("Capacity & RAID",node("div",{class:"metrics-grid"},
    ...["data_usage","root_usage","mdstat","smart_available"].map(k=>
     node("div",{class:"metric-item"},node("small",{text:humanize(k)}),node("strong",{text:dataValue(data?.[k])}))
    ))));
  }
  const items=rowsFromData(data,meta.resource);
  if(items.length)root.append(section("Reported inventory",recordsTable(items)));
  else if(view!=="storage")root.append(section("Reported status",node("div",{class:"metrics-grid"},
   ...Object.entries(data).slice(0,12).map(([k,v])=>node("div",{class:"metric-item"},
    node("small",{text:humanize(k)}),node("strong",{text:dataValue(v)}))))));
  root.append(node("p",{class:"hint",text:"These are live read-only values from Zero1Local. Create, edit, delete and other administrative operations remain available in the original secured NAS console."}));
 }
 return root;
}
function fileManagerPage(){
 const d=selected(),root=node("div",{},heading("File manager","Browse real NAS shared folders without exposing system directories or desktop credentials."));
 if(!d){root.append(empty("▣","No NAS selected","Add a Zero1Local appliance to browse its shared folders.",button("Add appliance",()=>addDialog(),"btn btn-primary")));return root}
 if(!sessionFor(d)){root.append(liveEmpty(d));return root}
 if(S.fileLocation.id!==d.id)S.fileLocation={id:d.id,share:"",rel:""};
 const key=liveKey(d,"files"),data=S.live[key],location=S.fileLocation;
 root.append(liveToolbar(d,key),liveErrorNode(key));
 if(location.share){
  const crumbs=[button("All shares",()=>openFileLocation(d.id,"",""),"btn btn-quiet"),
   node("span",{text:" / "+location.share})];
  if(location.rel){
   const parts=location.rel.split("/");
   crumbs.push(...parts.map((part,i)=>button(part,()=>openFileLocation(d.id,location.share,parts.slice(0,i+1).join("/")),"btn btn-quiet")));
  }
  root.append(node("div",{class:"file-crumbs"},...crumbs));
 }
 if(S.liveLoading===key&&!data)root.append(node("p",{class:"loading",role:"status",text:"Reading NAS shared folders…"}));
 if(!data&&!S.liveErrors[key])root.append(button("Browse shared folders",()=>refreshLive(),"btn btn-primary"));
 if(data?.scope==="shares"){
  const shares=Array.isArray(data.shares)?data.shares:[];
  root.append(section("Available shared folders",shares.length?
   table(["Shared folder","Availability","Action"],shares.map(sh=>[
    node("strong",{text:sh.name||"Shared folder"}),
    node("span",{text:sh.network_available===false?"Local-only":"Accessible"}),
    button("Open",()=>openFileLocation(d.id,sh.name,""),"btn btn-primary")
   ])):empty("▣","No user shares found","Only folders exposed by Zero1Local's share inventory are shown.")));
 }else if(data?.scope==="share"){
  const entries=Array.isArray(data.entries)?data.entries:[];
  root.append(section(location.share+" / "+(location.rel||""),
   entries.length?table(["Name","Size","Modified","Action"],entries.map(item=>[
    node("span",{class:"file-name",text:(item.directory?"📁 ":"📄 ")+item.name}),
    node("span",{text:item.directory?"Folder":formatBytes(item.size)}),
    node("span",{text:item.modified?new Date(item.modified).toLocaleString():"—"}),
    item.directory?button("Open",()=>openFileLocation(d.id,location.share,item.relative_path),"btn btn-quiet"):
     button("Download",()=>downloadNASFile(d,item),"btn btn-quiet")
   ])):empty("▣","This folder is empty","No files or directories were returned.")));
  if(location.rel)root.append(button("Back to parent",()=>openFileLocation(d.id,location.share,data.parent_relative_path||""),"btn btn-quiet"));
 }
 root.append(node("p",{class:"hint",text:"The desktop only displays share-rooted folders approved by Zero1Local. Upload, rename, move, trash and advanced file operations open in the full NAS console until their native operations are validated."}),
  button("Open NAS file manager",()=>openDevice(d.id),"btn btn-quiet"));
 return root;
}
function formatBytes(size){
 const v=Number(size)||0;
 if(v<1024)return v+" B";
 const units=["KB","MB","GB","TB"];
 let n=v/1024,index=0;
 while(n>=1024&&index<units.length-1){n/=1024;index++}
 return n.toFixed(n>=10?0:1)+" "+units[index];
}
function openFileLocation(id,share,rel){
 S.fileLocation={id,share,rel};S.view="files";render();refreshLive(false);
}
async function downloadNASFile(d,item){
 if(!item.path)return flash("The NAS did not supply a file path.");
 if(!window.URL?.createObjectURL)return flash("This runtime cannot download files directly; use the NAS file manager.");
 await task(async()=>{
  const response=await fetch("/api/nas/download",{
   method:"POST",headers:{"Content-Type":"application/json","X-Z1L-Token":csrf},
   body:JSON.stringify({id:d.id,path:item.path})
  });
  if(!response.ok){
   let message="Download failed (HTTP "+response.status+")";
   try{message=(await response.json()).error||message}catch{}
   throw new Error(message);
  }
  const blob=await response.blob();
  const blobURL=window.URL.createObjectURL(blob);
  const link=node("a",{href:blobURL,download:item.name});
  document.body.append(link);link.click();link.remove();
  window.setTimeout(()=>window.URL.revokeObjectURL(blobURL),30000);
  flash("Download started: "+item.name);
 });
}

navigation.forEach(b=>b.addEventListener("click",()=>switchView(b.dataset.nav)));
document.querySelector("#btn-add").addEventListener("click",addDialog);
document.querySelector("#btn-discover").addEventListener("click",discover);
reload().catch(e=>{
 main.replaceChildren(empty("!","Desktop service unavailable",e.message,
  button("Retry connection",()=>reload().catch(err=>flash(err.message||"Unable to reconnect")),"btn btn-primary",false)));
 flash(e.message);
});
