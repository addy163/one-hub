import { useEffect, useContext, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Divider,
  Radio,
  Stack,
  TextField,
  Typography,
  useTheme
} from '@mui/material';
import { LaunchOutlined } from '@mui/icons-material';
import SubCard from 'ui-component/cards/SubCard';
import { API } from 'utils/api';
import { showError, showSuccess } from 'utils/common';
import { LoadStatusContext } from 'contexts/StatusContext';
import { HOME_STYLE_OPTIONS, HOME_TEMPLATES, normalizeHomeStyle } from 'views/Home/templates';

// ==============================|| 首页设置 ||============================== //
// 首页 = 风格（HomePageStyle） × 内容来源。风格存 options 表，前端按 style 查模板注册表，
// 所以切换风格是即时的，不需要重新构建前端。

const STYLE_PREVIEWS = {
  brand: { bg: '#0C1230', bar: 'rgba(255,255,255,.55)', sub: 'rgba(255,255,255,.22)', card: 'rgba(91,124,250,.55)' },
  minimal: { bg: 'transparent', bar: 'rgba(15,23,42,.65)', sub: 'rgba(15,23,42,.2)', card: 'rgba(91,124,250,.35)' },
  custom: { bg: 'transparent', bar: 'rgba(15,23,42,.35)', sub: 'rgba(15,23,42,.15)', card: 'rgba(136,135,128,.2)' }
};

const StylePreview = ({ variant, active }) => {
  const p = STYLE_PREVIEWS[variant] || STYLE_PREVIEWS.brand;

  if (variant === 'custom') {
    return (
      <Box
        sx={{
          height: 56,
          borderRadius: '8px',
          border: `1px dashed ${active ? 'rgba(91,124,250,.5)' : 'rgba(136,135,128,.45)'}`,
          display: 'grid',
          placeItems: 'center',
          bgcolor: 'transparent'
        }}
      >
        <Typography sx={{ fontSize: '0.68rem', letterSpacing: '0.12em', color: 'text.disabled', fontFamily: 'monospace' }}>
          MARKDOWN / HTML
        </Typography>
      </Box>
    );
  }

  return (
    <Box
      sx={{
        position: 'relative',
        height: 56,
        borderRadius: '8px',
        overflow: 'hidden',
        bgcolor: p.bg,
        border: variant === 'brand' ? 'none' : '1px solid',
        borderColor: 'divider',
        p: 1.2,
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'center',
        gap: 0.6
      }}
    >
      <Box sx={{ width: '58%', height: 6, borderRadius: '3px', bgcolor: p.bar }} />
      <Box sx={{ width: '40%', height: 5, borderRadius: '3px', bgcolor: p.sub }} />
      <Box sx={{ width: 34, height: 9, borderRadius: '4px', bgcolor: p.card, mt: 0.4 }} />
    </Box>
  );
};

