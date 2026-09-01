"use strict";

const REFRESH_INTERVAL_MS = 15_000;
const STALE_AFTER_SECONDS = 180;
const params = new URLSearchParams(window.location.search);
const demoMode = params.get("demo") === "1";

const messages = {
  en: {
    skip: "Skip to content", readOnly: "Read-only", refresh: "Refresh backup status",
    loading: "Loading", connecting: "Connecting to backup status",
    autoRefresh: "This page refreshes automatically; no shell query is required.", noData: "No status received",
    coreMetrics: "Core backup metrics", recoveryReady: "Recovery readiness", checking: "Checking",
    waitingAudit: "Waiting for restore verification", offsiteGeneration: "Offsite generation",
    comparingGeneration: "Comparing local and offsite markers", verifiedRestore: "Verified restorable",
    automation: "Automation", pipelineTitle: "Backup pipeline",
    pipelineIntro: "Incremental backup, offsite synchronization, and scheduled restore verification.",
    syncStorage: "Synchronization and storage status", syncTitle: "Synchronization", readingProgress: "Reading transfer progress.",
    transferProgress: "Current transfer progress", currentProgress: "Progress", currentSpeed: "Speed", estimatedRemaining: "ETA",
    localGeneration: "Local generation", remoteGeneration: "Offsite generation", localRepository: "Local repository",
    storageTitle: "Backup storage", storageUsage: "Backup storage usage", used: "Used", available: "Available",
    readingPool: "Reading storage pool health.", recoverability: "Recoverability", auditTitle: "Latest restore verification",
    waitingAuditData: "Waiting for verification data", retentionPolicy: "Snapshot retention policy",
    remoteSnapshots: "offsite snapshots", keepLast: "keep last", keepDaily: "keep daily", recoveryChecks: "Recovery checks",
    datasetCaption: "Verified restore size and snapshots by dataset", dataset: "Dataset", verifiedSize: "Verified size",
    localSnapshot: "Local snapshot", remoteSnapshot: "Offsite snapshot", eventsRuntime: "Events and runtime dependencies",
    recentRecords: "Recent records", eventsTitle: "Backup events", maxEvents: "Up to 12", dependencies: "Dependencies",
    runtimeTitle: "Runtime environment", noWritePermission: "The dashboard has no restore or delete permission",
    securityDetail: "The browser reads sanitized status only; credentials stay on the backup host.",
    browserRefresh: "Browser refresh: 15 seconds", collectorRefresh: "Recommended collector interval: 1 minute",
    statusGenerated: "Status generated {value}", backupHost: "Backup host", canRestore: "Ready", needsCheck: "Needs review",
    latestAuditPassed: "Latest restore verification passed", latestAuditUnavailable: "No current restore verification",
    caughtUp: "Caught up", newerGeneration: "New generation", unknown: "Unknown", remoteCurrent: "Offsite contains the latest local generation",
    generationLag: "Generation lag: {value}", waitingMarkers: "Waiting for generation markers", auditedAt: "Verified {value}", noAudit: "No restore verification recorded",
    justNow: "just now", seconds: "{value}s", minutes: "{value}m", hours: "{value}h {minutes}m", days: "{value}d {hours}h",
    ago: "{value} ago", timeUnknown: "time unknown", taskRunning: "Task is running",
    lockSkipped: "Skipped because another protected task holds the lock; it will retry automatically",
    lastSuccessOkay: "Last run succeeded; waiting for the next scheduled trigger", delayed: "Last success is older than the preferred interval",
    timerUnavailable: "Timer is unavailable", timerInactive: "Timer is not active", taskFailed: "Last run failed (exit {value})",
    noSuccess: "No valid success marker is available", lastSuccess: "Last success", duration: "Duration", nextRun: "Next run",
    timerDetermines: "Timer controlled", uploading: "Transferring encrypted repository data", offsiteCurrent: "Offsite is current; waiting for the next change.",
    complete: "100% · current", idle: "Idle", noWait: "No wait", syncRunning: "Synchronization is running",
    waitingSync: "A new local generation is waiting for automatic synchronization.", waiting: "Waiting",
    poolHealthy: "Pool healthy", needsAttention: "Needs attention", poolUnavailable: "Storage pool status is unavailable",
    sampleDetail: "{time} · {age} · {percent}% encrypted data sample", noChecks: "No verification checks",
    checkPassed: "Latest check passed", checkMissing: "No fresh passing result", noDatasets: "No dataset records",
    noEvents: "No backup events", demoNotice: "Showing embedded demonstration data, not a live backup host.",
    staleNotice: "Status has not updated for {value}; check the collector timer.", unsupportedSchema: "Unsupported status schema",
    fetchFailed: "Unable to read backup status ({value}). Backup jobs continue independently.", connectionFailed: "Connection failed",
    panelUnavailable: "Dashboard status is unavailable", panelUnavailableDetail: "Check the collector and dashboard service. Existing backup jobs are not affected.",
    overallTitleHealthy: "Backup protection is current", overallTitleWarning: "Backup protection is delayed",
    overallTitleError: "Backup protection needs attention", overallTitleRunning: "Backup is running safely", overallTitleUnknown: "Backup status is uncertain",
    overallSummaryHealthy: "Backup, offsite sync, and recovery verification are within policy.",
    overallSummaryWarning: "The workflow is delayed or has incomplete status data.", overallSummaryError: "At least one backup control needs attention.",
    overallSummaryRunning: "A protected backup workflow is currently running.", overallSummaryUnknown: "A complete backup status is not available.",
    statusHealthy: "Healthy", statusWarning: "Delayed", statusError: "Attention", statusRunning: "Running", statusUnknown: "Unknown", statusLoading: "Loading"
  },
  zh: {
    skip: "跳到主要内容", readOnly: "只读面板", refresh: "立即刷新备份状态",
    loading: "读取中", connecting: "正在连接备份状态", autoRefresh: "面板会自动刷新，无需运行查询命令。", noData: "尚未取得数据",
    coreMetrics: "备份核心指标", recoveryReady: "恢复准备", checking: "检查中", waitingAudit: "等待恢复抽检状态",
    offsiteGeneration: "异地代次", comparingGeneration: "正在对比本地与异地代次", verifiedRestore: "已验证可恢复",
    automation: "自动化链路", pipelineTitle: "备份流水线", pipelineIntro: "增量备份完成后自动异地同步，并按计划执行恢复抽检。",
    syncStorage: "同步与存储状态", syncTitle: "异地同步", readingProgress: "正在读取传输进度。", transferProgress: "当前传输进度",
    currentProgress: "本次进度", currentSpeed: "实时速度", estimatedRemaining: "预计剩余", localGeneration: "本地成功代次",
    remoteGeneration: "异地已上传代次", localRepository: "本地仓库", storageTitle: "备份存储", storageUsage: "备份存储空间使用率",
    used: "已用", available: "可用", readingPool: "正在读取存储池健康状态。", recoverability: "可恢复性",
    auditTitle: "最近一次恢复抽检", waitingAuditData: "等待抽检数据", retentionPolicy: "快照保留策略",
    remoteSnapshots: "份异地快照", keepLast: "保留最近", keepDaily: "按日保留", recoveryChecks: "恢复校验项目",
    datasetCaption: "各数据组最近可恢复容量与快照", dataset: "数据组", verifiedSize: "已验证容量", localSnapshot: "本地快照",
    remoteSnapshot: "异地快照", eventsRuntime: "事件与运行依赖", recentRecords: "最近记录", eventsTitle: "备份事件",
    maxEvents: "最多 12 条", dependencies: "底层依赖", runtimeTitle: "运行环境", noWritePermission: "面板没有恢复或删除权限",
    securityDetail: "浏览器只能读取脱敏状态，备份凭据始终留在宿主机。", browserRefresh: "浏览器刷新间隔：15 秒",
    collectorRefresh: "建议采集间隔：1 分钟", statusGenerated: "状态生成于 {value}", backupHost: "备份主机", canRestore: "可以恢复",
    needsCheck: "需要检查", latestAuditPassed: "最近一次恢复抽检已通过", latestAuditUnavailable: "没有新鲜的恢复抽检结论",
    caughtUp: "已追平", newerGeneration: "有新代次", unknown: "未知", remoteCurrent: "异地已包含最新本地成功代次",
    generationLag: "当前代次差 {value}", waitingMarkers: "等待代次标记", auditedAt: "抽检于 {value}", noAudit: "尚无恢复抽检记录",
    justNow: "刚刚", seconds: "{value} 秒", minutes: "{value} 分钟", hours: "{value} 小时 {minutes} 分", days: "{value} 天 {hours} 小时",
    ago: "{value}前", timeUnknown: "时间未知", taskRunning: "任务运行中",
    lockSkipped: "本次因其他受保护任务占用锁而跳过，将自动重试", lastSuccessOkay: "最近一次成功，等待下次自动触发",
    delayed: "最近成功时间已超过建议间隔", timerUnavailable: "定时器未安装或不可读取", timerInactive: "定时器当前未启用",
    taskFailed: "上次任务失败（退出码 {value}）", noSuccess: "尚未取得有效的成功记录", lastSuccess: "最近成功", duration: "上次耗时",
    nextRun: "下次触发", timerDetermines: "由定时器决定", uploading: "正在传输加密仓库数据", offsiteCurrent: "异地已追平，等待下一次变化。",
    complete: "100% · 已追平", idle: "当前空闲", noWait: "无需等待", syncRunning: "同步任务运行中",
    waitingSync: "本地产生了新代次，等待下一轮自动同步。", waiting: "等待同步", poolHealthy: "存储池健康", needsAttention: "需检查",
    poolUnavailable: "未取得存储池状态", sampleDetail: "{time} · {age} · 抽样读取 {percent}% 加密数据", noChecks: "暂无校验数据",
    checkPassed: "最近一次检查通过", checkMissing: "没有新鲜的通过结论", noDatasets: "暂无数据组记录", noEvents: "暂无备份事件",
    demoNotice: "当前显示的是界面演示数据，不是实时备份状态。", staleNotice: "状态数据已经 {value} 没有更新，请检查采集定时器。",
    unsupportedSchema: "不支持的状态格式", fetchFailed: "无法读取备份状态（{value}）。备份任务本身不会因此停止。",
    connectionFailed: "连接失败", panelUnavailable: "面板暂时无法取得状态", panelUnavailableDetail: "请检查采集器和面板服务；现有备份任务独立运行，不受网页影响。",
    overallTitleHealthy: "备份链路运行正常", overallTitleWarning: "备份链路出现延迟", overallTitleError: "备份链路需要处理",
    overallTitleRunning: "正在安全备份", overallTitleUnknown: "备份状态暂不明确", overallSummaryHealthy: "备份、异地同步和恢复抽检均在策略范围内。",
    overallSummaryWarning: "备份链路存在延迟或状态数据不完整。", overallSummaryError: "至少有一项备份控制需要处理。",
    overallSummaryRunning: "受保护的备份流程正在运行。", overallSummaryUnknown: "当前没有完整的备份状态。",
    statusHealthy: "正常", statusWarning: "有延迟", statusError: "需处理", statusRunning: "运行中", statusUnknown: "未知", statusLoading: "读取中"
  }
};

