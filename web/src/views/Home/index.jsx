import { useEffect, useState } from 'react';
import { useSelector } from 'react-redux';
import { Box, CircularProgress } from '@mui/material';
import { useTranslation } from 'react-i18next';
import { API } from 'utils/api';
import { showError } from 'utils/common';
import ContentViewer from 'ui-component/ContentViewer';
import { DEFAULT_HOME_STYLE, HOME_TEMPLATES, normalizeHomeStyle } from './templates';

// ==============================|| 站点首页 ||============================== //
// 首页方案由后台「系统设置 → 首页设置」决定（options 表中的 HomePageStyle），
// 取值通过 /api/status 的 home_style 下发，改完刷新即生效，无需重新构建前端。
//
//   brand   -> 内置品牌首页
//   minimal -> 内置极简首页
//   custom  -> 后台「首页内容」（Markdown / HTML / 外链）；内容为空时回落品牌首页
//
// 未识别的取值（老版本、手改数据库、已删除的模板）统一回落 brand，
// 因此新增模板不需要改后端。

const CONTENT_CACHE_KEY = 'home_page_content';
const STYLE_CACHE_KEY = 'home_style';

const Home = () => {
  const { t } = useTranslation();
  const siteInfo = useSelector((state) => state.siteInfo);

  // 首屏用本地缓存的风格兜底，避免「先渲染默认模板再切到自定义」的闪烁；
  // 首次访问（无缓存）时按默认风格渲染。
  const style = normalizeHomeStyle(siteInfo?.home_style || localStorage.getItem(STYLE_CACHE_KEY) || DEFAULT_HOME_STYLE);

  const [content, setContent] = useState(() => localStorage.getItem(CONTENT_CACHE_KEY) || '');
  const [loaded, setLoaded] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');

  useEffect(() => {
    if (siteInfo?.home_style) {
      localStorage.setItem(STYLE_CACHE_KEY, siteInfo.home_style);
    }
  }, [siteInfo?.home_style]);

  useEffect(() => {
    if (style !== 'custom') {
      return;
    }

    let mounted = true;
    setLoaded(false);
    setErrorMessage('');

    API.get('/api/home_page_content')
      .then((res) => {
        if (!mounted) return;
        const { success, message, data } = res.data;
        if (success) {
          setContent(data || '');
          localStorage.setItem(CONTENT_CACHE_KEY, data || '');
        } else {
          showError(message);
          setErrorMessage(t('home.loadingErr'));
        }
      })
      .catch(() => {
        if (mounted) setErrorMessage(t('home.loadingErr'));
      })
      .finally(() => {
        if (mounted) setLoaded(true);
      });

    return () => {
      mounted = false;
    };
  }, [style, t]);

  // 内置模板
  if (style !== 'custom') {
    const Template = HOME_TEMPLATES[style].component;
    return <Template />;
  }

  // 自定义内容：还没拉到且没有缓存时给个占位，避免闪一下空白
  if (!loaded && !content) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: 'calc(100vh - 136px)' }}>
        <CircularProgress />
      </Box>
    );
  }

  // 内容为空（或只有空白字符）→ 回落品牌首页，访客看不到任何"未配置"痕迹
  if (!content.trim()) {
    const Fallback = HOME_TEMPLATES[DEFAULT_HOME_STYLE].component;
    return <Fallback />;
  }

  return (
    <Box>
      <ContentViewer
        content={content}
        loading={!loaded && !content}
        errorMessage={content.trim() ? '' : errorMessage}
        containerStyle={{ minHeight: 'calc(100vh - 136px)' }}
        contentStyle={{ fontSize: 'larger' }}
      />
    </Box>
  );
};

export default Home;
