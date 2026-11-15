import React, { useEffect, useState } from 'react';
import { showError } from 'utils/common';
import { API } from 'utils/api';
import BaseIndex from './baseIndex';
import { Box } from '@mui/material';
import { useTranslation } from 'react-i18next';
import ContentViewer from 'ui-component/ContentViewer';

// 首页模式开关：
// true  -> 使用代码内置的优智AI（UUAI）品牌首页（views/Home/baseIndex.jsx）
// false -> 使用后台「系统设置 → 自定义首页内容」里的 HTML / Markdown（原有行为）
// 切换后需重新构建前端；无需改动数据库或后台配置。
const FORCE_NEW_HOME = true;

const Home = () => {
  const { t } = useTranslation();
  const [homePageContentLoaded, setHomePageContentLoaded] = useState(false);
  const [homePageContent, setHomePageContent] = useState('');

  const displayHomePageContent = async () => {
    setHomePageContent(localStorage.getItem('home_page_content') || '');
    try {
      const res = await API.get('/api/home_page_content');
      const { success, message, data } = res.data;
      if (success) {
        setHomePageContent(data);
        localStorage.setItem('home_page_content', data);
      } else {
        showError(message);
        setHomePageContent(t('home.loadingErr'));
      }
      setHomePageContentLoaded(true);
    } catch (error) {
      return;
    }
  };

  useEffect(() => {
    if (!FORCE_NEW_HOME) {
      displayHomePageContent().then();
    }
  }, []);

  // 内置品牌首页：跳过自定义内容请求，避免后台遗留的旧 HTML 覆盖新版设计
  if (FORCE_NEW_HOME) {
    return <BaseIndex />;
  }

  return (
    <>
      {homePageContentLoaded && homePageContent === '' ? (
        <BaseIndex />
      ) : (
        <Box>
          <ContentViewer
            content={homePageContent}
            loading={!homePageContentLoaded}
            errorMessage={homePageContent === t('home.loadingErr') ? t('home.loadingErr') : ''}
            containerStyle={{ minHeight: 'calc(100vh - 136px)' }}
            contentStyle={{ fontSize: 'larger' }}
          />
        </Box>
      )}
    </>
  );
};

export default Home;
