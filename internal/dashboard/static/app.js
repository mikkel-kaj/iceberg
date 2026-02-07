async function fetchJSON(path) {
  const resp = await fetch(path);
  if (!resp.ok) {
    throw new Error(`${path}: ${resp.status}`);
  }
  return resp.json();
}

function statusClass(status) {
  return `status-${status || "stopped"}`;
}

function formatGB(used, total) {
  return `${used}GB / ${total}GB`;
}

function renderServer(metrics) {
  const el = document.getElementById("server");
  if (!metrics) {
    el.textContent = "No metrics available.";
    return;
  }
  el.innerHTML = `
    <h2>Server</h2>
    <p>CPU: ${metrics.cpu_percent.toFixed(1)}%</p>
    <p>RAM: ${metrics.memory_used_mb}MB / ${metrics.memory_total_mb}MB</p>
    <p>Disk: ${formatGB(metrics.disk_used_gb, metrics.disk_total_gb)}</p>
  `;
}

function renderServices(services) {
  const tbody = document.querySelector("#services-table tbody");
  tbody.innerHTML = "";
  if (!services || services.length === 0) {
    const row = document.createElement("tr");
    row.innerHTML = `<td colspan="5">No services deployed.</td>`;
    tbody.appendChild(row);
    return;
  }

  for (const svc of services) {
    const row = document.createElement("tr");
    const domain = svc.domain ? `<a href="https://${svc.domain}" target="_blank" rel="noreferrer">${svc.domain}</a>` : "-";
    row.innerHTML = `
      <td><a href="/dashboard/service.html?name=${encodeURIComponent(svc.name)}">${svc.name}</a></td>
      <td><span class="status-dot ${statusClass(svc.status)}"></span>${svc.status}</td>
      <td>${svc.memory_used_mb || 0} MB</td>
      <td>${domain}</td>
      <td>${svc.checked_at || "-"}</td>
    `;
    tbody.appendChild(row);
  }
}

async function refresh() {
  try {
    const status = await fetchJSON("/status");
    renderServer(status.metrics);
    renderServices(status.services || []);
  } catch (err) {
    console.error(err);
  }
}

setInterval(refresh, 10000);
refresh();
