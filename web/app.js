"use strict";

const HISTORY_LIMIT = 60;
const history = [];
const RECONNECTION_TIME_OUT = 2000;
const GAUGES = new Set(["goroutines", "heap_mb"]);
const CHART_PAD = {left: 44, right: 14, top: 12, bottom: 26};
const Y_STEPS = 5;
const X_LABEL_EVERY = 15;
const DOT_RADIUS = 2.5;
const Y_FROM_ZERO = false;
const UNITS = {heap_mb: " MB"};
const CARD_PREFIX = "card-";

function connect() {
    const scheme = location.protocol === "https:" ? "wss:" : "ws:";
    const url = `${scheme}//${location.host}/ws`;
    const soc = new WebSocket(url);

    soc.onopen = () => setStatus(true);
    soc.onmessage = (event) => {
        handleSnapshot(JSON.parse(event.data));
    };
    soc.onclose = () => {
        setStatus(false);
        setTimeout(connect, RECONNECTION_TIME_OUT);
    };
}

function setStatus(connected) {
    const status = document.getElementById("status");
    status.textContent = connected ? "ONLINE" : "TRANSMISSION ENDED";
    status.style.color = connected ? "green" : "red";
}

function handleSnapshot(snapshot) {
    history.push(snapshot);
    if (history.length > HISTORY_LIMIT) {
        history.shift();
    }
    renderCards(snapshot.metrics);
    renderChart();
    const total = snapshot.metrics?.wiki_edit?.perSecond ?? 0;
    document.getElementById("chart-title").textContent =
        `All wikis: ${total.toFixed(1)} edits/s`;
}

function renderCards(metrics) {
    const list = document.getElementById("cards");
    const current = metrics ?? {};

    for (const [name, stats] of Object.entries(current)) {
        if (name === "wiki_edit") continue;

        const id = CARD_PREFIX + name;
        let card = document.getElementById(id);
        if (!card) {
            card = document.createElement("div");
            card.id = id;
            card.className = "card";

            const nameEl = document.createElement("div");
            nameEl.className = "card-name";
            nameEl.textContent = name;

            const valueEl = document.createElement("div");
            valueEl.className = "card-value";

            const detailEl = document.createElement("div");
            detailEl.className = "card-detail";

            card.append(nameEl, valueEl, detailEl);
            list.append(card);
        }

        const valueEl = card.querySelector(".card-value");
        const detailEl = card.querySelector(".card-detail");

        if (GAUGES.has(name)) {
            valueEl.textContent = stats.last.toFixed(1) + (UNITS[name] ?? "");
            detailEl.textContent = `min ${stats.min.toFixed(1)} · max ${stats.max.toFixed(1)}`;
        } else {
            valueEl.textContent = `${stats.perSecond.toFixed(1)}/s`;
            detailEl.textContent = `${stats.count} in last minute`;
        }
    }

    for (const card of list.querySelectorAll(".card")) {
        const name = card.id.slice(CARD_PREFIX.length);
        if (!(name in current)) {
            card.remove();
        }
    }
}

function stepSize(range, steps) {
    const raw = range / steps;
    const p = 10 ** Math.floor(Math.log10(raw));
    for (const m of [1, 2, 2.5, 5, 10]) {
        if (raw <= m * p) return m * p;
    }
    return 10 * p;
}

function cssVar(name, fallback) {
    return getComputedStyle(document.documentElement).getPropertyValue(name).trim() || fallback;
}

function renderChart() {
    const canvas = document.getElementById("chart");
    const ctx = canvas.getContext("2d");

    const dpr = window.devicePixelRatio || 1;
    const rect = canvas.getBoundingClientRect();
    const w = rect.width, h = rect.height;
    if (canvas.width !== Math.round(w * dpr) || canvas.height !== Math.round(h * dpr)) {
        canvas.width = Math.round(w * dpr);
        canvas.height = Math.round(h * dpr);
    }
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, w, h);
    if (history.length === 0) return;

    const plotW = w - CHART_PAD.left - CHART_PAD.right;
    const plotH = h - CHART_PAD.top - CHART_PAD.bottom;
    const values = history.map(s => s.metrics?.wiki_edit?.perSecond ?? 0);

    let lo = Y_FROM_ZERO ? 0 : Math.min(...values);
    let hi = Math.max(...values);
    if (hi - lo < 1) hi = lo + 1;
    const step = stepSize(hi - lo, Y_STEPS);
    lo = Math.floor(lo / step) * step;
    hi = Math.ceil(hi / step) * step;

    const offset = HISTORY_LIMIT - values.length;
    const xAt = i => CHART_PAD.left + ((offset + i) / (HISTORY_LIMIT - 1)) * plotW;
    const yAt = v => CHART_PAD.top + plotH - ((v - lo) / (hi - lo)) * plotH;

    const muted = cssVar("--muted", "#6b7385");
    const grid = cssVar("--border", "#dde1e8");
    const line = "steelblue";

    ctx.font = "12px system-ui, sans-serif";
    ctx.lineWidth = 1;

    // Y axis
    ctx.strokeStyle = grid;
    ctx.fillStyle = muted;
    ctx.textAlign = "right";
    ctx.textBaseline = "middle";
    for (let v = lo; v <= hi + step / 2; v += step) {
        const y = yAt(v);
        ctx.beginPath();
        ctx.moveTo(CHART_PAD.left, y);
        ctx.lineTo(w - CHART_PAD.right, y);
        ctx.stroke();
        ctx.fillText(String(Number(v.toFixed(2))), CHART_PAD.left - 8, y);
    }

    // X axis
    ctx.textBaseline = "top";
    for (let i = values.length - 1; i >= 0; i -= X_LABEL_EVERY) {
        ctx.textAlign = i === values.length - 1 ? "right" : "center";
        const time = new Date(history[i].time).toLocaleTimeString();
        ctx.fillText(time, xAt(i), h - CHART_PAD.bottom + 8);
    }

    ctx.strokeStyle = line;
    ctx.lineWidth = 2;
    ctx.beginPath();
    values.forEach((v, i) => {
        if (i === 0) ctx.moveTo(xAt(i), yAt(v));
        else ctx.lineTo(xAt(i), yAt(v));
    });
    ctx.stroke();

    ctx.fillStyle = line;
    values.forEach((v, i) => {
        ctx.beginPath();
        ctx.arc(xAt(i), yAt(v), DOT_RADIUS, 0, Math.PI * 2);
        ctx.fill();
    });
}

connect();