let language = preferredLanguage();
let lastData = null;

const byId = (id) => document.getElementById(id);

function preferredLanguage() {
  try {
    const saved = window.localStorage.getItem("dashboard-language");
    if (saved === "en" || saved === "zh") return saved;
  } catch (_) { /* Storage can be disabled without breaking the dashboard. */ }
  return navigator.language.toLowerCase().startsWith("zh") ? "zh" : "en";
}

function t(key, variables = {}) {
  let value = messages[language][key] ?? messages.en[key] ?? key;
  for (const [name, replacement] of Object.entries(variables)) {
    value = value.replaceAll(`{${name}}`, String(replacement));
  }
  return value;
}

function applyLanguage() {
  document.documentElement.lang = language === "zh" ? "zh-CN" : "en";
  document.querySelectorAll("[data-i18n]").forEach((element) => { element.textContent = t(element.dataset.i18n); });
  document.querySelectorAll("[data-i18n-aria]").forEach((element) => { element.setAttribute("aria-label", t(element.dataset.i18nAria)); });
  document.querySelectorAll("[data-i18n-title]").forEach((element) => { element.title = t(element.dataset.i18nTitle); });
  byId("language-button").textContent = language === "zh" ? "EN" : "中文";
  byId("language-button").setAttribute("aria-label", language === "zh" ? "Switch to English" : "切换为中文");
  if (lastData) render(lastData);
}

