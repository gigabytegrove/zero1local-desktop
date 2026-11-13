"use strict";
(() => {
  let reported = false;
  function showFailure(message) {
    const target = document.querySelector("#main");
    if (reported || !target || !target.querySelector(".loading")) return;
    reported = true;
    const panel = document.createElement("section");
    panel.className = "startup-error";
    panel.setAttribute("role","alert");

    const title = document.createElement("h1");
    title.textContent = "Desktop workspace could not start";
    const description = document.createElement("p");
    description.textContent = "The app window opened, but the local interface did not initialize. You can retry without changing any saved NAS settings.";
    const detail = document.createElement("p");
    detail.className = "startup-detail";
    detail.textContent = message;
    const retry = document.createElement("button");
    retry.className = "btn btn-primary";
    retry.type = "button";
    retry.textContent = "Reload workspace";
    retry.addEventListener("click", () => window.location.reload());
    panel.append(title, description, detail, retry);
    target.replaceChildren(panel);
  }
  window.addEventListener("error", (event) => {
    if (event.filename && !/\/(?:app|startup)\.js(?:\?|$)/.test(event.filename)) return;
    showFailure("Frontend JavaScript error: " + (event.message || "script failed to load"));
  });
  window.addEventListener("unhandledrejection", (event) => {
    showFailure("Frontend initialization error: " + String(event.reason?.message || event.reason || "unexpected error"));
  });
  window.setTimeout(() => showFailure("The frontend did not finish loading. Check that the localhost service is reachable, then retry."), 8000);
})();
