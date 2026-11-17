import PropTypes from 'prop-types';
import { marked } from 'marked';
import { useMemo, useState, useEffect, useContext, useCallback, createContext } from 'react';

import { API, LoginCheckAPI } from 'utils/api';
import { sanitizeHtml } from 'utils/sanitize';

// 未登录用户（或接口不可用时的兜底）用本地记录已读，避免换页面重复弹窗。
const READ_KEY = 'notices_read';
const BANNER_KEY = 'notices_banner_closed';

// 首页自动弹窗开关。首页是品牌门面，普通公告只进铃铛收件箱与横幅，
// 不整屏遮挡；标记为「必须确认」的公告例外，任何页面都会弹。
const POPUP_ON_HOME = false;

function loadIdSet(key) {
  try {
    const raw = JSON.parse(localStorage.getItem(key) || '[]');
    return new Set(Array.isArray(raw) ? raw : []);
  } catch {
    return new Set();
  }
}

function saveIdSet(key, set) {
  try {
    localStorage.setItem(key, JSON.stringify([...set]));
  } catch {
    // 隐私模式下 localStorage 可能不可用，忽略即可。
  }
}

export const NoticeContext = createContext(undefined);

export function NoticeProvider({ children }) {
  const [notices, setNotices] = useState([]);
  const [readIds, setReadIds] = useState(() => loadIdSet(READ_KEY));
  const [dismissedBanners, setDismissedBanners] = useState(() => loadIdSet(BANNER_KEY));
  const [inboxOpen, setInboxOpen] = useState(false);
  const [detailId, setDetailId] = useState(null);
  const [queue, setQueue] = useState([]);
  const [queueIndex, setQueueIndex] = useState(0);

  const isRead = useCallback((notice) => !!(notice?.read || readIds.has(notice?.id)), [readIds]);

  const refresh = useCallback(async () => {
    try {
      const res = await LoginCheckAPI.get('/api/notices');
      const { success, data } = res.data;
      if (!success) return [];
      const list = Array.isArray(data) ? data : [];
      setNotices(list);
      return list;
    } catch (error) {
      console.error('Failed to fetch notices:', error);
      return [];
    }
  }, []);

  // 首次加载：拉取公告，并把未读的弹窗型公告排成队列。
  useEffect(() => {
    let cancelled = false;
    (async () => {
      const list = await refresh();
      if (cancelled || list.length === 0) return;

      const localRead = loadIdSet(READ_KEY);
      const onHome = typeof window !== 'undefined' && (window.location.pathname === '/' || window.location.pathname === '');
      const pending = list
        .filter((n) => n.popup && !n.read && !localRead.has(n.id))
        .filter((n) => !(onHome && !POPUP_ON_HOME && !n.require_ack))
        .map((n) => n.id);
      if (pending.length > 0) {
        setQueue(pending);
        setQueueIndex(0);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [refresh]);

  const markRead = useCallback(async (ids, ack = false) => {
    const list = Array.isArray(ids) ? ids : [ids];
    if (list.length === 0) return;

    setReadIds((prev) => {
      const next = new Set(prev);
      list.forEach((id) => next.add(id));
      saveIdSet(READ_KEY, next);
      return next;
    });
    setNotices((prev) => prev.map((n) => (list.includes(n.id) ? { ...n, read: true, acked: ack || n.acked } : n)));

    // 未登录时只写本地记录，避免每看一条公告都发一次会被 401 的请求。
    const loggedIn = typeof localStorage !== 'undefined' && !!localStorage.getItem('user');
    if (!loggedIn) return;

    try {
      // 用不带错误拦截器的实例：这里失败是正常情况（会话过期等），不应弹红条。
      await LoginCheckAPI.post('/api/notices/read', { ids: list, ack });
    } catch (error) {
      // 忽略，本地已记录。
    }
  }, []);

  const dismissBanner = useCallback((id) => {
    setDismissedBanners((prev) => {
      const next = new Set(prev);
      next.add(id);
      saveIdSet(BANNER_KEY, next);
      return next;
    });
  }, []);

  const closePopup = useCallback(async () => {
    const current = notices.find((n) => n.id === queue[queueIndex]);
    if (current) {
      await markRead(current.id, current.require_ack);
    }
    if (queueIndex + 1 < queue.length) {
      setQueueIndex(queueIndex + 1);
    } else {
      setQueue([]);
      setQueueIndex(0);
    }
  }, [notices, queue, queueIndex, markRead]);

  // 关闭整个弹窗队列（用户点了「查看全部」），剩余公告按已读处理但不记为确认。
  const dismissQueue = useCallback(async () => {
    const ids = queue.slice(queueIndex);
    setQueue([]);
    setQueueIndex(0);
    if (ids.length > 0) {
      await markRead(ids, false);
    }
  }, [queue, queueIndex, markRead]);

  const openInbox = useCallback(() => {
    setInboxOpen(true);
    setDetailId(null);
    refresh();
  }, [refresh]);

  const closeInbox = useCallback(() => {
    setInboxOpen(false);
    setDetailId(null);
  }, []);

  const value = useMemo(() => {
    const unread = notices.filter((n) => !isRead(n));
    const banners = notices.filter((n) => n.show_banner && !dismissedBanners.has(n.id));
    const current = queue.length > 0 ? notices.find((n) => n.id === queue[queueIndex]) || null : null;

    return {
      notices,
      unreadCount: unread.length,
      banners,
      current,
      queueTotal: queue.length,
      queueIndex,
      isRead,
      markRead,
      dismissBanner,
      closePopup,
      dismissQueue,
      openInbox,
      closeInbox,
      inboxOpen,
      detailId,
      setDetailId,
      refresh,
      renderMarkdown: (content) => sanitizeHtml(marked(content || '').toString())
    };
  }, [
    notices,
    isRead,
    dismissedBanners,
    queue,
    queueIndex,
    markRead,
    dismissBanner,
    closePopup,
    dismissQueue,
    openInbox,
    closeInbox,
    inboxOpen,
    detailId,
    refresh
  ]);

  return <NoticeContext.Provider value={value}>{children}</NoticeContext.Provider>;
}

NoticeProvider.propTypes = {
  children: PropTypes.node
};

export function useNotice() {
  const context = useContext(NoticeContext);
  if (context === undefined) {
    throw new Error('useNotice must be used within a NoticeProvider');
  }
  return context;
}