function setText(id, value) { byId(id).textContent = value ?? "—"; }

function makeElement(tag, className, text) {
  const element = document.createElement(tag);
  if (className) element.className = className;
  if (text !== undefined && text !== null) element.textContent = text;
  return element;
}

function normalizeStatus(status) {
  return ["healthy", "warning", "error", "running", "unknown", "loading"].includes(status) ? status : "unknown";
}

function statusLabel(status) {
  const normalized = normalizeStatus(status);
  return t(`status${normalized[0].toUpperCase()}${normalized.slice(1)}`);
}

function applyStatusPill(element, status, label) {
  const normalized = normalizeStatus(status);
  const dot = makeElement("span", "status-dot");
  dot.setAttribute("aria-hidden", "true");
  element.className = `status-pill status-${normalized}`;
  element.replaceChildren(dot, makeElement("span", "", label || statusLabel(normalized)));
}

function formatBytes(value) {
  if (!Number.isFinite(value) || value < 0) return "—";
  const units = ["B", "KiB", "MiB", "GiB", "TiB", "PiB"];
  const index = value === 0 ? 0 : Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
  const scaled = value / (1024 ** index);
  const decimals = index >= 4 ? 2 : index >= 2 ? 1 : 0;
  return `${scaled.toFixed(decimals)} ${units[index]}`;
}

