import BrandHome from './baseIndex';
import MinimalHome from './minimalIndex';

// ==============================|| 首页模板注册表 ||============================== //
// 新增一套内置首页：在 views/Home 下写一个组件，然后在这里加一行即可。
// 后端只透传 home_style 字符串，不做枚举校验，所以新增模板不需要改后端、发版本。
// custom 是特殊值：它不是模板，而是「使用后台自定义内容」，内容为空时回落 brand。

export const DEFAULT_HOME_STYLE = 'brand';

export const HOME_TEMPLATES = {
  brand: {
    value: 'brand',
    component: BrandHome,
    accent: '#5B7CFA',
    nameKey: 'setting_index.homeSetting.styleBrandName',
    descKey: 'setting_index.homeSetting.styleBrandDesc'
  },
  minimal: {
    value: 'minimal',
    component: MinimalHome,
    accent: '#0F6E56',
    nameKey: 'setting_index.homeSetting.styleMinimalName',
    descKey: 'setting_index.homeSetting.styleMinimalDesc'
  },
  custom: {
    value: 'custom',
    component: null,
    accent: '#888780',
    nameKey: 'setting_index.homeSetting.styleCustomName',
    descKey: 'setting_index.homeSetting.styleCustomDesc'
  }
};

export const HOME_STYLE_OPTIONS = ['brand', 'minimal', 'custom'];

// 非法值（老版本、手改数据库、以后删掉的模板）统一回落默认风格。
export const normalizeHomeStyle = (style) => (HOME_TEMPLATES[style] ? style : DEFAULT_HOME_STYLE);
