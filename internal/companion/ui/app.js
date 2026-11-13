"use strict";
const csrf=document.querySelector('meta[name="csrf-token"]').content;
const main=document.querySelector("#main");
const rail=document.querySelector("#device-rail");
const modal=document.querySelector("#modal");
const modalBody=document.querySelector("#modal-body");
const navigation=[...document.querySelectorAll("[data-nav]")];
const S={settings:{theme:"system",devices:[],mappings:[]},platform:"linux",view:"devices",selected:"",busy:false,scan:[],inventory:null,sessions:{},live:{},liveLoading:"",liveErrors:{},fileLocation:{id:"",share:"",rel:""},updater:{supported:false,version:"development preview",channel:"beta",checking:false}};

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
 S.sessionRecovery=result.sessionRecovery||{};S.credentialVaultAvailable=!!result.credentialVaultAvailable;
 if(S.selected&&!getDevice(S.selected)){S.selected="";S.view="devices"}
 theme();render();
}
const systemTheme=matchMedia("(prefers-color-scheme: dark)");
function theme(){
 document.documentElement.dataset.theme=S.settings.theme==="system"?(systemTheme.matches?"dark":"light"):S.settings.theme;
}
systemTheme.addEventListener("change",()=>{if(S.settings.theme==="system")theme()});
function choose(id){if(!getDevice(id))return;S.selected=id;S.inventory=null;S.fileLocation={id,share:"",rel:""};S.view="home";render();main.focus({preventScroll:true})}
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
 if(S.view!=="devices"&&S.view!=="settings"&&!selected())S.view="devices";
 const scoped=S.view!=="devices"&&S.view!=="settings";
 const deviceNav=document.querySelector("#device-navigation");if(deviceNav)deviceNav.hidden=!scoped;
 const switcherPanel=document.querySelector("#workspace-switcher");if(switcherPanel)switcherPanel.hidden=!scoped;
 const deviceTitle=document.querySelector("#device-nav-title");if(deviceTitle)deviceTitle.textContent=selected()?.name||"NAS workspace";
 navigation.forEach(b=>{
  const active=b.dataset.nav===S.view;b.classList.toggle("active",active);
  if(active)b.setAttribute("aria-current","page");else b.removeAttribute("aria-current");
 });
 const titles={home:"Overview",devices:"My appliances",files:"File manager",shares:"Shared folders",mounts:"Mapped drives",system:"System",storage:"Storage",network:"Network",docker:"Docker",hardware:"Hardware",tasks:"Tasks",users:"Users & groups",services:"Services",backups:"Backups",activity:"Notifications",protection:"Protection",settings:"Desktop settings"};
 document.querySelector("#top-title").textContent=titles[S.view];
 document.querySelector("#top-subtitle").textContent=scoped?(selected()?.name||"NAS workspace"):"Manage your Zero1Local NAS devices";
 document.querySelector("#btn-discover").disabled=S.busy;
 const manage=document.querySelector("#nav-management");
 if(manage&&["system","storage","network","docker","hardware","tasks","users","services","backups","protection","activity"].includes(S.view))manage.open=true;
 const switcher=document.querySelector("#appliance-switcher");
 if(switcher){
  switcher.replaceChildren(...(S.settings.devices.length?S.settings.devices.map(d=>node("option",{value:d.id,text:d.name})):
   [node("option",{value:"",text:"No appliance connected"})]));
  switcher.value=S.selected||"";
  switcher.disabled=S.busy||S.settings.devices.length===0;
 }
 const presence=document.querySelector("#topbar-presence");
 if(presence){const d=selected(),session=d&&S.sessions[d.id];presence.textContent=!d?"No NAS selected":session?"● Signed in":S.sessionRecovery?.[d.id]?"◌ Restoring sign-in":"○ Sign-in required";presence.className="topbar-presence "+(session?"online":"offline")}
 document.querySelector("#btn-discover").hidden=S.view!=="devices";
 document.querySelector("#btn-add").disabled=S.busy;
 switch(S.view){
 case "devices":main.replaceChildren(devicesPage());break;
 case "files":main.replaceChildren(fileManagerPage());break;
 case "system":case "storage":case "network":case "docker":case "hardware":case "tasks":case "users":case "services":case "backups":case "activity":case "protection":main.replaceChildren(managementPage(S.view));break;
 case "shares":main.replaceChildren(sharesPage());break;
 case "mounts":main.replaceChildren(mountsPage());break;
 case "settings":main.replaceChildren(settingsPage());break;
 default:main.replaceChildren(homePage());
 }
}
function homePage(){
 const d=selected(),maps=S.settings.mappings;
 const root=node("div",{class:"overview-shell"});
 if(!d){
  root.append(heading("Your NAS workspace","Keep your Zero1Local appliances, files and shared drives together."),
   empty("▣","No appliance connected","Find a Zero1Local NAS on your network, or add one directly by its management address.",
    node("div",{class:"empty-actions"},button("Find appliances",discover,"btn btn-primary"),
     button("Add by address",()=>addDialog(),"btn btn-quiet"))));
  return root;
 }
 const session=S.sessions[d.id];
 const dMaps=maps.filter(m=>m.deviceId===d.id);
 const overview=node("header",{class:"nas-overview"},
  node("div",{class:"nas-overview-main"},
   node("div",{class:"nas-overview-meta"},node("span",{class:"nas-state "+(session?"online":"offline"),
    text:session?"● Administrator connected":"○ Sign-in required"}),
    node("span",{class:"nas-version",text:"Zero1Local "+(d.version||"version unknown")})),
   node("h1",{text:d.name}),
   node("p",{class:"nas-address",text:d.url}),
   node("p",{class:"nas-overview-caption",text:session?
     "Signed in as "+session.username+". NAS permissions are enforced by Zero1Local.":
     "The appliance is saved on this computer, but no active administrator session is available."})
  ),
  node("div",{class:"nas-overview-actions"},
   session?button("Open files",()=>switchView("files"),"btn btn-primary"):
    button("Sign in to NAS",()=>credentialDialog(d.id),"btn btn-primary"),
   button("Open NAS console",()=>openDevice(d.id),"btn btn-quiet")
  )
 );
 root.append(overview);
 root.append(node("div",{class:"workspace-stats"},
  ...[["v"+(d.version||"—"),"NAS software"],[String(dMaps.length),"This NAS: mapped drives"],
    [session?"Connected":"Sign-in needed","Administrator session"]].map(([value,label])=>
    node("div",{class:"workspace-stat"},node("strong",{text:value}),node("span",{text:label}))
  )
 ));
 root.append(node("div",{class:"workspace-columns"},
  node("section",{class:"workspace-section"},
   node("h2",{text:"Work with "+d.name}),
   node("p",{class:"muted",text:"Common tasks stay in the desktop. Use the full console for advanced administration."}),
   node("div",{class:"workspace-task-list"},
    ...[
     ["▦","Browse files","Open shared folders and manage files",()=>switchView("files")],
     ["▤","Connect shared folders","Review SMB/NFS exports and map a drive",()=>switchView("shares")],
     ["◫","Check storage","RAID, disk usage and SMART information",()=>switchView("storage")],
     ["≡","View NAS tasks","Background work, jobs and notifications",()=>switchView("tasks")]
    ].map(([symbol,title,desc,handler])=>
     node("button",{class:"workspace-task",type:"button",onClick:handler},
      node("span",{class:"workspace-task-symbol",text:symbol,"aria-hidden":"true"}),
      node("span",{class:"workspace-task-copy"},node("strong",{text:title}),node("small",{text:desc})),
      node("span",{class:"workspace-task-chevron",text:"›","aria-hidden":"true"})
     ))
   )
  ),
  node("section",{class:"workspace-section"},
   node("div",{class:"workspace-section-header"},
    node("h2",{text:"This computer"}),
    button("Manage mappings",()=>switchView("mounts"),"btn btn-quiet")),
   node("dl",{class:"quick-facts"},
    node("dt",{text:"NAS address"}),node("dd",{class:"path",text:new URL(d.url).host}),
    node("dt",{text:"Active session"}),node("dd",{text:session?session.username+" · "+(session.role||"administrator"):"Not signed in"}),
    node("dt",{text:"Saved mappings"}),node("dd",{text:String(dMaps.length)}),
    node("dt",{text:"Security"}),node("dd",{text:d.url.startsWith("https://")?"HTTPS":"HTTP · unencrypted"})
   ),
   dMaps.length?node("div",{class:"home-mappings"},
    ...dMaps.slice(0,5).map(m=>node("div",{class:"home-mapping"},
     node("strong",{text:m.target||m.protocol.toUpperCase()}),
     node("span",{class:"path",text:m.server+"/"+m.remote})
    ))):node("div",{class:"quiet-empty",text:"No share mappings saved for this appliance."}),
   session?button("Open system overview",()=>switchView("system"),"btn btn-quiet"):
    button("Authenticate appliance",()=>credentialDialog(d.id),"btn btn-quiet")
  )
 ));
 return root;
}
function devicesPage(){
 const root=node("div",{class:"appliances-page"},heading("My appliances",
  "Select, sign in to or manage the Zero1Local installations registered on this computer."));
 root.append(node("div",{class:"appliance-directory-toolbar"},
  node("span",{class:"muted",text:S.settings.devices.length+" saved appliance"+(S.settings.devices.length===1?"":"s")}),
  node("div",{class:"actions"},button("Scan this network",discover,"btn btn-quiet"),
   button("Add appliance",()=>addDialog(),"btn btn-primary"))
 ));
 if(S.settings.devices.length){
  root.append(node("div",{class:"appliance-directory"},
   ...S.settings.devices.map(d=>{
    const session=S.sessions[d.id];
    return node("section",{class:"appliance-directory-row"},
     node("div",{class:"appliance-directory-identity"},
      node("span",{class:"connection-dot "+(session?"connected":"disconnected"),"aria-hidden":"true"}),
      node("div",{},node("h2",{text:d.name}),node("p",{class:"path",text:d.url}),
       node("small",{text:"Zero1Local "+(d.version||"version unknown")}))
     ),
     node("div",{class:"appliance-directory-status"},
      node("strong",{class:session?"online":"offline",text:session?"Signed in":S.sessionRecovery?.[d.id]?"Restoring sign-in":"Sign-in required"}),
      node("small",{text:session?session.username+" · "+(session.role||"administrator"):
       S.sessionRecovery?.[d.id]|| (d.rememberSignIn?"Saved sign-in enabled":"Sign in when you open this device")}),
      node("small",{text:d.rememberSignIn?"Keeps me signed in on this computer":"Session-only sign-in"})
     ),
     node("div",{class:"appliance-directory-actions"},
      button("Open workspace",()=>choose(d.id),"btn btn-primary"),
      session?button(d.rememberSignIn?"Sign out & forget":"Sign out",()=>disconnectDevice(d.id)):button("Sign in",()=>credentialDialog(d.id)),
      session?d.rememberSignIn?button("Stop remembering",()=>forgetSavedNAS(d.id)):button("Keep signed in",()=>credentialDialog(d.id)):null,
      button("NAS console",()=>openDevice(d.id)),
      button("Forget",()=>forget(d),"btn btn-danger")
     )
    );
   })
  ));
 }else root.append(empty("▣","No appliances registered",
  "Discover a NAS on your local network or enter its management address to sign in.",
  button("Add appliance",()=>addDialog(),"btn btn-primary")));
 if(S.scan.length){
  root.append(section("Discovered devices",table(["Appliance","Version","Address","Action"],
   S.scan.map(d=>[primary(d.name,d.url),node("span",{text:d.version}),
    node("span",{class:"path",text:d.url}),button("Sign in & save",()=>addDialog(d.url),"btn btn-primary")]))));
 }
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
  S.platform==="windows"?node("label",{class:"share-reconnect-option"},
   node("input",{id:"share-reconnect",type:"checkbox"}),
   node("span",{},node("strong",{text:"Reconnect this SMB drive when I sign in to Windows"}),
    node("small",{text:"Windows manages this persistent mapping and any share credentials in Windows Credential Manager. It does not use the NAS administrator password."}))
  ):node("p",{class:"hint",text:"On this operating system, share authentication and reconnect behavior are managed by your file manager. The companion will not silently store SMB or NFS passwords."}),
  node("div",{class:"form-actions"},button("Connect share",connectShare,"btn btn-primary"))
 );
 const p=form.querySelector("#protocol"),t=form.querySelector("#target");
 function updateMountTarget(){
  const nfs=p.value==="nfs";
  t.disabled=S.platform==="darwin"||(S.platform==="linux"&&!nfs);
  t.placeholder=nfs?"/home/you/nfs-share":S.platform==="windows"?"Z:":"Managed by OS";
 }
 p.addEventListener("change",updateMountTarget);updateMountTarget();
 // Verified export discovery is presented before advanced manual mapping.
 const inventory=node("div",{class:"detail-main"},
  node("h2",{text:"Verified folders on "+d.name}),
  node("p",{class:"hint",text:"Choose a verified SMB share or NFS export below to prefill a connection. The NAS enforces share access."}),
  S.sessions[d.id]?
   button("Refresh NAS share inventory",loadInventory,"btn btn-primary",S.busy):
   button("Sign in to view NAS shares",()=>credentialDialog(d.id),"btn btn-primary",S.busy)
 );
 root.append(section("Available shared folders",inventory));
 if(S.inventory)root.append(inventoryPage());
 root.append(node("details",{class:"mapping-details"},
  node("summary",{text:"Connect an SMB share or NFS export"}),
  node("div",{class:"mapping-details-body"},form)));
 return root;
}
function inventoryPage(){
 const root=node("div",{}),smb=S.inventory["/api/shares"],nfs=S.inventory["/api/nfs/exports"];
 // The NAS returns one logical shared folder per backing directory. A
 // directory exported over both SMB and NFS must not appear as two shares.
 const folders=Array.isArray(smb?.folders)?smb.folders:[];
 const legacy=Array.isArray(smb?.active)?smb.active:[];
 if(folders.length){
  root.append(table(["Shared folder","Protocols","Backing folder","Connect"],folders.map(folder=>[
   primary(String(folder.name||"Unnamed"),Array.isArray(folder.aliases)&&folder.aliases.length?"Also known as: "+folder.aliases.join(", "):"Unified shared folder"),
   node("span",{text:[folder.smb?"SMB":"",folder.nfs?"NFSv3":""].filter(Boolean).join(" + ")}),
   node("span",{class:"path",text:String(folder.path||"")}),
   node("div",{class:"actions"},
    folder.smb?button("Use SMB",()=>selectNASShare("smb",folder.smb_name||folder.name)):null,
    folder.nfs&&S.platform==="linux"?button("Use NFS",()=>selectNASShare("nfs",folder.nfs_details?.path||folder.path)):null,
    folder.name_conflict?node("span",{class:"warning",text:"Name conflict"}):null
   )
  ])));
 }else if(legacy.length){
  root.append(table(["SMB share","Folder","Action"],legacy.map(x=>[
   primary(String(x.name||"Unnamed"),x.browseable?"Browseable":"Restricted"),
   node("span",{class:"path",text:String(x.path||"")}),
   button("Use SMB",()=>selectNASShare("smb",x.name||""))
  ])));
 }else root.append(node("p",{class:"hint",text:"No shared folders were reported by this NAS."}));
 if(nfs&&Array.isArray(nfs.exports)&&!folders.length){
  root.append(section("NFSv3 exports",table(["Export","Path","Action"],nfs.exports.map(ex=>[
   node("strong",{text:ex.share_name||String(ex.id)}),
   node("span",{class:"path",text:ex.path||""}),
   S.platform==="linux"?button("Use NFS",()=>selectNASShare("nfs",ex.path||"")):
    node("span",{class:"hint",text:"Linux client required"})
  ]))));
 }
 return root;
}
function selectNASShare(protocol,remote){
 const p=document.querySelector("#protocol"),r=document.querySelector("#remote");
 const details=document.querySelector(".mapping-details");if(details)details.open=true;
 if(!p||!r)return;
 p.value=protocol;
 p.dispatchEvent(new Event("change",{bubbles:true}));
 r.value=remote;
 r.focus();
 flash("Selected "+protocol.toUpperCase()+" share: "+remote);
}
function mountsPage(){
 const d=selected();
 if(!d)return node("div",{},heading("Mapped drives","Choose a NAS first."),
  button("All NAS devices",()=>switchView("devices"),"btn btn-primary"));
 const mappings=S.settings.mappings.filter(m=>m.deviceId===d.id);
 const root=node("div",{class:"device-mappings"},heading(d.name+" · Mapped drives",
  "Drive mappings associated only with this NAS. Your operating system handles share authentication."));
 if(!mappings.length)root.append(empty("⌘","No drives mapped from "+d.name,
  "Open Shared folders to connect an SMB or NFS export from this NAS.",
  button("Connect a share",()=>switchView("shares"),"btn btn-primary")));
 else root.append(section("Saved mappings from "+d.name,table(["Drive or folder","Share","Reconnect","Action"],
  mappings.map(m=>[
   primary(m.target||"Desktop volume",m.protocol.toUpperCase()),
   node("span",{class:"path",text:m.server+"/"+m.remote}),
   node("span",{text:m.reconnect?"Reconnect at Windows sign-in":"This session only"}),
   button("Disconnect",()=>confirmUnmount(m),"btn btn-danger")
  ]))));
 root.append(node("p",{class:"hint",text:"Mapping records do not prove a drive is currently available. Use File Explorer, Finder or your Linux file manager to confirm the connection. Share credentials are separate from NAS administrator sign-in."}));
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
function addDialog(address=""){credentialDialog("",typeof address==="string"?address:"")}
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
  node("label",{class:"remember-consent"},
   node("input",{type:"checkbox",id:"remember-nas",disabled:!S.credentialVaultAvailable}),
   node("span",{},node("strong",{text:"Keep me signed in to this NAS"}),
    node("small",{text:S.credentialVaultAvailable?
     "Save this NAS username and password in your computer’s secure credential vault for automatic sign-in and session renewal. Two-factor verification may still be required.":
     "Secure credential storage isn't available in this runtime. Sign-in will last for this app session only."}))
  ),
  node("p",{class:"hint",text:"Each NAS has its own sign-in. NAS passwords are never written to Zero1Local Desktop settings. Windows SMB share credentials are managed separately by Windows."}),
  node("p",{class:"signin-error",id:"signin-error","aria-live":"assertive"})
 );
 const username=content.querySelector("#admin-username"),password=content.querySelector("#admin-password");
 content.querySelector("#remember-nas").checked=!!d?.rememberSignIn;
 username.autocomplete="username";password.autocomplete="current-password";password.required=true;
 dialog(d?"Sign in to "+d.name:"Add Zero1Local appliance",content,async()=>{
  const url=d?d.url:content.querySelector("#add-url").value.trim();
  if(!url)throw new Error("Enter the NAS management address.");
  const payload={username:username.value.trim(),password:password.value,
   totp:content.querySelector("#admin-totp").value.trim(),
   allowHTTP:content.querySelector("#allow-http").checked,
   remember:content.querySelector("#remember-nas").checked};
  if(!payload.username||!payload.password)throw new Error("Enter both administrator username and password.");
  if(d)payload.id=d.id;else payload.url=url;
  try{
   const result=await request(d?"/api/nas/login":"/api/add",payload);
   password.value="";content.querySelector("#admin-totp").value="";
   await reload();
   S.selected=result.id;S.inventory=null;S.fileLocation={id:result.id,share:"",rel:""};S.view="home";render();
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
  target:document.querySelector("#target").value,
  reconnect:!!document.querySelector("#share-reconnect")?.checked};
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
 system:{title:"System overview",description:"Device identity, uptime, memory, operating system and readiness.",resource:"system"},
 network:{title:"Network",description:"Current addresses, interfaces, VLANs and routing information.",resource:"network"},
 docker:{title:"Containers & Docker",description:"Containers, image inventory and Docker runtime status.",resource:"docker"},
 hardware:{title:"Hardware",description:"Device hardware, fan, temperatures and system monitoring.",resource:"hardware"},
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
 if(!["home","devices","files","shares","mounts","system","storage","network","docker","hardware","tasks","users","services","backups","activity","protection","settings"].includes(view))return;
 if(!["devices","settings"].includes(view)&&!selected()){S.view="devices";render();return}
 S.view=view;render();main.focus({preventScroll:true});
 if(view==="files"||managedViews[view])refreshLive(false);
 if(view==="shares"&&sessionFor(selected())&&!S.inventory)loadInventory();
}
function sessionState(d){
 const session=sessionFor(d);
 if(!session)return S.sessionRecovery?.[d.id]||"Sign in required";
 return "Signed in as "+session.username+" · "+(session.role||"administrator");
}
function connectionBanner(d){
 const session=sessionFor(d);
 return node("div",{class:"connection-banner"},
  node("span",{class:"connection-dot "+(session?"connected":"disconnected"),"aria-hidden":"true"}),
  node("div",{class:"connection-description"},
   node("strong",{text:session?"Connected to "+d.name:S.sessionRecovery?.[d.id]||"Administrator sign-in needed"}),
   node("span",{text:session?sessionState(d):d.rememberSignIn?"Saved sign-in enabled for this NAS. Reconnect if verification is required.":"Sign in to access files, storage, tasks and user management on this appliance."})
  ),
  session?button(d.rememberSignIn?"Sign out & forget":"Sign out",()=>disconnectDevice(d.id),"btn btn-quiet",S.busy):
   button("Sign in",()=>credentialDialog(d.id),"btn btn-primary",S.busy),
  session?d.rememberSignIn?button("Stop remembering",()=>forgetSavedNAS(d.id),"btn btn-quiet"):
   button("Keep me signed in",()=>credentialDialog(d.id),"btn btn-quiet"):null,
  button("NAS console",()=>openDevice(d.id),"btn btn-quiet",S.busy)
 );
}
async function disconnectDevice(id){
 await task(async()=>{await request("/api/nas/logout",{id});await reload();flash("Signed out of the NAS. Saved sign-in was also cleared.")});
}
async function forgetSavedNAS(id){
 await task(async()=>{await request("/api/nas/forget-credential",{id});await reload();flash("Saved NAS sign-in disabled. Your active session stays connected.")});
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
  docker:["containers","images","stacks"],network:["interfaces","vlans"],hardware:["devices","disks","fans"],
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
function infoLine(label,value){
 return node("div",{class:"info-line"},node("dt",{text:label}),node("dd",{text:dataValue(value)}));
}
function formatPercent(num){
 const v=Number(num);if(!Number.isFinite(v))return null;
 return Math.max(0,Math.min(100,v));
}
function storageGauge(label,usage){
 if(!usage||typeof usage!=="object")return null;
 const percent=formatPercent(usage.percent??usage.used_percent??usage.use_percent);
 return node("div",{class:"capacity-line"},
  node("div",{class:"capacity-top"},node("strong",{text:label}),
   node("span",{text:percent===null?"Capacity unknown":percent.toFixed(1)+"% used"})),
  node("div",{class:"capacity-bar",role:"meter","aria-label":label,
   "aria-valuenow":percent===null?0:percent,"aria-valuemin":"0","aria-valuemax":"100"},
   node("span",{style:"width:"+(percent===null?0:percent)+"%"})),
  node("div",{class:"capacity-meta"},
   node("span",{text:"Used "+formatBytes(usage.used||0)}),
   node("span",{text:"Available "+formatBytes(usage.available??usage.avail??0)}),
   node("span",{text:usage.fstype||""})
  )
 );
}
function managementOverview(view,data){
 if(view==="storage"){
  const usages=[storageGauge("Main storage · /disk0",data.data_usage),storageGauge("System volume · /",data.root_usage)].filter(Boolean);
  const status=node("div",{class:"system-summary"},
   node("h2",{text:"Storage capacity"}),
   usages.length?node("div",{class:"capacity-stack"},...usages):node("p",{class:"hint",text:"The NAS has not reported usable capacity details."})
  );
  const raid=Array.isArray(data.raid)?data.raid:[];
  const root=node("div",{class:"data-blocks"},status);
  root.append(section("RAID arrays",raid.length?recordsTable(raid):
   node("p",{class:"hint",text:"No RAID arrays were reported."})));
  if(typeof data.mdstat==="string"&&data.mdstat.trim()){
   const details=node("details",{class:"raw-details"},node("summary",{text:"View raw RAID diagnostics"}),
    node("pre",{text:data.mdstat}));
   root.append(details);
  }
  const smart=Array.isArray(data.smart)?data.smart:[];
  if(smart.length)root.append(section("Disk health and SMART",recordsTable(smart)));
  return root;
 }
 if(view==="system"){
  const os=data.os&&typeof data.os==="object"?data.os:{};
  const facts=[
   ["Hostname",data.hostname],["Device model",data.model],
   ["Operating system",os.name??data.os],["OS kernel",data.kernel],
   ["Architecture",data.arch],["Uptime",data.uptime]
  ];
  return node("div",{class:"data-blocks"},
   section("Device identity",node("dl",{class:"info-list"},
    ...facts.filter(([,v])=>v!==undefined&&v!==null&&v!=="").map(([k,v])=>infoLine(k,v)))),
   data.memory?section("Memory",node("dl",{class:"info-list"},...Object.entries(data.memory).slice(0,9).map(([k,v])=>infoLine(humanize(k),v)))):null
  );
 }
 return null;
}
function managementPage(view){
 const d=selected(),meta=managedViews[view],root=node("div",{class:"management-view"},heading(meta.title,meta.description));
 if(!d){root.append(empty("▣","Select an appliance","Choose a NAS to view management information.",button("My appliances",()=>switchView("devices"),"btn btn-primary")));return root}
 if(!sessionFor(d)){root.append(liveEmpty(d));return root}
 const key=liveKey(d,meta.resource),data=S.live[key];
 root.append(liveToolbar(d,key));
 const warning=liveErrorNode(key);if(warning)root.append(warning);
 if(S.liveLoading===key&&!data)root.append(node("div",{class:"loading-state",role:"status"},
  node("span",{class:"loading-dot","aria-hidden":"true"}),node("span",{text:"Retrieving "+meta.title.toLowerCase()+" from "+d.name+"…"})));
 if(!data&&!S.liveErrors[key]&&S.liveLoading!==key)root.append(node("div",{class:"start-state"},
  node("strong",{text:"Ready to read "+meta.title.toLowerCase()}),
  node("p",{text:"This section only reads data that "+d.name+" permits your account to access."}),
  button("Load "+meta.title,()=>refreshLive(),"btn btn-primary")));
 if(data){
  const specialized=managementOverview(view,data);
  if(specialized)root.append(specialized);
  else{
   const items=rowsFromData(data,meta.resource);
   if(items.length)root.append(section("NAS inventory",recordsTable(items)));
   const summary=Object.entries(data).filter(([,v])=>v!==null&&v!==undefined&&!Array.isArray(v)&&(typeof v==="string"?v.length<180:typeof v!=="object")).slice(0,12);
   if(summary.length)root.append(section("Details",node("dl",{class:"info-list"},
    ...summary.map(([k,v])=>infoLine(humanize(k),v)))));
   if(!items.length&&!summary.length)root.append(node("div",{class:"quiet-empty",text:"No additional data was reported by the NAS."}));
  }
  root.append(node("p",{class:"hint management-footer",text:"Live, read-only NAS data. Administrative changes remain in the original secure NAS console."}));
 }
 return root;
}
function fileManagerPage(){
 const d=selected(),root=node("div",{},heading("File manager","Browse real NAS shared folders without exposing system directories or desktop credentials."));
 if(!d){root.append(empty("▣","No NAS selected","Add a Zero1Local appliance to browse its shared folders.",button("Add appliance",()=>addDialog(),"btn btn-primary")));return root}
 if(!sessionFor(d)){root.append(liveEmpty(d));return root}
 if(S.fileLocation.id!==d.id)S.fileLocation={id:d.id,share:"",rel:""};
 const key=liveKey(d,"files"),data=S.live[key],location=S.fileLocation;
 root.append(liveToolbar(d,key));
 const fileError=liveErrorNode(key);if(fileError)root.append(fileError);
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
 if(!data&&!S.liveErrors[key]&&S.liveLoading!==key)root.append(node("div",{class:"start-state"},node("strong",{text:"Browse your NAS shared folders"}),node("p",{text:"Only folders this NAS allows you to access will appear."}),button("Load shared folders",()=>refreshLive(),"btn btn-primary")));
 if(data?.scope==="shares"){
  const shares=Array.isArray(data.shares)?data.shares:[];
  root.append(section("Shared folders",shares.length?
   node("div",{class:"share-location-list"},...shares.map(sh=>
    node("button",{type:"button",class:"share-location",onClick:()=>openFileLocation(d.id,sh.name,"")},
     node("span",{class:"share-location-icon",text:"▦","aria-hidden":"true"}),
     node("span",{class:"share-location-copy"},node("strong",{text:sh.name||"Shared folder"}),
      node("small",{text:sh.network_available===false?"Local-only NAS folder":"Zero1Local shared folder"})),
     node("span",{class:"share-location-arrow",text:"Open ›","aria-hidden":"true"})
    )
   )):empty("▣","No user shares found","Only folders exposed by Zero1Local's share inventory are shown.")));
 }else if(data?.scope==="share"){
  const entries=Array.isArray(data.entries)?data.entries:[];
  root.append(node("div",{class:"file-actions"},
   button("New folder",()=>newFolderDialog(d,data.path),"btn btn-primary",S.busy||!data.writable),
   button("Upload files",()=>chooseUpload(d,data.path),"btn btn-quiet",S.busy||!data.writable)
  ));
  root.append(section(location.share+" / "+(location.rel||""),
   entries.length?table(["Name","Size","Modified","Actions"],entries.map(item=>[
    node("span",{class:"file-name",text:(item.directory?"📁 ":"📄 ")+item.name}),
    node("span",{text:item.directory?"Folder":formatBytes(item.size)}),
    node("span",{text:item.modified?new Date(item.modified).toLocaleString():"—"}),
    node("div",{class:"actions"},
     item.symlink?node("span",{class:"hint",text:"Link · use NAS console"}):
      item.directory?button("Open",()=>openFileLocation(d.id,location.share,item.relative_path),"btn btn-quiet"):
       button("Download",()=>downloadNASFile(d,item),"btn btn-quiet"),
     button("Rename",()=>renameNASItem(d,item),"btn btn-quiet",S.busy||!data.writable),
     button("Recycle",()=>recycleNASItem(d,item),"btn btn-danger",S.busy||!data.writable)
    )
   ])):empty("▣","This folder is empty","No files or directories were returned.")));
  if(location.rel)root.append(button("Back to parent",()=>openFileLocation(d.id,location.share,data.parent_relative_path||""),"btn btn-quiet"));
 }
 root.append(node("p",{class:"hint file-manager-footnote",text:"Zero1Local enforces file permissions. Bulk moves, sharing links, previews and permanent deletion remain in the full NAS console."}),
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

async function performFileAction(d,action,path,name=""){
 const result=await request("/api/nas/file-action",{id:d.id,action,path,name});
 if(result?.ok===false)throw new Error(result.error||"The NAS rejected this operation");
 const key=liveKey(d,"files");
 delete S.live[key];
 await refreshLive(true);
 return result;
}
function newFolderDialog(d,parentPath){
 const content=node("div",{},field("Folder name","new-folder-name","New folder"));
 dialog("Create a folder",content,async()=>{
  const name=content.querySelector("#new-folder-name").value.trim();
  if(!name)throw new Error("Enter a folder name");
  await performFileAction(d,"mkdir",parentPath,name);
  flash("Folder created: "+name);
 },"Create folder");
}
function renameNASItem(d,item){
 const content=node("div",{},field("New name","rename-item",item.name,item.name));
 dialog("Rename "+item.name,content,async()=>{
  const name=content.querySelector("#rename-item").value.trim();
  if(!name||name===item.name)throw new Error("Enter a different name");
  await performFileAction(d,"rename",item.path,name);
  flash("Renamed to "+name);
 },"Rename");
}
function recycleNASItem(d,item){
 const content=node("div",{},
  node("p",{text:"Move “"+item.name+"” to the Zero1Local Recycle Bin?"}),
  node("p",{class:"hint",text:"This changes files on the NAS and may affect anyone using this shared folder. It does not permanently delete the item."})
 );
 dialog("Move to Recycle Bin",content,async()=>{
  await performFileAction(d,"trash",item.path);
  flash("Moved to NAS Recycle Bin: "+item.name);
 },"Move to Recycle Bin");
}
function chooseUpload(d,path){
 const picker=node("input",{type:"file",multiple:"multiple","aria-label":"Select files to upload"});
 picker.addEventListener("change",async()=>{
  const files=[...picker.files||[]];
  if(!files.length)return;
  if(files.some(f=>f.size>256*1024*1024))return flash("Each upload must be smaller than 256 MiB.");
  await task(async()=>{
   for(const file of files){
    const form=new FormData();form.append("file",file,file.name);
    const res=await fetch("/api/nas/upload?id="+encodeURIComponent(d.id)+"&path="+encodeURIComponent(path),
     {method:"POST",headers:{"X-Z1L-Token":csrf},body:form});
    let body;
    try{body=await res.json()}catch{throw new Error("NAS upload response was invalid")}
    if(!res.ok)throw new Error(body.error||"Upload rejected by the NAS");
   }
   delete S.live[liveKey(d,"files")];
   await refreshLive(true);
   flash("Uploaded "+files.length+" file"+(files.length===1?"":"s")+" to the NAS");
  });
 });
 picker.click();
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
document.querySelector("#btn-add").addEventListener("click",()=>addDialog());
document.querySelector("#btn-discover").addEventListener("click",discover);
 document.querySelector("#appliance-switcher")?.addEventListener("change",e=>{if(e.target.value)choose(e.target.value)});
 document.querySelector("#back-to-devices")?.addEventListener("click",()=>switchView("devices"));
reload().catch(e=>{
 main.replaceChildren(empty("!","Desktop service unavailable",e.message,
  button("Retry connection",()=>reload().catch(err=>flash(err.message||"Unable to reconnect")),"btn btn-primary",false)));
 flash(e.message);
});
