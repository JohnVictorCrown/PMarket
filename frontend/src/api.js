const BASE = 'https://pmarket-vgtl.onrender.com';

export async function fetchJSON(path) {
  const r = await fetch(`${BASE}${path}`);
  if (!r.ok) throw new Error(`${path} ${r.status}`);
  return r.json();
}

export function keepalive(interval = 300000) {
  setInterval(() => fetch(`${BASE}/health`).catch(() => {}), interval);
}
