import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import { resources } from './resources';
import LanguageDetector from 'i18next-browser-languagedetector';

// 用用户上次选择的语言初始化，避免刷新页面时先闪一下默认中文。
// 未选择过时保持原有行为，之后由 StatusContext 依据站点 /api/status.language 决定。
const storedLanguage = typeof localStorage !== 'undefined' ? localStorage.getItem('appLanguage') : null;

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources,
    fallbackLng: 'zh_CN',
    debug: false,
    lng: storedLanguage || 'zh_CN',
    interpolation: {
      escapeValue: false
    }
  });

export default i18n;
