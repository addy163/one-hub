import { useEffect, useMemo, useState } from 'react';
import { Box, Button, Chip, Container, Stack, Typography, useTheme } from '@mui/material';
import { useSelector } from 'react-redux';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import { API } from 'utils/api';
import {
  HubOutlined,
  SecurityOutlined,
  PaymentsOutlined,
  BoltOutlined,
  SpeedOutlined,
  LockOutlined,
  ArrowForwardOutlined,
  ContentCopyOutlined,
  CheckOutlined,
  AutoAwesomeOutlined,
  CodeOutlined
} from '@mui/icons-material';

// ==============================|| 优智AI（UUAI）品牌首页 ||============================== //
// 内置首页模板之一，由后台「首页设置」的 HomePageStyle 选择（默认 brand）。
// 模板注册见 views/Home/templates.js，新增风格无需改动本文件。

const BRAND = {
  blue: '#5B7CFA',
  purple: '#8B5CF6',
  cyan: '#22D3EE'
};

// i18nKey -> homePage.features.<key>.title / .desc in src/i18n/locales/*.json
const FEATURES = [
  { icon: HubOutlined, i18nKey: 'integration', color: '#5B7CFA' },
  { icon: SecurityOutlined, i18nKey: 'stability', color: '#22D3EE' },
  { icon: PaymentsOutlined, i18nKey: 'payg', color: '#8B5CF6' },
  { icon: SpeedOutlined, i18nKey: 'concurrency', color: '#F59E0B' },
  { icon: BoltOutlined, i18nKey: 'latency', color: '#EC4899' },
  { icon: LockOutlined, i18nKey: 'security', color: '#10B981' }
];

// 厂商展示优先级，越靠前越先显示
const VENDOR_PRIORITY = [
  'openai', 'anthropic', 'gemini', 'google', 'deepseek', 'qwen', 'zhipu', 'moonshot', 'xai',
  'mistral', 'groq', 'volc', 'baidu', 'tencent', 'hunyuan', 'siliconflow', 'minimax',
  'midjourney', 'openrouter', 'cohere', 'stability', 'suno', 'kling', 'flux', 'perplexity',
  'stepfun', 'meta', 'ollama', 'cloudflare'
];

const SectionHeading = ({ eyebrow, title, subtitle, isDark, center = true }) => (
  <Box sx={{ textAlign: center ? 'center' : 'left', maxWidth: 720, mx: center ? 'auto' : 0 }}>
    <Typography
      sx={{
        fontSize: '0.75rem',
        fontWeight: 700,
        letterSpacing: '0.18em',
        color: BRAND.blue,
        mb: 1.5,
        textTransform: 'uppercase'
      }}
    >
      {eyebrow}
    </Typography>
    <Typography
      component="h2"
      sx={{
        fontSize: { xs: '1.75rem', md: '2.4rem' },
        fontWeight: 700,
        letterSpacing: '-0.02em',
        lineHeight: 1.3,
        color: isDark ? '#F1F5F9' : '#0F172A'
      }}
    >
      {title}
    </Typography>
    <Typography sx={{ mt: 2, fontSize: { xs: '0.95rem', md: '1.02rem' }, lineHeight: 1.85, color: isDark ? 'rgba(226,232,240,.6)' : 'rgba(15,23,42,.6)' }}>
      {subtitle}
    </Typography>
  </Box>
);