const HomeSetting = () => {
  const { t } = useTranslation();
  const theme = useTheme();
  const loadStatus = useContext(LoadStatusContext);

  const [loaded, setLoaded] = useState(false);
  const [savingStyle, setSavingStyle] = useState(false);
  const [savingContent, setSavingContent] = useState(false);
  const [style, setStyle] = useState('brand');
  const [content, setContent] = useState('');
  const [savedContent, setSavedContent] = useState('');

  const loadOptions = async () => {
    try {
      const res = await API.get('/api/option/');
      const { success, message, data } = res.data;
      if (success) {
        const map = {};
        (data || []).forEach((item) => {
          map[item.key] = item.value;
        });
        setStyle(normalizeHomeStyle(map.HomePageStyle));
        setContent(map.HomePageContent || '');
        setSavedContent(map.HomePageContent || '');
      } else {
        showError(message);
      }
    } catch (error) {
      // 静默失败，表单保持默认值
    } finally {
      setLoaded(true);
    }
  };

  useEffect(() => {
    loadOptions().then();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const updateOption = async (key, value) => {
    const res = await API.put('/api/option/', { key, value });
    const { success, message } = res.data;
    if (!success) {
      showError(message);
      return false;
    }
    if (loadStatus) await loadStatus();
    return true;
  };

  const handleSelectStyle = async (next) => {
    if (next === style || savingStyle) return;
    setSavingStyle(true);
    const previous = style;
    setStyle(next);
    const ok = await updateOption('HomePageStyle', next);
    if (!ok) setStyle(previous);
    else showSuccess(t('setting_index.homeSetting.styleSaved'));
    setSavingStyle(false);
  };

  const handleSaveContent = async () => {
    setSavingContent(true);
    const ok = await updateOption('HomePageContent', content);
    if (ok) {
      setSavedContent(content);
      showSuccess(t('setting_index.homeSetting.contentSaved'));
    }
    setSavingContent(false);
  };

  const savedLength = useMemo(() => (savedContent || '').trim().length, [savedContent]);
  const draftLength = useMemo(() => (content || '').trim().length, [content]);
  const dirty = content !== savedContent;

  const previewUrl = `${window.location.origin}/`;

  return (
    <SubCard title={t('setting_index.homeSetting.title')}>
      <Stack spacing={2.5}>
        <Typography variant="body2" sx={{ color: 'text.secondary', lineHeight: 1.8 }}>
          {t('setting_index.homeSetting.desc')}
        </Typography>

        <Box
          sx={{
            display: 'grid',
            gridTemplateColumns: { xs: '1fr', sm: 'repeat(3, 1fr)' },
            gap: 1.5
          }}
        >
          {HOME_STYLE_OPTIONS.map((value) => {
            const meta = HOME_TEMPLATES[value];
            const active = style === value;
            return (
              <Box
                key={value}
                role="radio"
                aria-checked={active}
                tabIndex={0}
                onClick={() => handleSelectStyle(value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    handleSelectStyle(value);
                  }
                }}
                sx={{
                  position: 'relative',
                  p: 1.5,
                  borderRadius: '10px',
                  cursor: savingStyle ? 'progress' : 'pointer',
                  bgcolor: active ? (theme.palette.mode === 'dark' ? 'rgba(91,124,250,.12)' : 'rgba(91,124,250,.06)') : 'transparent',
                  border: '1px solid',
                  borderColor: active ? 'primary.main' : 'divider',
                  transition: 'all .18s ease',
                  '&:hover': { borderColor: active ? 'primary.main' : 'text.disabled' },
                  '&:focus-visible': { outline: '2px solid', outlineColor: 'primary.main', outlineOffset: 2 }
                }}
              >
                <StylePreview variant={value} active={active} />
                <Stack direction="row" alignItems="center" spacing={0.4} sx={{ mt: 1.2 }}>
                  <Radio checked={active} size="small" sx={{ p: 0.4, ml: -0.4 }} tabIndex={-1} />
                  <Typography sx={{ fontSize: '0.88rem', fontWeight: 600 }}>{t(meta.nameKey)}</Typography>
                </Stack>
                <Typography sx={{ mt: 0.4, fontSize: '0.75rem', lineHeight: 1.7, color: 'text.secondary' }}>
                  {t(meta.descKey)}
                </Typography>
              </Box>
            );
          })}
        </Box>

        <Stack direction="row" spacing={1} alignItems="center" sx={{ flexWrap: 'wrap' }}>
          <Button
            variant="outlined"
            size="small"
            endIcon={<LaunchOutlined sx={{ fontSize: 16 }} />}
            href={previewUrl}
            target="_blank"
            rel="noopener noreferrer"
          >
            {t('setting_index.homeSetting.preview')}
          </Button>
          {savingStyle && <CircularProgress size={16} />}
          <Typography sx={{ fontSize: '0.75rem', color: 'text.secondary' }}>{t('setting_index.homeSetting.previewHint')}</Typography>
        </Stack>

        {style !== 'custom' && savedLength > 0 && (
          <Alert
            severity="info"
            action={
              <Button color="inherit" size="small" onClick={() => handleSelectStyle('custom')}>
                {t('setting_index.homeSetting.enableLegacy')}
              </Button>
            }
          >
            {t('setting_index.homeSetting.legacyHint', { count: savedLength })}
          </Alert>
        )}

        {style === 'custom' && (
          <>
            <Divider />
            {draftLength === 0 && <Alert severity="warning">{t('setting_index.homeSetting.emptyHint')}</Alert>}
            <TextField
              multiline
              fullWidth
              minRows={10}
              maxRows={22}
              id="HomePageContent"
              label={t('setting_index.homeSetting.contentLabel')}
              value={content}
              onChange={(e) => setContent(e.target.value)}
              placeholder={t('setting_index.homeSetting.contentPlaceholder')}
              helperText={t('setting_index.homeSetting.contentHelper', { count: draftLength })}
            />
            <Stack direction="row" spacing={1.5} alignItems="center">
              <Button variant="contained" onClick={handleSaveContent} disabled={savingContent || !dirty}>
                {t('setting_index.homeSetting.saveContent')}
              </Button>
              {dirty && <Typography sx={{ fontSize: '0.75rem', color: 'warning.main' }}>{t('setting_index.homeSetting.unsaved')}</Typography>}
            </Stack>
          </>
        )}

        {!loaded && (
          <Stack direction="row" spacing={1} alignItems="center">
            <CircularProgress size={14} />
            <Typography sx={{ fontSize: '0.75rem', color: 'text.secondary' }}>{t('setting_index.homeSetting.loading')}</Typography>
          </Stack>
        )}
      </Stack>
    </SubCard>
  );
};

export default HomeSetting;
