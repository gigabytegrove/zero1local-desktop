import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {test} from "node:test";
import {runInNewContext} from "node:vm";

const source=readFileSync(new URL("../internal/companion/ui/app.js",import.meta.url),"utf8");

class UIElement{
 constructor(tag="div"){
  this.tag=tag;this.children=[];this.dataset={};this.attributes={};this.handlers={};
  this.id="";this.value="";this.textContent="";this.disabled=false;this.hidden=false;
  this.classList={add(){},remove(){},toggle(){}};
 }
 append(...children){this.children.push(...children)}
 replaceChildren(...children){this.children=children}
 addEventListener(event,fn){this.handlers[event]=fn}
 setAttribute(key,value){this.attributes[key]=String(value);if(key==="id")this.id=String(value)}
 removeAttribute(key){delete this.attributes[key]}
 focus(){}
 querySelector(){return null}
}
function textOf(v){
 if(typeof v==="string")return v;
 if(!(v instanceof UIElement))return "";
 return [v.textContent,...v.children.map(textOf)].join(" ");
}
function findButtons(node){
 if(!(node instanceof UIElement))return [];
 return [...(node.tag==="button"?[node]:[]),...node.children.flatMap(findButtons)];
}
async function showSettings(updater){
 const elements=new Map(["main","device-rail","modal","modal-body","modal-title","modal-form",
  "modal-submit","toast","top-title","top-subtitle","btn-discover","btn-add",
  "appliance-switcher","topbar-presence","nav-management","device-navigation",
  "workspace-switcher","device-nav-title","back-to-devices"].map(k=>["#"+k,new UIElement(k)]));
 const nav=["home","devices","files","shares","mounts","system","storage","network","docker","hardware",
  "tasks","users","services","backups","protection","activity","settings"].map(name=>{
  const elem=new UIElement("button");elem.dataset.nav=name;return elem;
 });
 const document={
  documentElement:{dataset:{}},
  querySelector:query=>query==='meta[name="csrf-token"]'?{content:"test-token"}:elements.get(query),
  querySelectorAll:query=>query==="[data-nav]"?nav:[],
  createElement:tag=>new UIElement(tag),
  createTextNode:text=>String(text),
  body:new UIElement("body")
 };
 const settings={theme:"system",devices:[],mappings:[]};
 const initial={
  settings,platform:"linux",sessions:{},updater,
  credentialVaultAvailable:true
 };
 runInNewContext(source,{
  document,fetch:async(path)=>{
   if(path==="/api/state")return {ok:true,json:async()=>initial};
   if(path==="/api/updates/check")return {ok:true,json:async()=>({started:true})};
   if(path==="/api/updates/install")return {ok:true,json:async()=>({started:true})};
   if(path==="/api/updates/restart")return {ok:true,json:async()=>({restarting:true})};
   throw new Error("Unexpected network request: "+path);
  },
  matchMedia:()=>({matches:false,addEventListener(){}}),
  window:{location:{reload(){}},setTimeout(){},addEventListener(){}},
  setTimeout(){},clearTimeout(){},URL,
 });
 await new Promise(resolve=>setImmediate(resolve));
 const settingsNav=nav.find(n=>n.dataset.nav==="settings");
 assert.ok(settingsNav.handlers.click);
 settingsNav.handlers.click();
 return {main:elements.get("#main"),buttons:findButtons(elements.get("#main"))};
}
function status(overrides={}){
 return {version:"0.1.0-beta.7",channel:"beta",supported:true,installMode:"debian",
  canInstall:true,phase:"idle",message:"Check for the latest release.",
  checking:false,installing:false,available:false,lastError:"",...overrides};
}
test("Debian installation can always check for updates",async()=>{
 const view=await showSettings(status());
 const check=view.buttons.find(n=>n.textContent==="Check for updates");
 assert.ok(check,"missing Check for updates button");
 assert.equal(check.disabled,false,"check button was disabled on Debian");
 assert.doesNotMatch(textOf(view.main),/Download verified \.deb|Download \.deb|install it manually/i);
});
test("an available release exposes an explicit Install update action",async()=>{
 const view=await showSettings(status({
  available:true,latestVersion:"0.1.0-beta.8",phase:"available",
  message:"Version 0.1.0-beta.8 is ready to install."
 }));
 assert.ok(view.buttons.some(n=>n.textContent==="Install update"&&!n.disabled));
 assert.match(textOf(view.main),/0.1.0-beta.8/);
});
test("Debian upgrade completion offers restart, not another download",async()=>{
 const view=await showSettings(status({
  phase:"restart",available:false,restartReady:true,
  message:"Update installed. Restart Zero1Local Desktop."
 }));
 assert.ok(view.buttons.some(n=>n.textContent==="Restart Desktop now"&&!n.disabled));
 assert.ok(!view.buttons.some(n=>n.textContent==="Download verified .deb"));
});
test("failure is visible and retry remains possible",async()=>{
 const view=await showSettings(status({
  phase:"error",lastError:"PolicyKit authorization was denied."
 }));
 assert.match(textOf(view.main),/PolicyKit authorization was denied/);
 assert.ok(view.buttons.some(n=>n.textContent==="Check for updates"&&!n.disabled));
});
test("unsupported runtimes don't falsely claim update installation",async()=>{
 const view=await showSettings(status({
  supported:false,canInstall:false,installMode:"unavailable",
  message:"Native desktop updater unavailable"
 }));
 const check=view.buttons.find(n=>n.textContent==="Check for updates");
 assert.ok(check?.disabled);
 assert.match(textOf(view.main),/Native desktop updater unavailable/);
});