function formatSpeed(value) {
  const formatted = formatBytes(value);
  return formatted === "—" ? formatted : `${formatted}/s`;
}

function formatDuration(value) {
  if (!Number.isFinite(value) || value < 0) return "—";
  if (value < 60) return t("seconds", { value: Math.round(value) });
  const days = Math.floor(value / 86400);
  const hours = Math.floor((value % 86400) / 3600);
  const minutes = Math.floor((value % 3600) / 60);
  if (days) return t("days", { value: days, hours });
  if (hours) return t("hours", { value: hours, minutes });
  return t("minutes", { value: minutes });
}

function formatDateTime(value, full = false) {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return new Intl.DateTimeFormat(language === "zh" ? "zh-CN" : "en-GB", {
    year: full ? "numeric" : undefined, month: "2-digit", day: "2-digit",
    hour: "2-digit", minute: "2-digit", second: full ? "2-digit" : undefined, hour12: false
  }).format(date);
}

function relativeAge(seconds) {
  if (!Number.isFinite(seconds) || seconds < 0) return t("timeUnknown");
  if (seconds < 60) return t("justNow");
  return t("ago", { value: formatDuration(seconds) });
}

function snapshotLabel(value) { return value ? String(value).slice(0, 12) : "—"; }

function generationLabel(value) {
  if (!value) return "—";
  return formatDateTime(value, true) === "—" ? snapshotLabel(value) : formatDateTime(value, true);
}

