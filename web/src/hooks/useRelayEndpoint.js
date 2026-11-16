import { useEffect, useState } from 'react';
import { useSelector } from 'react-redux';
import { getRelayEndpoints, getSelectedEndpoint, selectEndpoint, subscribeEndpointChange } from 'utils/endpoints';

// 统一读取「当前选中的线路」，并让多个页面之间保持同步。
export default function useRelayEndpoint() {
  const siteInfo = useSelector((state) => state.siteInfo);
  const endpoints = getRelayEndpoints(siteInfo);
  const [endpoint, setEndpoint] = useState(() => getSelectedEndpoint(siteInfo));

  useEffect(() => {
    setEndpoint(getSelectedEndpoint(siteInfo));
    return subscribeEndpointChange(() => setEndpoint(getSelectedEndpoint(siteInfo)));
  }, [siteInfo]);

  const choose = (item) => {
    selectEndpoint(item);
    setEndpoint(item);
  };

  const url = endpoint?.url || siteInfo?.server_address || '';

  return { endpoints, endpoint, url, choose };
}
