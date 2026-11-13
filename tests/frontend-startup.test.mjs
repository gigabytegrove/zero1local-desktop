import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";
import { Script, runInNewContext } from "node:vm";

const root = new URL("../internal/companion/ui/", import.meta.url);
const source = readFileSync(new URL("app.js", root), "utf8");
const watchdog = readFileSync(new URL("startup.js", root), "utf8");
const html = readFileSync(new URL("index.html", root), "utf8");

class FakeElement {
  constructor(tag = "div") {
    this.tag = tag;
    this.children = [];
    this.attributes = {};
    this.dataset = {};
    this.classList = { toggle() {}, add() {}, remove() {} };
    this.textContent = "";
    this.value = "";
    this.disabled = false;
  }
  append(...children) { this.children.push(...children); }
  replaceChildren(...children) { this.children = children; }
  addEventListener() {}
  setAttribute(name, value) { this.attributes[name] = value; }
  removeAttribute(name) { delete this.attributes[name]; }
  focus() {}
  querySelector(selector) {
    if (selector === ".loading") return this.children.find(child => child?.className === "loading") || null;
    return null;
  }
}

test("credentialed onboarding, desktop management, and file features are present", () => {
  assert.ok(source.includes('"/api/add",payload'),"new appliances must go through credentialed enrollment");
  assert.ok(source.includes('"/api/nas/login"'),"previously saved appliances must support sign-in");
  assert.ok(source.includes('"/api/nas/read"'),"live management data should come from the NAS");
  assert.ok(source.includes('"/api/nas/file-action"'),"file actions must use scoped NAS APIs");
  assert.ok(source.includes('"/api/nas/download"'),"file downloads must not expose NAS cookies");
  assert.ok(source.includes('"/api/nas/upload?id="'),"NAS uploads must use the local authenticated bridge");
  for(const view of ["files","system","storage","network","docker","hardware","tasks","users","services","backups","protection","activity"]){
    assert.ok(html.includes('data-nav="'+view+'"'),"missing desktop navigation: "+view);
  }
});

test("the shipped frontend and startup watchdog are valid JavaScript", () => {
  assert.doesNotThrow(() => new Script(source, { filename: "app.js" }));
  assert.doesNotThrow(() => new Script(watchdog, { filename: "startup.js" }));
  assert.ok(html.indexOf('src="/startup.js"') < html.indexOf('src="/app.js"'), "watchdog must load first");
});

for (const populated of [false, true]) {
  test(`the desktop workspace renders from local API state (populated=${populated})`, async () => {
    const bySelector = new Map();
    for (const name of ["main", "device-rail", "modal", "modal-body", "toast", "top-title", "top-subtitle", "btn-discover", "btn-add"]) {
      bySelector.set("#" + name, new FakeElement(name));
    }
    const loading = new FakeElement("div");
    loading.className = "loading";
    bySelector.get("#main").children = [loading];
    const nav = ["home", "devices", "shares", "mounts", "settings"].map(name => {
      const e = new FakeElement("button");
      e.dataset.nav = name;
      return e;
    });
    const settings = {
      theme: "system",
      devices: populated ? [{ id: "nas1", name: "Sample NAS", url: "http://127.0.0.1:8088", version: "1.4.6" }] : [],
      mappings: populated ? [{ id: "m1", deviceId: "nas1", server: "nas", remote: "Documents", protocol: "smb", target: "Z:" }] : [],
    };
    const context = {
      document: {
        documentElement: { dataset: {} },
        querySelector: selector => selector === 'meta[name="csrf-token"]' ? { content: "test-token" } : bySelector.get(selector),
        querySelectorAll: selector => selector === "[data-nav]" ? nav : [],
        createElement: tag => new FakeElement(tag),
      },
      matchMedia: () => ({ matches: false, addEventListener() {} }),
      fetch: async path => {
        assert.equal(path, "/api/state");
        return { ok: true, json: async () => ({ settings, platform: "linux", updater: { supported: true, version: "0.1.0-beta.3" } }) };
      },
      window: { setTimeout() {}, addEventListener() {}, location: { reload() {} } },
      setTimeout() {}, clearTimeout() {}, URL,
    };
    runInNewContext(source, context, { filename: "app.js" });
    await new Promise(resolve => setImmediate(resolve));
    const main = bySelector.get("#main");
    assert.ok(main.children.length > 0, "main element must have content");
    assert.notEqual(main.children[0], loading, "loading screen must be replaced");
    assert.equal(bySelector.get("#top-title").textContent, "My appliances");
  });
}