function taskHint(task) {
  if (task.active && task.phase) return task.phase;
  if (task.service?.exit_status === 75) return t("lockSkipped");
  if (task.status === "healthy") return t("lastSuccessOkay");
  if (task.status === "warning") return t("delayed");
  if (task.timer?.load_state && !task.timer.available) return t("timerUnavailable");
  if (task.timer?.available && task.timer.active_state !== "active") return t("timerInactive");
  if (task.service?.result === "failed") return t("taskFailed", { value: task.service.exit_status ?? t("unknown") });
  return t("noSuccess");
}

function renderOverall(data) {
  const status = normalizeStatus(data.overall?.status);
  document.body.dataset.overall = status;
  applyStatusPill(byId("overall-badge"), status);
  setText("overall-title", t(`overallTitle${status[0].toUpperCase()}${status.slice(1)}`));
  setText("overall-summary", t(`overallSummary${status[0].toUpperCase()}${status.slice(1)}`));
  setText("host-name", data.host || t("backupHost"));
  setText("generated-time", t("statusGenerated", { value: formatDateTime(data.generated_at, true) }));

  const display = data.display || {};
  setText("brand-title", display.title || "Restic Backup Dashboard");
  setText("brand-subtitle", display.subtitle || [display.provider_name, display.repository_name].filter(Boolean).join(" · "));
  setText("provider-name", display.provider_name || t("offsiteGeneration"));
  document.title = display.title || "Restic Backup Dashboard";

  const ready = Boolean(data.overall?.recovery_ready);
  setText("recovery-ready", ready ? t("canRestore") : t("needsCheck"));
  setText("recovery-caption", ready ? t("latestAuditPassed") : t("latestAuditUnavailable"));

  const caughtUp = data.sync?.caught_up;
  setText("sync-generation", caughtUp === true ? t("caughtUp") : caughtUp === false ? t("newerGeneration") : t("unknown"));
  setText("sync-caption", caughtUp === true ? t("remoteCurrent") : caughtUp === false ? t("generationLag", { value: formatDuration(data.sync?.lag_seconds) }) : t("waitingMarkers"));
  setText("restore-size", formatBytes(data.recovery?.total_restore_bytes));
  setText("audit-caption", data.recovery?.last_audit ? t("auditedAt", { value: formatDateTime(data.recovery.last_audit) }) : t("noAudit"));
}

function renderPipeline(tasks = []) {
  const fragment = document.createDocumentFragment();
  tasks.forEach((task, index) => {
    const article = makeElement("article", "task-card");
    const top = makeElement("div", "task-topline");
    top.append(makeElement("span", "task-number", String(index + 1)));
    const pill = makeElement("span", "status-pill");
    applyStatusPill(pill, task.status);
    top.append(pill);
    const meta = makeElement("dl", "task-meta");
    const rows = [
      [t("lastSuccess"), task.last_success ? `${formatDateTime(task.last_success)} · ${relativeAge(task.age_seconds)}` : "—"],
      [t("duration"), formatDuration(task.duration_seconds)],
      [t("nextRun"), task.next_run ? formatDateTime(task.next_run) : t("timerDetermines")]
    ];
    rows.forEach(([label, value]) => { const row = makeElement("div"); row.append(makeElement("dt", "", label), makeElement("dd", "", value)); meta.append(row); });
    article.append(top, makeElement("h3", "", task.name), makeElement("p", "task-schedule", task.schedule), makeElement("p", "task-phase", taskHint(task)), meta);
    fragment.append(article);
  });
  byId("pipeline").replaceChildren(fragment);
}

