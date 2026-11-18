import { Box, Button, Chip, Container, Divider, Stack, Typography, useTheme } from '@mui/material';
import { useSelector } from 'react-redux';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import { CheckOutlined, ArrowForwardOutlined } from '@mui/icons-material';

// ==============================|| 极简首页 ||============================== //
// 面向「只想快速上线、不想维护营销页」的部署：纯文字、单屏、无外部图片依赖。
// 文案全部复用 homePage.* 的 i18n key，改语言不用改组件。

const MinimalHome = () => {
  const theme = useTheme();
  const { t } = useTranslation();
  const account = useSelector((state) => state.account);
  const siteInfo = useSelector((state) => state.siteInfo);

  const isDark = theme.palette.mode === 'dark';
  const brandName = siteInfo?.system_name || 'UUAI';
  const loggedIn = !!account?.user;
  const primaryTo = loggedIn ? '/panel' : '/login';

  return (
    <Box
      sx={{
        minHeight: 'calc(100vh - 136px)',
        display: 'flex',
        alignItems: 'center',
        py: { xs: 8, md: 12 }
      }}
    >
      <Container maxWidth="sm">
        <Stack alignItems="center" sx={{ textAlign: 'center' }}>
          <Chip
            label={brandName}
            sx={{
              mb: 3.5,
              height: 30,
              px: 1,
              fontSize: '0.78rem',
              fontWeight: 500,
              bgcolor: isDark ? 'rgba(255,255,255,.08)' : 'rgba(91,124,250,.08)',
              color: isDark ? 'rgba(241,245,249,.85)' : '#3D5BE0',
              border: `1px solid ${isDark ? 'rgba(255,255,255,.14)' : 'rgba(91,124,250,.18)'}`
            }}
          />

          <Typography
            component="h1"
            sx={{
              fontSize: { xs: '1.9rem', sm: '2.4rem' },
              fontWeight: 700,
              lineHeight: 1.35,
              letterSpacing: '-0.025em',
              color: 'text.primary'
            }}
          >
            {t('homePage.hero.title1')}
            <Box component="span" sx={{ color: isDark ? '#8FA6FF' : '#4C6BF5' }}>
              {t('homePage.hero.title2')}
            </Box>
          </Typography>

          <Typography
            sx={{
              mt: 2.5,
              maxWidth: 520,
              fontSize: '0.95rem',
              lineHeight: 1.9,
              color: 'text.secondary'
            }}
          >
            {t('homePage.hero.desc', { brand: brandName })}
          </Typography>

          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5} sx={{ mt: 4 }}>
            <Button
              component={Link}
              to={primaryTo}
              variant="contained"
              endIcon={<ArrowForwardOutlined sx={{ fontSize: 17 }} />}
              sx={{ height: 46, px: 3, borderRadius: '10px', textTransform: 'none', fontWeight: 600, boxShadow: 'none' }}
            >
              {loggedIn ? t('homePage.hero.ctaConsole') : t('homePage.hero.ctaStart')}
            </Button>
            <Button
              component={Link}
              to="/price"
              variant="outlined"
              sx={{ height: 46, px: 2.6, borderRadius: '10px', textTransform: 'none', fontWeight: 500 }}
            >
              {t('homePage.hero.ctaPricing')}
            </Button>
          </Stack>

          <Stack
            direction="row"
            sx={{ mt: 4.5, flexWrap: 'wrap', justifyContent: 'center', gap: 1.2, rowGap: 1 }}
          >
            {['tag1', 'tag2', 'tag3', 'tag4'].map((tagKey) => (
              <Stack key={tagKey} direction="row" spacing={0.6} alignItems="center">
                <CheckOutlined sx={{ fontSize: 13, color: isDark ? '#5DCAA5' : '#0F6E56' }} />
                <Typography sx={{ fontSize: '0.78rem', color: 'text.secondary' }}>{t(`homePage.hero.${tagKey}`)}</Typography>
              </Stack>
            ))}
          </Stack>

          <Divider sx={{ width: '100%', my: 5 }} />

          <Stack direction="row" spacing={2.5} sx={{ flexWrap: 'wrap', justifyContent: 'center' }}>
            {[
              { to: '/price', label: t('homePage.vendors.cta') },
              { to: '/about', label: t('homePage.cta.secondary') }
            ].map((item) => (
              <Button
                key={item.to}
                component={Link}
                to={item.to}
                size="small"
                sx={{ textTransform: 'none', fontSize: '0.82rem', color: 'text.secondary' }}
              >
                {item.label}
              </Button>
            ))}
          </Stack>
        </Stack>
      </Container>
    </Box>
  );
};

export default MinimalHome;
