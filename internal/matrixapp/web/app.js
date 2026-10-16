// The Matrix app client. Text goes through textContent only; no innerHTML.
"use strict";

(() => {
  const main = document.getElementById("main");
  const $ = (id) => document.getElementById(id);
  const tasksByStep = {}; // step id -> latest run view, kept across re-renders

  function h(tag, props, ...kids) {
    const el = document.createElement(tag);
    for (const [k, v] of Object.entries(props ?? {})) {
      if (v == null || v === false) continue;
      if (k === "class") el.className = v;
      else if (k.startsWith("on")) el.addEventListener(k.slice(2), v);
      else if (k === "checked") el.checked = !!v;
      else el.setAttribute(k, v === true ? "" : v);
    }
    el.append(...kids.flat(Infinity).filter((x) => x != null && x !== false));
    return el;
  }
  // Line icons drawn on a 24px grid, stroked in the current text color.
  const ICONS = {
    chev: "M9 5l7 7-7 7", back: "M15 5l-7 7 7 7", check: "M5 12.5l4.5 4.5L19 7", x: "M7 7l10 10M17 7L7 17",
    folder: "M3.5 7.5A1.5 1.5 0 0 1 5 6h4l2 2h8a1.5 1.5 0 0 1 1.5 1.5v8A1.5 1.5 0 0 1 19 19H5a1.5 1.5 0 0 1-1.5-1.5z",
  };
  function icon(name) {
    const ns = "http://www.w3.org/2000/svg", svg = document.createElementNS(ns, "svg"), path = document.createElementNS(ns, "path");
    svg.setAttribute("viewBox", "0 0 24 24");
    svg.setAttribute("class", "icon i-" + name);
    svg.setAttribute("aria-hidden", "true");
    path.setAttribute("d", ICONS[name]);
    svg.append(path);
    return svg;
  }
  const show = (...nodes) => main.replaceChildren(...nodes.flat(Infinity).filter((x) => x != null && x !== false));
  const plural = (n, one, many) => `${n} ${n === 1 ? one : many ?? one + "s"}`;
  async function api(url, body) {
    const res = await fetch(url, body ? { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) } : { cache: "no-store" });
    const json = await res.json().catch(() => ({}));
    if (!res.ok) throw new Error(json.error || "Something went wrong (" + res.status + ")");
    return json;
  }
  function confirmBox(title, text, ok) {
    $("confirm-title").textContent = title;
    $("confirm-text").textContent = text;
    $("confirm-ok").textContent = ok;
    const dlg = $("confirm");
    dlg.returnValue = "";
    dlg.showModal();
    return new Promise((r) => dlg.addEventListener("close", () => r(dlg.returnValue === "ok"), { once: true }));
  }

  // ── projects ───────────────────────────────────────────
  async function projectsPage() {
    document.title = "The Matrix";
    const data = await api("/api/projects");
    const missing = data.missing_tools.length
      ? h("p", { class: "notice bad" }, "Missing: " + data.missing_tools.join(", ")) : null;
    show(missing,
      h("div", { class: "head-row" }, h("h1", null, "Projects"),
        h("button", { class: "btn primary", type: "button", onclick: () => pickFolder() }, "Add project")),
      data.projects.length
        ? h("ul", { class: "list" }, data.projects.map((p) => {
          const [state, text] = !p.exists ? ["bad", "Not found"]
            : !p.set_up ? ["", "Not set up"]
            : p.stale || p.issues ? ["warn", "Needs attention"]
            : ["ok", "Ready"];
          return h("li", null, h("a", { href: "#/project/" + encodeURIComponent(p.path), title: p.path },
            h("span", { class: "dot " + state, "aria-hidden": "true" }),
            h("strong", { class: "grow" }, p.name),
            h("span", { class: "status" }, text), icon("chev")));
        }))
        : h("p", { class: "empty muted" }, "No projects yet."));
  }

  // ── folder picker ──────────────────────────────────────
  let pickerPath = "~";
  async function pickFolder() {
    await showFolder("~");
    $("picker").showModal();
  }
  async function showFolder(path) {
    let data;
    try { data = await api("/api/browse?path=" + encodeURIComponent(path)); }
    catch (e) { $("picker-list").replaceChildren(h("li", { class: "none" }, e.message)); return; }
    pickerPath = data.path;
    const parts = data.path === "~" ? ["~"] : ["~", ...data.path.slice(2).split("/")];
    $("picker-path").replaceChildren(...parts.flatMap((name, i) => {
      const label = name === "~" ? "Home" : name;
      const target = i === 0 ? "~" : "~/" + parts.slice(1, i + 1).join("/");
      return [i ? icon("chev") : null,
        i === parts.length - 1 ? h("span", null, label) : h("button", { type: "button", onclick: () => showFolder(target) }, label)];
    }).filter(Boolean));
    $("picker-list").replaceChildren(...(data.folders.length ? data.folders.map((f) =>
      h("li", null, h("button", { type: "button", onclick: () => showFolder(f.path) }, icon("folder"), h("span", { class: "grow" }, f.name),
        f.project ? h("span", { class: "dot ok tag", title: "Set up" }) : null, icon("chev"))))
      : [h("li", { class: "none" }, "Empty")]));
    $("picker-use").disabled = data.path === "~";
  }
  $("picker-cancel").addEventListener("click", () => $("picker").close());
  $("picker-use").addEventListener("click", async () => {
    try {
      const p = await api("/api/projects", { path: pickerPath });
      $("picker").close();
      location.hash = "#/project/" + encodeURIComponent(p.path);
    } catch (e) { $("picker-list").replaceChildren(h("li", { class: "none" }, e.message)); }
  });

  // ── project page ───────────────────────────────────────
  let project = null;
  async function projectPage(path) {
    try { project = await api("/api/project?path=" + encodeURIComponent(path)); }
    catch (e) { return show(backLink(), h("p", { class: "notice bad" }, e.message)); }
    renderProject();
    if (anyBusy(project)) pollTimer = setTimeout(refresh, 2000);
  }

  function renderProject() {
    const p = project, hl = p.state.health;
    document.title = p.name + " · The Matrix";
    const docs = hl.knowledge.flatMap((k) => k.docs);
    const stale = docs.filter((d) => d.status === "stale").length;
    const missing = [...(hl.ecosystem.agents ?? []), ...(hl.ecosystem.skills ?? [])].filter((r) => !r.present);

    const research = p.research ?? [], b = p.build;
    const waiting = research.some((r) => ["review", "ready"].includes(r.phase));
    const s1 = p.set_up ? "done" : "current";
    const s2 = !p.set_up ? "later" : p.research_ready.length || waiting || research.some((r) => busy(r.job)) ? "current" : docs.length ? "done" : "";
    const s3 = !p.set_up ? "later" : stale || hl.fail ? "current" : hl.warn ? "" : "done";
    const s4 = !p.set_up ? "later" : busy(b.job) || b.result === "decisions" ? "current" : b.result === "approved" ? "done" : "";
    const remove = h("button", { class: "btn link", type: "button", onclick: async () => {
      if (await confirmBox("Remove " + p.name + "?", "Files stay on your computer.", "Remove")) {
        await api("/api/projects/remove", { path: p.path });
        location.hash = "#/";
      }
    } }, "Remove");

    show(
      backLink(),
      h("h1", { title: p.path }, p.name),
      h("ol", { class: "steps" },
        step(1, s1, "Agents", setupBody(p, missing)),
        step(2, s2, "Knowledge", p.set_up && knowledgeBody(p, docs)),
        step(3, s3, "Health", p.set_up && healthBody(hl, stale)),
        step(4, s4, "Build", p.set_up && buildBody(p))),
      h("p", { class: "actions" }, remove));
  }

  const backLink = () => h("a", { class: "back", href: "#/" }, icon("back"), "Projects");

  function step(n, state, title, body) {
    return h("li", { class: "step " + state },
      h("span", { class: "num", "aria-hidden": "true" }, state === "done" ? icon("check") : String(n)),
      h("div", null, h("h2", null, title), body));
  }

  function setupBody(p, missing) {
    if (p.set_up && !setupForm.open) {
      return [
        missing.length ? fact("warn", "Missing: " + missing.map((m) => m.name).join(", ")) : null,
        h("div", { class: "actions" },
          taskButton("check-setup", "Check", "setup"),
          h("button", { class: "btn link", type: "button", onclick: () => { setupForm.open = true; renderProject(); } }, "Redo")),
        taskStatus("setup"),
      ];
    }
    const form = h("form", { onsubmit: async (e) => {
      e.preventDefault();
      const data = new FormData(e.target);
      const params = { preset: data.get("preset"), runtime: data.get("runtime"), replace: p.set_up };
      const ok = p.set_up
        ? await confirmBox("Redo setup?", "Your changes to agent files will be replaced.", "Replace")
        : await confirmBox("Set up " + p.name + "?", "Adds agents and settings to this folder.", "Set up");
      if (ok) runTask("setup", "setup", params);
    } },
      choices("Type", "preset", p.presets.map((pr) => [pr.id, pr.name])),
      choices("Assistant", "runtime", [["claude", "Claude Code"], ["codex", "Codex"]]),
      h("div", { class: "actions" },
        h("button", { class: "btn primary", type: "submit" }, "Set up"),
        p.set_up ? h("button", { class: "btn link", type: "button", onclick: () => { setupForm.open = false; renderProject(); } }, "Cancel") : null));
    return [form, taskStatus("setup")];
  }
  const setupForm = { open: false };

  // A few short options become a segmented control; longer ones a checklist.
  // Both are plain radio groups underneath. items: [value, label, note?].
  function choices(legend, name, items, value) {
    const on = (v, i) => value === undefined ? i === 0 : v === value;
    const radio = (v, i) => h("input", { type: "radio", name, value: v, required: true, checked: on(v, i) });
    const seg = items.length <= 3 && items.every(([, label, note]) => !note && label.length <= 16);
    const group = seg
      ? h("div", { class: "seg" }, items.map(([v, label], i) => h("label", null, radio(v, i), h("span", null, label))))
      : h("div", { class: "options" }, items.map(([v, label, note], i) => h("label", { class: "option" },
        radio(v, i), h("span", { class: "grow" }, label), note ? h("span", { class: "note" }, note) : null, icon("check"))));
    if (seg) group.style.setProperty("--n", items.length);
    return h("fieldset", null, h("legend", null, legend), group);
  }

  const fact = (state, text) => h("ul", { class: "facts" }, h("li", null, h("span", { class: "dot " + state }), text));

  function knowledgeBody(p, docs) {
    const many = (p.research ?? []).length > 1;
    return [
      fact(docs.length ? "ok" : "", docs.length ? plural(docs.length, "document") : "None yet"),
      p.research_ready.length ? h("div", { class: "actions" }, taskButton("sync", "Add research", "knowledge", true)) : null,
      (p.research ?? []).map((r) => h("div", { class: "sub" }, many ? h("h3", null, r.stack) : null, researchBody(p, r))),
      taskStatus("knowledge"),
    ];
  }

  // ── research (long job) ────────────────────────────────
  const drafts = {}; // form values kept across re-renders, by form key
  const forms = {};  // form key -> open

  const PHASE = {
    running: "Researching", writing: "Writing docs", review: "Review within an hour",
    "review-timeout": "Review timed out", ready: "Docs ready", added: "Added", stopped: "Stopped", failed: "Failed",
  };

  function researchBody(p, r) {
    const key = "research:" + r.stack;
    const job = r.job;
    if (forms[key]) return researchForm(p, r, key);
    const lines = [];
    if (r.phase === "running") lines.push(working(job, r.cycle ? `Researching · round ${r.cycle} of ${r.cycles}` : job?.step || "Starting"));
    else if (r.phase === "writing") lines.push(working(job, "Writing docs"));
    else if (r.phase !== "none") lines.push(fact({ ready: "ok", added: "ok", review: "warn", failed: "bad" }[r.phase] ?? "warn",
      r.phase === "ready" ? PHASE[r.phase] + " · " + plural(r.docs, "document") : PHASE[r.phase]));

    if (r.phase === "review") {
      lines.push(h("ul", { class: "findings" }, r.found.map((f) => h("li", null,
          h("button", { class: "btn link", type: "button", onclick: () => readFinding(p, r, f) }, f.title)))));
    }

    const acts = [];
    if (r.phase === "review") acts.push(h("button", { class: "btn primary", type: "button", onclick: () => approve(p, r) }, "Write docs"));
    if (r.phase === "ready") acts.push(h("button", { class: "btn primary", type: "button", disabled: tasksByStep["add:" + r.stack]?.status === "running", onclick: async () => {
      if (await confirmBox("Add for agents?", "Copies the docs into this project.", "Add")) runTask("add-knowledge", "add:" + r.stack, { stack: r.stack });
    } }, "Add for agents"));
    if (busy(job)) acts.push(stopButton(job));
    else acts.push(h("button", { class: r.phase === "none" ? "btn" : "btn link", type: "button", disabled: !p.runtime_ready,
      onclick: () => { forms[key] = true; renderProject(); } }, r.phase === "none" ? "Research" : "Research again"));
    lines.push(h("div", { class: "actions" }, acts), runtimeNote(p), taskStatus("add:" + r.stack), jobDetails(job));
    return lines;
  }

  function researchForm(p, r, key) {
    const d = (drafts[key] ??= { ...r.defaults });
    const keep = (e) => { const el = e.target; d[el.name] = el.type === "checkbox" ? el.checked : el.value; };
    return h("form", { class: "form", oninput: keep, onchange: keep, onsubmit: async (e) => {
      e.preventDefault();
      const again = r.phase !== "none" ? " Replaces earlier research." : "";
      if (!(await confirmBox("Start research?", `Uses your ${p.runtime_name} account. Can take an hour or more.` + again, "Start"))) return;
      startJob(p, "research", { research: { ...d, stack: r.stack } }, key);
    } },
      field("Built with", h("input", { name: "framework", value: d.framework, required: true, maxlength: 120, placeholder: "Go 1.24 with chi" })),
      choices("Kind", "app_type", [["api-only", "Service or API"], ["full-stack", "Website with backend"], ["frontend", "Website or app front end"], ["cli", "Command-line tool"], ["library", "Library"]], d.app_type),
      field("Data", h("input", { name: "data_layer", value: d.data_layer, maxlength: 120, placeholder: "Optional" })),
      field("Tests", h("input", { name: "testing", value: d.testing, maxlength: 120, placeholder: "go test" })),
      check("queue", "Message queue", d.queue),
      check("review", "Review before writing", d.review),
      h("div", { class: "actions" },
        h("button", { class: "btn primary", type: "submit" }, "Start research"),
        h("button", { class: "btn link", type: "button", onclick: () => { forms[key] = false; renderProject(); } }, "Cancel")),
      formError(key));
  }

  const field = (label, input) => h("label", { class: "field" }, h("span", null, label), input);
  const check = (name, label, on) => h("label", { class: "check" }, h("span", null, label), h("input", { type: "checkbox", role: "switch", name, checked: on }));
  const formErrors = {};
  const formError = (key) => formErrors[key] ? h("p", { class: "notice bad", role: "alert" }, formErrors[key]) : null;
  const runtimeNote = (p) => p.runtime_ready ? null : fact("bad", p.runtime_name + " is not installed");

  async function readFinding(p, r, f) {
    $("reader-title").textContent = f.title;
    $("reader-text").textContent = "Loading…";
    $("reader").showModal();
    try {
      const q = new URLSearchParams({ path: p.path, stack: r.stack, name: f.name });
      $("reader-text").textContent = (await api("/api/research/file?" + q)).text;
    } catch (e) { $("reader-text").textContent = e.message; }
  }
  $("reader-close").addEventListener("click", () => $("reader").close());

  async function approve(p, r) {
    if (!(await confirmBox("Write docs?", "Turns the findings into docs.", "Write docs"))) return;
    try { await api("/api/research/approve", { path: p.path, stack: r.stack, confirm: true }); }
    catch (e) { formErrors["research:" + r.stack] = e.message; }
    refresh(true);
  }

  // ── build (long job) ───────────────────────────────────
  const RESULT = {
    approved: ["ok", "Finished"], "rounds-used": ["warn", "Out of rounds"],
    decisions: ["warn", "Needs a decision"], stopped: ["warn", "Stopped"], failed: ["bad", "Failed"],
  };

  function buildBody(p) {
    const b = p.build, job = b.job, key = "build";
    if (forms[key]) return buildForm(p, key);
    const lines = [];
    if (b.goal) lines.push(h("p", { class: "goal" }, b.goal));
    if (busy(job)) lines.push(working(job, b.round ? `Round ${b.round} of ${b.rounds}` + (b.activity ? " · " + b.activity : "") : job.step));
    else if (b.result) lines.push(fact(...RESULT[b.result]));
    if (b.total) lines.push(fact(b.done === b.total ? "ok" : "", `${b.done} of ${plural(b.total, "task")} done`));
    if (b.result === "decisions" && b.decisions.length) {
      lines.push(h("ul", { class: "findings" }, b.decisions.map((d) => h("li", null, d))));
    }
    if (!b.git) lines.push(fact("warn", "Needs Git"));
    const acts = [];
    if (busy(job)) acts.push(stopButton(job));
    else {
      const blocked = !p.runtime_ready || !b.git;
      if (b.exists && job) acts.push(h("button", { class: "btn primary", type: "button", disabled: blocked, onclick: async () => {
        if (await confirmBox("Continue?", `Uses your ${p.runtime_name} account.`, "Continue")) startJob(p, "build", { build: { continue: true } }, key);
      } }, b.result === "decisions" ? "Let agents decide" : "Continue"));
      acts.push(h("button", { class: b.exists && job ? "btn link" : "btn", type: "button", disabled: blocked,
        onclick: () => { forms[key] = true; renderProject(); } }, b.exists && job ? "New goal" : "Start"));
    }
    lines.push(h("div", { class: "actions" }, acts), runtimeNote(p), formError(key), jobDetails(job));
    return lines;
  }

  function buildForm(p, key) {
    const d = (drafts[key] ??= { goal: "", size: "medium" });
    const keep = (e) => { d[e.target.name] = e.target.value; };
    return h("form", { class: "form", oninput: keep, onchange: keep, onsubmit: async (e) => {
      e.preventDefault();
      const again = p.build.exists ? " Replaces the task list." : "";
      if (!(await confirmBox("Start building?", `Uses your ${p.runtime_name} account. Can take hours.` + again, "Start"))) return;
      startJob(p, "build", { build: { goal: d.goal, size: d.size } }, key);
    } },
      field("Goal", h("textarea", { name: "goal", rows: 4, required: true, minlength: 10, maxlength: 2000, placeholder: "Add a checkout page with card payments" }, d.goal)),
      choices("How long", "size", [["small", "Short", "Up to 3 rounds"], ["medium", "Medium", "Up to 6 rounds"], ["large", "Long", "Up to 10 rounds"]], d.size),
      h("div", { class: "actions" },
        h("button", { class: "btn primary", type: "submit" }, "Start building"),
        h("button", { class: "btn link", type: "button", onclick: () => { forms[key] = false; renderProject(); } }, "Cancel")),
      formError(key));
  }

  // ── long jobs ──────────────────────────────────────────
  const busy = (j) => j?.status === "running";
  const logs = {}; // job id -> { open, text, next }

  const anyBusy = (p) => [...(p.research ?? []).map((r) => r.job), p.build.job].some(busy);

  function working(job, text) {
    const el = h("div", { class: "task running", role: "status", "aria-live": "polite" },
      h("div", { class: "task-line" }, h("span", { class: "spin", "aria-hidden": "true" }), h("span", null, text),
        job ? h("span", { class: "muted elapsed" }, elapsed(job.duration_ms)) : null));
    el.style.setProperty("--t", -(performance.now() % 1600) + "ms"); // same clock as taskStatus
    return el;
  }

  function elapsed(ms) {
    const m = Math.floor(ms / 60000);
    return m < 1 ? "just started" : m < 60 ? `${m} min` : `${Math.floor(m / 60)} h ${m % 60} min`;
  }

  const stopButton = (job) => h("button", { class: "btn", type: "button", onclick: async () => {
    if (!(await confirmBox("Stop?", "Finished work stays.", "Stop"))) return;
    try { await api(`/api/jobs/${job.id}/stop`, {}); } catch { /* already finished */ }
    refresh(true);
  } }, "Stop");

  function jobDetails(job) {
    if (!job) return null;
    const log = (logs[job.id] ??= { open: job.status === "fail", text: "", next: 0 });
    if (log.open && !log.next) loadLog(job.id);
    return h("details", { open: log.open || null, ontoggle: (e) => { log.open = e.target.open; if (log.open) loadLog(job.id); } },
      h("summary", null, "Details"),
      h("pre", { "data-log": job.id }, job.commands.map((c) => "$ " + c).join("\n") + "\n" + log.text));
  }

  async function loadLog(id) {
    const log = logs[id];
    if (!log || log.loading) return;
    log.loading = true;
    try {
      const v = await api(`/api/jobs/${id}?from=${log.next}`);
      if (v.output) {
        log.text += v.output;
        log.next = v.next;
        const pre = document.querySelector(`[data-log="${id}"]`);
        if (pre) { const end = pre.scrollTop + pre.clientHeight >= pre.scrollHeight - 4; pre.textContent += v.output; if (end) pre.scrollTop = pre.scrollHeight; }
        else renderProject();
      }
      else log.next = v.next || 1;
    } catch { /* keep what we have */ }
    log.loading = false;
  }

  async function startJob(p, job, body, key) {
    delete formErrors[key];
    try {
      const v = await api("/api/jobs", { job, path: p.path, confirm: true, ...body });
      forms[key] = false;
      delete drafts[key];
      logs[v.id] = { open: false, text: "", next: 0 };
    } catch (e) { formErrors[key] = e.message; }
    refresh(true);
  }

  // refresh reloads the project; while a job runs it keeps polling. It waits
  // while someone is typing so a re-render never eats their text.
  let pollTimer = null;
  const seen = (p) => JSON.stringify(p, (k, v) => k === "duration_ms" ? Math.floor(v / 60000) : k === "produced_at" ? undefined : v);
  async function refresh(now) {
    clearTimeout(pollTimer);
    if (!project) return;
    const path = project.path;
    const typing = () => main.contains(document.activeElement) && /^(INPUT|TEXTAREA)$/.test(document.activeElement.tagName) && document.activeElement.type !== "radio" && document.activeElement.type !== "checkbox";
    if (!now && typing()) { pollTimer = setTimeout(refresh, 2000); return; }
    try {
      const fresh = await api("/api/project?path=" + encodeURIComponent(path));
      if (!project || project.path !== path) return;
      const changed = now || seen(fresh) !== seen(project);
      project = fresh;
      for (const j of [...(fresh.research ?? []).map((r) => r.job), fresh.build.job]) if (busy(j) && logs[j.id]?.open) await loadLog(j.id);
      // Re-render only when something visible changed: replacing the page
      // every poll would drop clicks and keyboard focus.
      if (changed) renderProject();
    } catch { /* try again on the next tick */ }
    if (anyBusy(project)) pollTimer = setTimeout(refresh, 2000);
  }

  function healthBody(hl, stale) {
    return [
      fact(hl.fail ? "bad" : hl.warn || stale ? "warn" : "ok",
        hl.fail ? plural(hl.fail, "problem") : stale ? plural(stale, "outdated document") : hl.warn ? plural(hl.warn, "warning") : "All good"),
      h("div", { class: "actions" }, taskButton("health", "Check", "health")),
      taskStatus("health"),
    ];
  }

  // ── tasks ──────────────────────────────────────────────
  function taskButton(task, label, step, writes) {
    const running = tasksByStep[step]?.status === "running";
    return h("button", { class: "btn", type: "button", disabled: running, onclick: async () => {
      if (writes && !(await confirmBox(label + "?", "Copies the research into this project.", label))) return;
      runTask(task, step, {});
    } }, label);
  }

  function taskStatus(step) {
    const run = tasksByStep[step];
    if (!run) return null;
    const failed = run.status !== "running" && run.status !== "ok";
    const text = { running: "Working…", ok: "Done", timeout: "Timed out", canceled: "Stopped" }[run.status] ?? "Failed";
    const key = run.id + ":" + run.status, fresh = !shownTasks.has(key);
    shownTasks.add(key);
    const el = h("div", { class: ["task", run.status === "ok" ? "ok" : failed ? "bad" : "running", fresh && "fresh"].filter(Boolean).join(" "), role: "status", "aria-live": "polite" },
      h("div", { class: "task-line" }, run.status === "running" ? h("span", { class: "spin", "aria-hidden": "true" }) : icon(run.status === "ok" ? "check" : "x"), h("span", null, text)),
      run.output ? h("details", { open: failed ? true : null }, h("summary", null, "Details"), h("pre", null, run.output)) : null);
    // Polling rebuilds this node every 300ms; a clock-based delay keeps the motion continuous.
    el.style.setProperty("--t", -(performance.now() % 1600) + "ms");
    return el;
  }
  const shownTasks = new Set();

  async function runTask(task, step, params) {
    try {
      let view = await api("/api/tasks", { task, path: project.path, params, confirm: true });
      let out = view.output;
      tasksByStep[step] = { ...view, output: out };
      renderProject();
      while (view.status === "running") {
        await new Promise((r) => setTimeout(r, 300));
        view = await api(`/api/tasks/${view.id}?from=${view.next}`);
        out += view.output;
        tasksByStep[step] = { ...view, output: out };
        if (view.status === "running") renderProject();
      }
      if (task === "setup") setupForm.open = false;
      const fresh = await api("/api/project?path=" + encodeURIComponent(project.path));
      project = fresh;
      renderProject();
    } catch (e) {
      tasksByStep[step] = { action: task, status: "fail", output: e.message };
      renderProject();
    }
  }

  // ── router ─────────────────────────────────────────────
  async function route() {
    const m = location.hash.match(/^#\/project\/(.+)$/);
    main.classList.add("enter");
    setTimeout(() => main.classList.remove("enter"), 700);
    try {
      if (m) await projectPage(decodeURIComponent(m[1]));
      else await projectsPage();
    } catch (e) {
      show(h("p", { class: "notice bad" }, "Can't reach The Matrix."));
    }
    main.focus({ preventScroll: true });
  }
  window.addEventListener("hashchange", () => {
    clearTimeout(pollTimer);
    project = null;
    for (const o of [tasksByStep, drafts, forms, formErrors]) Object.keys(o).forEach((k) => delete o[k]);
    setupForm.open = false;
    route();
  });
  route();
})();