function renderTransfer(data) {
  const taskID = data.sync?.task_id;
  const syncTask = (data.pipeline || []).find((task) => task.id === taskID) || {};
  applyStatusPill(byId("transfer-badge"), syncTask.status || "unknown");
  const progress = data.sync?.progress;
  const caughtUp = data.sync?.caught_up;
  let percent = 0;
  if (syncTask.active && progress) {
    percent = Number(progress.percent) || 0;
    setText("transfer-phase", progress.phase || syncTask.phase || t("uploading"));
    setText("transfer-percent", `${percent}% · ${formatBytes(progress.transferred_bytes)} / ${formatBytes(progress.total_bytes)}`);
    setText("transfer-speed", formatSpeed(progress.speed_bytes_per_second));
    setText("transfer-eta", formatDuration(progress.eta_seconds));
  } else if (caughtUp === true) {
    percent = 100;
    setText("transfer-phase", t("offsiteCurrent")); setText("transfer-percent", t("complete")); setText("transfer-speed", t("idle")); setText("transfer-eta", t("noWait"));
  } else {
    setText("transfer-phase", syncTask.active ? (syncTask.phase || t("syncRunning")) : t("waitingSync"));
    setText("transfer-percent", t("waiting")); setText("transfer-speed", t("idle")); setText("transfer-eta", formatDuration(data.sync?.lag_seconds));
  }
  percent = Math.min(100, Math.max(0, percent));
  byId("transfer-progress").value = percent;
  setText("local-generation", generationLabel(data.sync?.local_generation));
  setText("cloud-generation", generationLabel(data.sync?.uploaded_generation));
}

function renderStorage(storage = {}) {
  setText("storage-used", formatBytes(storage.used_bytes)); setText("storage-quota", `/ ${formatBytes(storage.quota_bytes)}`);
  setText("storage-percent", Number.isFinite(storage.used_percent) ? `${storage.used_percent.toFixed(1)}%` : "—");
  setText("storage-available", formatBytes(storage.available_bytes));
  const percent = Math.min(100, Math.max(0, Number(storage.used_percent) || 0));
  byId("storage-progress").value = percent;
  const pool = storage.pool || {};
  applyStatusPill(byId("pool-badge"), pool.status || storage.status || "unknown", pool.healthy ? t("poolHealthy") : t("needsAttention"));
  const summary = byId("pool-summary"); summary.textContent = pool.summary || t("poolUnavailable"); summary.className = `health-callout status-${normalizeStatus(pool.status)}`;
}

function renderRecovery(recovery = {}) {
  setText("audit-time", recovery.last_audit ? t("sampleDetail", { time: formatDateTime(recovery.last_audit, true), age: relativeAge(recovery.audit_age_seconds), percent: recovery.sample_percent ?? 0 }) : t("noAudit"));
  setText("snapshot-count", Number.isFinite(recovery.remote_snapshot_count) ? recovery.remote_snapshot_count : "—");
  setText("keep-last", recovery.retention?.keep_last ?? "—"); setText("keep-daily", recovery.retention?.keep_daily ?? "—");
  const checks = document.createDocumentFragment();
  (recovery.checks || []).forEach((check) => {
    const item = makeElement("li", `check-item${check.passed ? "" : " is-failed"}`);
    item.append(makeElement("span", "check-mark", check.passed ? "✓" : "!"));
    const copy = makeElement("span"); copy.append(makeElement("span", "", check.name), makeElement("small", "", check.passed ? t("checkPassed") : t("checkMissing"))); item.append(copy); checks.append(item);
  });
  if (!recovery.checks?.length) checks.append(makeElement("li", "empty-state", t("noChecks")));
  byId("recovery-checks").replaceChildren(checks);
  const rows = document.createDocumentFragment();
  (recovery.datasets || []).forEach((dataset) => { const row = document.createElement("tr"); row.append(makeElement("td", "", dataset.name), makeElement("td", "", formatBytes(dataset.verified_restore_bytes)), makeElement("td", "", snapshotLabel(dataset.local_snapshot)), makeElement("td", "", snapshotLabel(dataset.remote_snapshot))); rows.append(row); });
  if (!recovery.datasets?.length) { const row = document.createElement("tr"); const cell = makeElement("td", "empty-state", t("noDatasets")); cell.colSpan = 4; row.append(cell); rows.append(row); }
  byId("dataset-rows").replaceChildren(rows);
}