const BaseIndex = () => {
  const theme = useTheme();
  const { t } = useTranslation();
  const isDark = theme.palette.mode === 'dark';
  const account = useSelector((state) => state.account);
  const siteInfo = useSelector((state) => state.siteInfo);
  const brandName = siteInfo?.system_name || 'UUAI';
  const [ownedby, setOwnedby] = useState({});
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    let mounted = true;
    API.get('/api/ownedby')
      .then((res) => {
        if (mounted && res.data && res.data.success) {
          setOwnedby(res.data.data || {});
        }
      })
      .catch(() => {});
    return () => {
      mounted = false;
    };
  }, []);

  // 从真实服务商元数据生成 Logo 墙，加载失败自动降级为文字
  const vendors = useMemo(() => {
    const list = Object.values(ownedby).filter((v) => v && v.name && v.icon);
    const rank = (v) => {
      const name = String(v.name).toLowerCase();
      const idx = VENDOR_PRIORITY.findIndex((k) => name.includes(k));
      return idx === -1 ? 999 : idx;
    };
    return list.sort((a, b) => rank(a) - rank(b) || (a.id || 0) - (b.id || 0)).slice(0, 18);
  }, [ownedby]);

  const vendorCount = Object.keys(ownedby).length || 46;
  const loggedIn = !!account?.user;
  const primaryTo = loggedIn ? '/panel' : '/login';
  const primaryLabel = loggedIn ? t('homePage.hero.ctaConsole') : t('homePage.hero.ctaStart');
  const origin = typeof window !== 'undefined' ? window.location.origin : 'https://api.uuz.ai';

  const codeSnippet = `from openai import OpenAI

client = OpenAI(
    api_key="${t('homePage.quickstart.tokenPlaceholder')}",
    base_url="${origin}/v1"
)

resp = client.chat.completions.create(
    model="gpt-5.2",
    messages=[{"role": "user", "content": "${t('homePage.quickstart.hello')}"}]
)
print(resp.choices[0].message.content)`;

  const handleCopy = () => {
    if (!navigator.clipboard || !navigator.clipboard.writeText) return;
    navigator.clipboard
      .writeText(codeSnippet)
      .then(() => {
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      })
      .catch(() => {});
  };

  const surface = isDark ? 'rgba(255,255,255,.035)' : '#FFFFFF';
  const border = isDark ? 'rgba(255,255,255,.09)' : 'rgba(15,23,42,.08)';

  const STATS = [
    { value: `${vendorCount}+`, label: t('homePage.stats.vendors') },
    { value: '100%', label: t('homePage.stats.compat') },
    { value: t('homePage.stats.free'), label: t('homePage.stats.monthlyFee') },
    { value: '7×24', label: t('homePage.stats.uptime') }
  ];

  return (
    <Box sx={{ width: '100%', overflow: 'hidden' }}>
      {/* ==================== HERO ==================== */}
      <Box
        component="section"
        sx={{
          position: 'relative',
          overflow: 'hidden',
          color: '#fff',
          background:
            'radial-gradient(1100px 620px at 12% 8%, rgba(91,124,250,.30), transparent 62%),' +
            'radial-gradient(900px 560px at 88% 18%, rgba(139,92,246,.26), transparent 62%),' +
            'radial-gradient(800px 520px at 62% 96%, rgba(34,211,238,.13), transparent 64%),' +
            'linear-gradient(160deg, #070B1A 0%, #0C1230 52%, #080E24 100%)'
        }}
      >
        {/* 网格纹理 */}
        <Box
          sx={{
            position: 'absolute',
            inset: 0,
            pointerEvents: 'none',
            backgroundImage:
              'linear-gradient(rgba(255,255,255,.045) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,.045) 1px, transparent 1px)',
            backgroundSize: '58px 58px',
            maskImage: 'linear-gradient(to bottom, rgba(0,0,0,.8), transparent 88%)',
            WebkitMaskImage: 'linear-gradient(to bottom, rgba(0,0,0,.8), transparent 88%)'
          }}
        />
        {/* 柔和光斑 */}
        <Box
          sx={{
            position: 'absolute',
            top: '-160px',
            left: '8%',
            width: 420,
            height: 420,
            borderRadius: '50%',
            background: 'radial-gradient(circle, rgba(91,124,250,.28), transparent 68%)',
            filter: 'blur(30px)',
            pointerEvents: 'none'
          }}
        />
        <Box
          sx={{
            position: 'absolute',
            bottom: '-220px',
            right: '4%',
            width: 520,
            height: 520,
            borderRadius: '50%',
            background: 'radial-gradient(circle, rgba(139,92,246,.22), transparent 68%)',
            filter: 'blur(40px)',
            pointerEvents: 'none'
          }}
        />

        <Container maxWidth="lg" sx={{ position: 'relative', zIndex: 1, pt: { xs: 8, md: 12 }, pb: { xs: 13, md: 17 } }}>
          <Box
            sx={{
              display: 'grid',
              gridTemplateColumns: { xs: '1fr', md: '1.08fr 0.92fr' },
              gap: { xs: 6, md: 7 },
              alignItems: 'center'
            }}
          >
            {/* 左侧主文案 */}
            <Box
              sx={{
                '@keyframes fadeUp': {
                  from: { opacity: 0, transform: 'translateY(20px)' },
                  to: { opacity: 1, transform: 'translateY(0)' }
                },
                animation: 'fadeUp .7s ease-out both'
              }}
            >
              <Chip
                icon={<AutoAwesomeOutlined sx={{ fontSize: 15, color: BRAND.cyan + ' !important' }} />}
                label={t('homePage.hero.badge')}
                sx={{
                  mb: 3.5,
                  height: 34,
                  px: 1.2,
                  fontSize: '0.8rem',
                  fontWeight: 500,
                  color: 'rgba(255,255,255,.88)',
                  backgroundColor: 'rgba(255,255,255,.07)',
                  border: '1px solid rgba(255,255,255,.14)',
                  backdropFilter: 'blur(10px)',
                  '& .MuiChip-label': { px: 0.5 }
                }}
              />

              <Typography
                component="h1"
                sx={{
                  fontSize: { xs: '2.05rem', sm: '3.1rem', md: '3.65rem' },
                  fontWeight: 800,
                  lineHeight: 1.22,
                  letterSpacing: '-0.035em',
                  color: '#fff'
                }}
              >
                {t('homePage.hero.title1')}
                <br />
                <Box
                  component="span"
                  sx={{
                    background: `linear-gradient(100deg, ${BRAND.cyan} 0%, ${BRAND.blue} 42%, ${BRAND.purple} 92%)`,
                    WebkitBackgroundClip: 'text',
                    WebkitTextFillColor: 'transparent',
                    backgroundClip: 'text'
                  }}
                >
                  {t('homePage.hero.title2')}
                </Box>
              </Typography>

              <Typography
                sx={{
                  mt: 3,
                  maxWidth: 560,
                  fontSize: { xs: '0.98rem', md: '1.08rem' },
                  lineHeight: 1.95,
                  color: 'rgba(226,232,240,.72)'
                }}
              >
                {t('homePage.hero.desc', { brand: brandName })}
              </Typography>

              <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} sx={{ mt: 4.5 }}>
                <Button
                  component={Link}
                  to={primaryTo}
                  variant="contained"
                  endIcon={<ArrowForwardOutlined />}
                  sx={{
                    height: 52,
                    px: 3.4,
                    borderRadius: '12px',
                    fontSize: '0.95rem',
                    fontWeight: 600,
                    textTransform: 'none',
                    color: '#fff',
                    background: `linear-gradient(100deg, ${BRAND.blue}, ${BRAND.purple})`,
                    boxShadow: '0 12px 32px rgba(91,124,250,.36)',
                    '&:hover': {
                      background: `linear-gradient(100deg, ${BRAND.blue}, ${BRAND.purple})`,
                      boxShadow: '0 16px 38px rgba(91,124,250,.48)',
                      transform: 'translateY(-2px)'
                    },
                    transition: 'all .25s ease'
                  }}
                >
                  {primaryLabel}
                </Button>
                <Button
                  component={Link}
                  to="/price"
                  variant="outlined"
                  startIcon={<CodeOutlined />}
                  sx={{
                    height: 52,
                    px: 3,
                    borderRadius: '12px',
                    fontSize: '0.95rem',
                    fontWeight: 500,
                    textTransform: 'none',
                    color: 'rgba(255,255,255,.9)',
                    borderColor: 'rgba(255,255,255,.24)',
                    backgroundColor: 'rgba(255,255,255,.04)',
                    '&:hover': {
                      borderColor: 'rgba(255,255,255,.5)',
                      backgroundColor: 'rgba(255,255,255,.09)'
                    }
                  }}
                >
                  {t('homePage.hero.ctaPricing')}
                </Button>
              </Stack>

              <Stack direction="row" spacing={3} sx={{ mt: 5, flexWrap: 'wrap', gap: 1.5 }}>
                {['tag1', 'tag2', 'tag3', 'tag4'].map((tagKey) => (
                  <Stack key={tagKey} direction="row" spacing={0.8} alignItems="center">
                    <CheckOutlined sx={{ fontSize: 15, color: BRAND.cyan }} />
                    <Typography sx={{ fontSize: '0.82rem', color: 'rgba(226,232,240,.6)' }}>{t(`homePage.hero.${tagKey}`)}</Typography>
                  </Stack>
                ))}
              </Stack>
            </Box>

            {/* 右侧网关示意卡片 */}
            <Box
              sx={{
                position: 'relative',
                display: { xs: 'none', md: 'block' },
                '@keyframes floaty': {
                  '0%, 100%': { transform: 'translateY(0)' },
                  '50%': { transform: 'translateY(-12px)' }
                },
                animation: 'floaty 7s ease-in-out infinite'
              }}
            >
              <Box
                sx={{
                  position: 'absolute',
                  inset: -40,
                  background: 'radial-gradient(circle at 50% 45%, rgba(91,124,250,.28), transparent 68%)',
                  filter: 'blur(36px)',
                  pointerEvents: 'none'
                }}
              />
              <Box
                sx={{
                  position: 'relative',
                  borderRadius: '18px',
                  border: '1px solid rgba(255,255,255,.13)',
                  background: 'linear-gradient(160deg, rgba(23,32,68,.94), rgba(10,16,40,.96))',
                  backdropFilter: 'blur(20px)',
                  boxShadow: '0 32px 80px rgba(0,0,0,.5)',
                  overflow: 'hidden'
                }}
              >
                {/* 窗口栏 */}
                <Stack
                  direction="row"
                  alignItems="center"
                  spacing={1}
                  sx={{ px: 2.2, py: 1.6, borderBottom: '1px solid rgba(255,255,255,.08)' }}
                >
                  {['#FF5F57', '#FEBC2E', '#28C840'].map((c) => (
                    <Box key={c} sx={{ width: 9, height: 9, borderRadius: '50%', bgcolor: c, opacity: 0.85 }} />
                  ))}
                  <Typography sx={{ ml: 1.2, fontSize: '0.72rem', color: 'rgba(226,232,240,.45)', fontFamily: 'monospace' }}>
                    unified-gateway
                  </Typography>
                </Stack>

                <Box sx={{ p: 2.6 }}>
                  {/* 请求行 */}
                  <Stack direction="row" alignItems="center" spacing={1.2} sx={{ mb: 2 }}>
                    <Box
                      sx={{
                        px: 0.9,
                        py: 0.25,
                        borderRadius: '5px',
                        fontSize: '0.68rem',
                        fontWeight: 700,
                        color: '#C3E88D',
                        bgcolor: 'rgba(195,232,141,.12)'
                      }}
                    >
                      POST
                    </Box>
                    <Typography sx={{ fontSize: '0.78rem', fontFamily: 'monospace', color: 'rgba(226,232,240,.78)' }}>
                      /v1/chat/completions
                    </Typography>
                    <Box sx={{ ml: 'auto !important' }}>
                      <Box
                        sx={{
                          px: 0.9,
                          py: 0.25,
                          borderRadius: '5px',
                          fontSize: '0.68rem',
                          fontWeight: 700,
                          color: '#28C840',
                          bgcolor: 'rgba(40,200,64,.12)'
                        }}
                      >
                        200 OK
                      </Box>
                    </Box>
                  </Stack>

                  {/* 请求体 */}
                  <Box
                    component="pre"
                    sx={{
                      m: 0,
                      p: 2,
                      borderRadius: '10px',
                      bgcolor: 'rgba(0,0,0,.26)',
                      border: '1px solid rgba(255,255,255,.06)',
                      fontFamily: '"SFMono-Regular", Consolas, monospace',
                      fontSize: '0.75rem',
                      lineHeight: 1.95,
                      color: '#B9C7EE',
                      whiteSpace: 'pre-wrap'
                    }}
                  >
                    {'{\n  '}
                    <Box component="span" sx={{ color: '#82AAFF' }}>{'"model"'}</Box>
                    {': '}
                    <Box component="span" sx={{ color: '#C3E88D' }}>{'"gpt-5.2"'}</Box>
                    {',\n  '}
                    <Box component="span" sx={{ color: '#82AAFF' }}>{'"messages"'}</Box>
                    {': [{ '}
                    <Box component="span" sx={{ color: '#82AAFF' }}>{'"role"'}</Box>
                    {': '}
                    <Box component="span" sx={{ color: '#C3E88D' }}>{'"user"'}</Box>
                    {' }]\n}'}
                  </Box>

                  {/* 路由分流 */}
                  <Typography sx={{ mt: 2.4, mb: 1.2, fontSize: '0.7rem', letterSpacing: '0.1em', color: 'rgba(226,232,240,.4)' }}>
                    {t('homePage.terminal.routing')}
                  </Typography>
                  <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.9 }}>
                    {[
                      { n: 'OpenAI', c: '#10A37F' },
                      { n: 'Claude', c: '#D97757' },
                      { n: 'Gemini', c: '#4285F4' },
                      { n: 'DeepSeek', c: '#4D6BFE' },
                      { n: 'Qwen', c: '#615CED' },
                      { n: 'Grok', c: '#8B5CF6' }
                    ].map((v) => (
                      <Box
                        key={v.n}
                        sx={{
                          display: 'flex',
                          alignItems: 'center',
                          gap: 0.7,
                          px: 1.1,
                          py: 0.6,
                          borderRadius: '7px',
                          fontSize: '0.72rem',
                          color: 'rgba(255,255,255,.82)',
                          bgcolor: 'rgba(255,255,255,.05)',
                          border: '1px solid rgba(255,255,255,.08)'
                        }}
                      >
                        <Box sx={{ width: 6, height: 6, borderRadius: '50%', bgcolor: v.c, boxShadow: `0 0 8px ${v.c}` }} />
                        {v.n}
                      </Box>
                    ))}
                  </Box>

                  <Stack direction="row" spacing={2} sx={{ mt: 2.4, pt: 2, borderTop: '1px solid rgba(255,255,255,.07)' }}>
                    {[
                      [t('homePage.terminal.latency'), '0.82s'],
                      ['Tokens', '1.2K'],
                      [t('homePage.terminal.billing'), t('homePage.terminal.payg')]
                    ].map(([k, v]) => (
                      <Box key={k}>
                        <Typography sx={{ fontSize: '0.68rem', color: 'rgba(226,232,240,.4)' }}>{k}</Typography>
                        <Typography sx={{ fontSize: '0.82rem', fontWeight: 600, color: 'rgba(255,255,255,.9)' }}>{v}</Typography>
                      </Box>
                    ))}
                  </Stack>
                </Box>
              </Box>

              {/* 浮动标签 */}
              <Box
                sx={{
                  position: 'absolute',
                  top: -18,
                  right: -18,
                  px: 1.6,
                  py: 0.9,
                  borderRadius: '10px',
                  fontSize: '0.74rem',
                  fontWeight: 600,
                  color: '#fff',
                  background: `linear-gradient(120deg, ${BRAND.blue}, ${BRAND.purple})`,
                  boxShadow: '0 12px 28px rgba(91,124,250,.4)',
                  transform: 'rotate(4deg)'
                }}
              >
                ✦ {t('homePage.terminal.auth')}
              </Box>
              <Box
                sx={{
                  position: 'absolute',
                  bottom: -34,
                  left: -18,
                  px: 1.6,
                  py: 0.9,
                  borderRadius: '10px',
                  fontSize: '0.74rem',
                  fontWeight: 600,
                  color: '#fff',
                  background: 'rgba(16,185,129,.92)',
                  boxShadow: '0 12px 28px rgba(16,185,129,.28)',
                  transform: 'rotate(-4deg)'
                }}
              >
                ✓ {t('homePage.terminal.failover')}
              </Box>
            </Box>
          </Box>
        </Container>
      </Box>

      {/* ==================== 指标条 ==================== */}
      <Container maxWidth="lg" sx={{ position: 'relative', zIndex: 2, mt: { xs: -7, md: -9 } }}>
        <Box
          sx={{
            display: 'grid',
            gridTemplateColumns: { xs: 'repeat(2, 1fr)', md: 'repeat(4, 1fr)' },
            borderRadius: '18px',
            overflow: 'hidden',
            bgcolor: surface,
            border: `1px solid ${border}`,
            boxShadow: isDark ? '0 20px 50px rgba(0,0,0,.35)' : '0 20px 50px rgba(15,23,42,.09)',
            backdropFilter: 'blur(16px)'
          }}
        >
          {STATS.map((s, i) => (
            <Box
              key={s.label}
              sx={{
                px: { xs: 2, md: 3 },
                py: { xs: 2.6, md: 3.2 },
                textAlign: 'center',
                borderRight: { md: i < 3 ? `1px solid ${border}` : 'none' },
                borderBottom: { xs: i < 2 ? `1px solid ${border}` : 'none', md: 'none' }
              }}
            >
              <Typography
                sx={{
                  fontSize: { xs: '1.55rem', md: '1.95rem' },
                  fontWeight: 800,
                  letterSpacing: '-0.03em',
                  background: `linear-gradient(100deg, ${BRAND.blue}, ${BRAND.purple})`,
                  WebkitBackgroundClip: 'text',
                  WebkitTextFillColor: 'transparent',
                  backgroundClip: 'text'
                }}
              >
                {s.value}
              </Typography>
              <Typography sx={{ mt: 0.6, fontSize: '0.8rem', color: isDark ? 'rgba(226,232,240,.55)' : 'rgba(15,23,42,.55)' }}>
                {s.label}
              </Typography>
            </Box>
          ))}
        </Box>
      </Container>

      {/* ==================== 厂商 Logo 墙 ==================== */}
      <Box component="section" sx={{ py: { xs: 8, md: 11 } }}>
        <Container maxWidth="lg">
          <SectionHeading
            eyebrow={t('homePage.vendors.eyebrow')}
            title={t('homePage.vendors.title', { total: vendorCount })}
            subtitle={t('homePage.vendors.subtitle')}
            isDark={isDark}
          />
          <Box sx={{ display: 'flex', flexWrap: 'wrap', justifyContent: 'center', gap: 1.4, mt: 5.5 }}>
            {(vendors.length ? vendors : [{ name: 'OpenAI' }, { name: 'Anthropic' }, { name: 'Google Gemini' }, { name: 'DeepSeek' }, { name: 'Qwen' }, { name: 'xAI' }]).map(
              (v) => (
                <Box
                  key={v.name}
                  sx={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 1,
                    px: 1.8,
                    py: 1.15,
                    minWidth: 132,
                    borderRadius: '11px',
                    bgcolor: surface,
                    border: `1px solid ${border}`,
                    boxShadow: isDark ? 'none' : '0 2px 10px rgba(15,23,42,.035)',
                    transition: 'all .22s ease',
                    '&:hover': {
                      transform: 'translateY(-3px)',
                      borderColor: 'rgba(91,124,250,.45)',
                      boxShadow: '0 10px 24px rgba(91,124,250,.14)'
                    }
                  }}
                >
                  {v.icon ? (
                    <Box
                      component="img"
                      src={v.icon}
                      alt=""
                      loading="lazy"
                      onError={(e) => {
                        e.currentTarget.style.display = 'none';
                      }}
                      sx={{ width: 20, height: 20, objectFit: 'contain', flexShrink: 0 }}
                    />
                  ) : (
                    <Box
                      sx={{
                        width: 20,
                        height: 20,
                        borderRadius: '6px',
                        display: 'grid',
                        placeItems: 'center',
                        fontSize: '0.65rem',
                        fontWeight: 700,
                        color: '#fff',
                        background: `linear-gradient(135deg, ${BRAND.blue}, ${BRAND.purple})`,
                        flexShrink: 0
                      }}
                    >
                      {String(v.name).charAt(0).toUpperCase()}
                    </Box>
                  )}
                  <Typography
                    sx={{
                      fontSize: '0.83rem',
                      fontWeight: 600,
                      color: isDark ? 'rgba(241,245,249,.88)' : 'rgba(15,23,42,.82)',
                      whiteSpace: 'nowrap'
                    }}
                  >
                    {v.name}
                  </Typography>
                </Box>
              )
            )}
          </Box>
          <Box sx={{ textAlign: 'center', mt: 4 }}>
            <Button
              component={Link}
              to="/price"
              endIcon={<ArrowForwardOutlined sx={{ fontSize: 17 }} />}
              sx={{ textTransform: 'none', fontSize: '0.88rem', fontWeight: 600 }}
            >
              {t('homePage.vendors.cta')}
            </Button>
          </Box>
        </Container>
      </Box>

      {/* ==================== 核心能力 ==================== */}
      <Box
        component="section"
        sx={{ py: { xs: 8, md: 11 }, bgcolor: isDark ? 'rgba(255,255,255,.018)' : '#F7F9FF' }}
      >
        <Container maxWidth="lg">
          <SectionHeading
            eyebrow={t('homePage.features.eyebrow')}
            title={t('homePage.features.title')}
            subtitle={t('homePage.features.subtitle')}
            isDark={isDark}
          />
          <Box
            sx={{
              display: 'grid',
              gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)', md: 'repeat(3, 1fr)' },
              gap: 2.5,
              mt: 6
            }}
          >
            {FEATURES.map((f) => {
              const Icon = f.icon;
              return (
                <Box
                  key={f.i18nKey}
                  sx={{
                    p: 3.2,
                    borderRadius: '16px',
                    bgcolor: surface,
                    border: `1px solid ${border}`,
                    boxShadow: isDark ? 'none' : '0 4px 18px rgba(15,23,42,.035)',
                    transition: 'all .25s ease',
                    '&:hover': {
                      transform: 'translateY(-5px)',
                      borderColor: `${f.color}66`,
                      boxShadow: `0 18px 38px ${f.color}1F`
                    }
                  }}
                >
                  <Box
                    sx={{
                      width: 46,
                      height: 46,
                      borderRadius: '12px',
                      display: 'grid',
                      placeItems: 'center',
                      mb: 2.4,
                      color: '#fff',
                      background: `linear-gradient(135deg, ${f.color}, ${f.color}AA)`,
                      boxShadow: `0 8px 20px ${f.color}38`
                    }}
                  >
                    <Icon sx={{ fontSize: 23 }} />
                  </Box>
                  <Typography sx={{ fontSize: '1.08rem', fontWeight: 700, mb: 1.2, color: isDark ? '#F1F5F9' : '#0F172A' }}>
                    {t(`homePage.features.${f.i18nKey}.title`)}
                  </Typography>
                  <Typography sx={{ fontSize: '0.86rem', lineHeight: 1.85, color: isDark ? 'rgba(226,232,240,.58)' : 'rgba(15,23,42,.6)' }}>
                    {t(`homePage.features.${f.i18nKey}.desc`)}
                  </Typography>
                </Box>
              );
            })}
          </Box>
        </Container>
      </Box>

      {/* ==================== 接入示例 ==================== */}
      <Box component="section" sx={{ py: { xs: 8, md: 11 } }}>
        <Container maxWidth="lg">
          <Box
            sx={{
              display: 'grid',
              gridTemplateColumns: { xs: '1fr', md: '0.9fr 1.1fr' },
              gap: { xs: 5, md: 7 },
              alignItems: 'center'
            }}
          >
            <Box>
              <SectionHeading
                eyebrow={t('homePage.quickstart.eyebrow')}
                title={t('homePage.quickstart.title')}
                subtitle={t('homePage.quickstart.subtitle')}
                isDark={isDark}
                center={false}
              />
              <Stack spacing={1.8} sx={{ mt: 4 }}>
                {['point1', 'point2', 'point3', 'point4'].map((pointKey) => (
                  <Stack key={pointKey} direction="row" spacing={1.3} alignItems="flex-start">
                    <Box
                      sx={{
                        mt: '3px',
                        width: 18,
                        height: 18,
                        borderRadius: '50%',
                        display: 'grid',
                        placeItems: 'center',
                        bgcolor: 'rgba(91,124,250,.12)',
                        flexShrink: 0
                      }}
                    >
                      <CheckOutlined sx={{ fontSize: 12, color: BRAND.blue }} />
                    </Box>
                    <Typography sx={{ fontSize: '0.9rem', lineHeight: 1.7, color: isDark ? 'rgba(226,232,240,.7)' : 'rgba(15,23,42,.7)' }}>
                      {t(`homePage.quickstart.${pointKey}`)}
                    </Typography>
                  </Stack>
                ))}
              </Stack>
              <Button
                component={Link}
                to="/price"
                variant="outlined"
                endIcon={<ArrowForwardOutlined sx={{ fontSize: 17 }} />}
                sx={{ mt: 4, px: 2.6, height: 44, borderRadius: '10px', textTransform: 'none', fontWeight: 600, fontSize: '0.88rem' }}
              >
                {t('homePage.quickstart.cta')}
              </Button>
            </Box>

            {/* 代码卡片 */}
            <Box
              sx={{
                borderRadius: '16px',
                overflow: 'hidden',
                border: '1px solid rgba(255,255,255,.1)',
                background: 'linear-gradient(160deg, #101838, #0A1028)',
                boxShadow: '0 26px 60px rgba(8,14,36,.28)'
              }}
            >
              <Stack
                direction="row"
                alignItems="center"
                sx={{ px: 2.2, py: 1.5, borderBottom: '1px solid rgba(255,255,255,.08)' }}
              >
                <Typography sx={{ fontSize: '0.76rem', fontFamily: 'monospace', color: 'rgba(226,232,240,.65)' }}>
                  quickstart.py
                </Typography>
                <Button
                  onClick={handleCopy}
                  size="small"
                  startIcon={copied ? <CheckOutlined sx={{ fontSize: 15 }} /> : <ContentCopyOutlined sx={{ fontSize: 15 }} />}
                  sx={{
                    ml: 'auto',
                    px: 1.2,
                    py: 0.4,
                    minWidth: 0,
                    fontSize: '0.74rem',
                    textTransform: 'none',
                    color: copied ? '#28C840' : 'rgba(226,232,240,.6)',
                    '&:hover': { bgcolor: 'rgba(255,255,255,.07)' }
                  }}
                >
                  {copied ? t('homePage.quickstart.copied') : t('homePage.quickstart.copy')}
                </Button>
              </Stack>
              <Box
                component="pre"
                sx={{
                  m: 0,
                  p: { xs: 2.2, md: 3 },
                  fontFamily: '"SFMono-Regular", Consolas, "Liberation Mono", monospace',
                  fontSize: { xs: '0.72rem', md: '0.79rem' },
                  lineHeight: 2.05,
                  color: '#C7D3F5',
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-all',
                  overflowX: 'auto'
                }}
              >
                <Box component="span" sx={{ color: '#5C6B8A' }}>{t('homePage.quickstart.codeComment')}</Box>
                <Box component="span" sx={{ color: '#C792EA' }}>from</Box>
                {' openai '}
                <Box component="span" sx={{ color: '#C792EA' }}>import</Box>
                {' OpenAI\n\n'}
                {'client = OpenAI(\n'}
                {'    api_key='}
                <Box component="span" sx={{ color: '#C3E88D' }}>{`"${t('homePage.quickstart.tokenPlaceholder')}"`}</Box>
                {',\n'}
                {'    base_url='}
                <Box component="span" sx={{ color: '#C3E88D' }}>{`"${origin}/v1"`}</Box>
                {'\n)\n\n'}
                {'resp = client.chat.completions.create(\n'}
                {'    model='}
                <Box component="span" sx={{ color: '#C3E88D' }}>{'"gpt-5.2"'}</Box>
                {',\n'}
                {'    messages=[{'}
                <Box component="span" sx={{ color: '#82AAFF' }}>{'"role"'}</Box>
                {': '}
                <Box component="span" sx={{ color: '#C3E88D' }}>{'"user"'}</Box>
                {', '}
                <Box component="span" sx={{ color: '#82AAFF' }}>{'"content"'}</Box>
                {': '}
                <Box component="span" sx={{ color: '#C3E88D' }}>{`"${t('homePage.quickstart.hello')}"`}</Box>
                {'}]\n)\n'}
                <Box component="span" sx={{ color: '#82AAFF' }}>print</Box>
                {'(resp.choices[0].message.content)'}
              </Box>
            </Box>
          </Box>
        </Container>
      </Box>

      {/* ==================== 底部 CTA ==================== */}
      <Box component="section" sx={{ pb: { xs: 9, md: 12 }, px: 2 }}>
        <Container maxWidth="lg">
          <Box
            sx={{
              position: 'relative',
              overflow: 'hidden',
              borderRadius: '24px',
              px: { xs: 3, md: 8 },
              py: { xs: 6, md: 8 },
              textAlign: 'center',
              color: '#fff',
              background:
                'radial-gradient(700px 320px at 15% 0%, rgba(34,211,238,.22), transparent 65%),' +
                'radial-gradient(700px 320px at 85% 100%, rgba(139,92,246,.3), transparent 65%),' +
                'linear-gradient(110deg, #3D5BE0, #6D4AE8 60%, #8B5CF6)'
            }}
          >
            <Box
              sx={{
                position: 'absolute',
                inset: 0,
                pointerEvents: 'none',
                backgroundImage:
                  'linear-gradient(rgba(255,255,255,.07) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,.07) 1px, transparent 1px)',
                backgroundSize: '44px 44px',
                maskImage: 'radial-gradient(circle at 50% 50%, rgba(0,0,0,.7), transparent 75%)',
                WebkitMaskImage: 'radial-gradient(circle at 50% 50%, rgba(0,0,0,.7), transparent 75%)'
              }}
            />
            <Box sx={{ position: 'relative', zIndex: 1 }}>
              <Typography sx={{ fontSize: { xs: '1.75rem', md: '2.35rem' }, fontWeight: 800, letterSpacing: '-0.025em' }}>
                {t('homePage.cta.title')}
              </Typography>
              <Typography sx={{ mt: 2, fontSize: { xs: '0.95rem', md: '1.04rem' }, color: 'rgba(255,255,255,.82)', lineHeight: 1.8 }}>
                {t('homePage.cta.desc')}
              </Typography>
              <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} justifyContent="center" sx={{ mt: 4.5 }}>
                <Button
                  component={Link}
                  to={primaryTo}
                  variant="contained"
                  endIcon={<ArrowForwardOutlined />}
                  sx={{
                    height: 50,
                    px: 3.4,
                    borderRadius: '12px',
                    textTransform: 'none',
                    fontSize: '0.93rem',
                    fontWeight: 700,
                    color: BRAND.blue,
                    bgcolor: '#fff',
                    boxShadow: '0 12px 30px rgba(0,0,0,.18)',
                    '&:hover': { bgcolor: '#F1F5FF', transform: 'translateY(-2px)' },
                    transition: 'all .25s ease'
                  }}
                >
                  {primaryLabel}
                </Button>
                <Button
                  component={Link}
                  to="/about"
                  variant="outlined"
                  sx={{
                    height: 50,
                    px: 3,
                    borderRadius: '12px',
                    textTransform: 'none',
                    fontSize: '0.93rem',
                    fontWeight: 600,
                    color: '#fff',
                    borderColor: 'rgba(255,255,255,.45)',
                    '&:hover': { borderColor: '#fff', bgcolor: 'rgba(255,255,255,.1)' }
                  }}
                >
                  {t('homePage.cta.secondary')}
                </Button>
              </Stack>
            </Box>
          </Box>
        </Container>
      </Box>
    </Box>
  );
};

export default BaseIndex;
