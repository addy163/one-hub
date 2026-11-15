import PropTypes from 'prop-types';
import { marked } from 'marked';
import { useMemo, useState, useEffect, useContext, useCallback, createContext } from 'react';

import { API } from 'utils/api';
import { sanitizeHtml } from 'utils/sanitize';

// 首页自动弹公告开关。
// false：访问首页时不自动弹出公告弹窗，也不把公告标记为已读；
//        导航栏的公告按钮仍可随时查看，用户进入控制台等其他页面时仍会正常弹出一次。
// true ：保持原有行为（公告内容变更时，任何页面首次访问都弹窗）。
const AUTO_POPUP_ON_HOME = false;

export const NoticeContext = createContext(undefined);

export function NoticeProvider({ children }) {
  const [notice, setNotice] = useState(null);
  const [isOpen, setOpen] = useState(false);

  const openNotice = () => setOpen(true);
  const closeNotice = () => setOpen(false);

  const checkNotice = useCallback(async () => {
    if (typeof window !== 'undefined') {
      // 确保代码只在客户端执行
      try {
        const noticeMsg = await getNotice();
        const oldNotice = localStorage.getItem('notice');

        const processAndSetNotice = async (content) => {
          // 公告内容来自后台且会被 dangerouslySetInnerHTML 渲染；
          // localStorage 里缓存的旧公告同样会走这里，必须统一消毒。
          const htmlNotice = sanitizeHtml(marked(content || '').toString());

          setNotice(htmlNotice);
          if (content !== oldNotice) {
            // 首页只静默缓存公告内容供铃铛查看，不弹窗、不标记已读，
            // 避免整屏公告遮挡品牌首页；其他页面行为不变。
            const onHome = window.location.pathname === '/' || window.location.pathname === '';
            if (onHome && !AUTO_POPUP_ON_HOME) {
              return;
            }
            setOpen(true);
            localStorage.setItem('notice', content || '');
          }
        };

        if (noticeMsg !== oldNotice && noticeMsg) {
          await processAndSetNotice(noticeMsg);
        } else if (oldNotice) {
          await processAndSetNotice(oldNotice);
        }
      } catch (error) {
        console.error('Failed to fetch notice:', error);
      }
    }
  }, []);

  useEffect(() => {
    checkNotice();
  }, [checkNotice]);

  const memoizedValue = useMemo(
    () => ({
      notice,
      isOpen,
      openNotice,
      closeNotice
    }),
    [notice, isOpen]
  );

  return <NoticeContext.Provider value={memoizedValue}>{children}</NoticeContext.Provider>;
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

async function getNotice() {
  try {
    const res = await API.get('/api/notice');
    const { success, data } = res.data;

    if (!success) {
      return null;
    }

    return data;
  } catch (error) {
    console.error('Error fetching notice:', error);

    return null;
  }
}
