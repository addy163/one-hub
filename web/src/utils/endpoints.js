// 多线路地址清单：由后端 /api/status 按用户分组下发，未配置时回落到 server_address。
const STORAGE_KEY = 'relay_endpoint_id';
const CHANGE_EVENT = 'relay-endpoint-changed';

// 允许后端返回数组或 JSON 字符串，两种都能解析
function normalizeList(raw) {
  if (Array.isArray(raw)) return raw;
  if (typeof raw === 'string' && raw.trim() !== '') {
    try {
      const parsed = JSON.parse(raw);
      return Array.isArray(parsed) ? parsed : [];
    } catch (e) {
      return [];
    }
  }
  return [];
}

export function getRelayEndpoints(siteInfo) {
  const list = normalizeList(siteInfo?.relay_endpoints).filter((item) => item && item.url && item.enabled !== false);

  if (list.length === 0 && siteInfo?.server_address) {
    return [{ id: 'default', name: '默认线路', url: siteInfo.server_address, enabled: true }];
  }
  return list;
}

export function getSelectedEndpoint(siteInfo) {
  const list = getRelayEndpoints(siteInfo);
  if (list.length === 0) return null;

  let savedId = null;
  try {
    savedId = localStorage.getItem(STORAGE_KEY);
  } catch (e) {
    savedId = null;
  }
  return list.find((item) => item.id === savedId) || list[0];
}

export function selectEndpoint(endpoint) {
  if (!endpoint?.id) return;
  try {
    localStorage.setItem(STORAGE_KEY, endpoint.id);
  } catch (e) {
    // 隐私模式下 localStorage 不可写，忽略即可
  }
  window.dispatchEvent(new CustomEvent(CHANGE_EVENT, { detail: endpoint }));
}

export function subscribeEndpointChange(handler) {
  const listener = (event) => handler(event.detail);
  window.addEventListener(CHANGE_EVENT, listener);
  return () => window.removeEventListener(CHANGE_EVENT, listener);
}