function renderEvents(events = []) {
  const fragment = document.createDocumentFragment();
  events.slice(0, 12).forEach((event) => { const item = makeElement("li", "event-item"); item.dataset.severity = event.severity || "info"; item.append(makeElement("span", "event-marker")); const copy = makeElement("span", "event-copy"); copy.append(makeElement("strong", "", event.title), makeElement("span", "", [event.source, event.detail].filter(Boolean).join(" · "))); item.append(copy, makeElement("time", "event-time", formatDateTime(event.timestamp))); fragment.append(item); });
  if (!events.length) fragment.append(makeElement("li", "empty-state", t("noEvents")));
  byId("event-list").replaceChildren(fragment);
}

function renderDependencies(data) {
  const dependencies = [...(data.dependencies || [])];
  if (data.storage?.dataset || data.storage?.pool?.status !== "unknown") dependencies.unshift({ id: "storage-pool", name: data.storage?.dataset || t("storageTitle"), status: data.storage?.pool?.status || data.storage?.status, summary: data.storage?.pool?.summary || t("poolUnavailable") });
  const fragment = document.createDocumentFragment();
  dependencies.forEach((dependency) => { const item = makeElement("li", "dependency-item"); const copy = makeElement("span", "dependency-name"); copy.append(makeElement("strong", "", dependency.name), makeElement("span", "", dependency.summary)); const status = normalizeStatus(dependency.status); item.append(copy, makeElement("span", `dependency-state status-${status}`, statusLabel(status))); fragment.append(item); });
  byId("dependency-list").replaceChildren(fragment);
}

function showNotice(message, kind = "warning") { const notice = byId("connection-notice"); notice.hidden = !message; notice.className = `notice${kind === "error" ? " notice-error" : ""}`; notice.textContent = message || ""; }

function render(data) {
  lastData = data;
  renderOverall(data); renderPipeline(data.pipeline || []); renderTransfer(data); renderStorage(data.storage || {}); renderRecovery(data.recovery || {}); renderEvents(data.events || []); renderDependencies(data);
  const generated = new Date(data.generated_at).getTime();
  const age = Number.isNaN(generated) ? Infinity : Math.max(0, (Date.now() - generated) / 1000);
  if (demoMode) showNotice(t("demoNotice")); else if (age > STALE_AFTER_SECONDS) showNotice(t("staleNotice", { value: formatDuration(age) })); else showNotice("");
}

async function loadStatus() {
  const button = byId("refresh-button"); button.classList.add("is-refreshing"); button.disabled = true;
  try {
    const source = demoMode ? "status.example.json" : "/api/v1/status";
    const response = await fetch(`${source}?t=${Date.now()}`, { cache: "no-store" });
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    const data = await response.json();
    if (data.schema_version !== 1) throw new Error(t("unsupportedSchema"));
    render(data);
  } catch (error) {
    showNotice(t("fetchFailed", { value: error.message }), "error"); applyStatusPill(byId("overall-badge"), "error", t("connectionFailed"));
    setText("overall-title", t("panelUnavailable")); setText("overall-summary", t("panelUnavailableDetail"));
  } finally { button.classList.remove("is-refreshing"); button.disabled = false; }
}

byId("refresh-button").addEventListener("click", loadStatus);
byId("language-button").addEventListener("click", () => { language = language === "en" ? "zh" : "en"; try { window.localStorage.setItem("dashboard-language", language); } catch (_) { /* Optional preference only. */ } applyLanguage(); });
applyLanguage(); loadStatus(); window.setInterval(loadStatus, REFRESH_INTERVAL_MS);
