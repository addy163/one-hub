import PropTypes from 'prop-types';
import { Box, Stack, Typography } from '@mui/material';
import { alpha, useTheme } from '@mui/material/styles';
import { Icon } from '@iconify/react';
import dayjs from 'dayjs';

import { useNotice } from './NoticeContext';

// 公告级别的展示元数据：决定图标、配色与中文标签。
export const NOTICE_LEVELS = {
  info: { icon: 'solar:info-circle-bold', label: '通知' },
  success: { icon: 'solar:check-circle-bold', label: '上新' },
  warning: { icon: 'solar:danger-triangle-bold', label: '重要' },
  error: { icon: 'solar:danger-circle-bold', label: '紧急' }
};

export function levelMeta(level) {
  return NOTICE_LEVELS[level] || NOTICE_LEVELS.info;
}

// 取主题里对应级别的色板；level 没有对应色时回退到 info。
export function levelPalette(theme, level) {
  const key = NOTICE_LEVELS[level] ? level : 'info';
  return theme.palette[key] || theme.palette.info;
}

// 级别配色：底色用主色低透明度叠加，前景色按明暗模式取深浅两档，
// 保证浅色/深色主题下对比度都足够。
export function levelColors(theme, level) {
  const palette = levelPalette(theme, level);
  const isDark = theme.palette.mode === 'dark';
  return {
    main: palette.main,
    fg: isDark ? palette.light : palette.dark,
    bg: alpha(palette.main, isDark ? 0.2 : 0.1)
  };
}

export function formatNoticeTime(ts) {
  if (!ts) return '';
  return dayjs(ts * 1000).format('YYYY-MM-DD HH:mm');
}

// Markdown 正文的排版：标题、列表、引用、代码块、表格、链接都收进统一节奏。
export const markdownSx = {
  fontSize: '0.875rem',
  lineHeight: 1.75,
  color: 'text.secondary',
  wordBreak: 'break-word',
  '& > :first-of-type': { mt: 0 },
  '& > :last-child': { mb: 0 },
  '& h1, & h2, & h3, & h4, & h5, & h6': {
    color: 'text.primary',
    fontWeight: 600,
    lineHeight: 1.4,
    mt: 2.5,
    mb: 1
  },
  '& h1': { fontSize: '1.25rem' },
  '& h2': { fontSize: '1.125rem' },
  '& h3': { fontSize: '1rem' },
  '& h4, & h5, & h6': { fontSize: '0.9375rem' },
  '& p': { my: 1 },
  '& ul, & ol': { pl: 3, my: 1 },
  '& li': { my: 0.5 },
  '& a': { color: 'primary.main', textDecoration: 'none', '&:hover': { textDecoration: 'underline' } },
  '& strong': { color: 'text.primary', fontWeight: 600 },
  '& blockquote': {
    m: 0,
    my: 1.5,
    pl: 2,
    borderLeft: '3px solid',
    borderColor: 'divider',
    color: 'text.secondary'
  },
  '& code': {
    px: 0.75,
    py: 0.25,
    borderRadius: '4px',
    fontSize: '0.8125rem',
    fontFamily: 'var(--font-mono, monospace)',
    bgcolor: 'action.hover'
  },
  '& pre': {
    p: 1.5,
    my: 1.5,
    borderRadius: '8px',
    overflowX: 'auto',
    bgcolor: 'action.hover',
    '& code': { p: 0, bgcolor: 'transparent' }
  },
  '& table': { width: '100%', borderCollapse: 'collapse', my: 1.5, fontSize: '0.8125rem' },
  '& th, & td': { border: '1px solid', borderColor: 'divider', px: 1.25, py: 0.75, textAlign: 'left' },
  '& th': { color: 'text.primary', fontWeight: 600, bgcolor: 'action.hover' },
  '& img': { maxWidth: '100%', borderRadius: '8px' },
  '& hr': { border: 0, borderTop: '1px solid', borderColor: 'divider', my: 2 }
};

/**
 * NoticeContent 渲染公告正文（Markdown → 消毒后的 HTML）。
 */
export default function NoticeContent({ notice, sx }) {
  const { renderMarkdown } = useNotice();
  if (!notice) return null;

  return <Box sx={{ ...markdownSx, ...sx }} dangerouslySetInnerHTML={{ __html: renderMarkdown(notice.content) }} />;
}

NoticeContent.propTypes = {
  notice: PropTypes.object,
  sx: PropTypes.object
};

/**
 * NoticeMeta 公告的级别标签 + 发布时间，用于列表与弹窗头部。
 */
export function NoticeMeta({ notice, showTime = true }) {
  const theme = useTheme();
  const meta = levelMeta(notice?.level);
  const colors = levelColors(theme, notice?.level);

  if (!notice) return null;

  return (
    <Stack direction="row" spacing={1} alignItems="center" flexWrap="wrap" useFlexGap>
      <Stack
        direction="row"
        spacing={0.5}
        alignItems="center"
        sx={{
          px: 0.875,
          py: 0.25,
          borderRadius: '6px',
          bgcolor: colors.bg,
          color: colors.fg,
          fontSize: '0.75rem',
          fontWeight: 600
        }}
      >
        <Icon icon={meta.icon} width={13} height={13} />
        <span>{meta.label}</span>
      </Stack>
      {notice.require_ack && (
        <Typography variant="caption" sx={{ color: 'text.secondary' }}>
          需确认
        </Typography>
      )}
      {showTime && notice.created_at > 0 && (
        <Typography variant="caption" sx={{ color: 'text.disabled' }}>
          {formatNoticeTime(notice.created_at)}
        </Typography>
      )}
    </Stack>
  );
}

NoticeMeta.propTypes = {
  notice: PropTypes.object,
  showTime: PropTypes.bool
};
