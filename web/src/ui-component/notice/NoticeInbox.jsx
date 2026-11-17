import PropTypes from 'prop-types';
import { Box, Button, Chip, Divider, Drawer, IconButton, Stack, Typography } from '@mui/material';
import { useTheme } from '@mui/material/styles';
import { Icon } from '@iconify/react';
import dayjs from 'dayjs';

import { useNotice } from './NoticeContext';
import NoticeContent, { NoticeMeta, levelColors, levelMeta } from './NoticeContent';

// 列表里显示的正文摘要：去掉 Markdown 记号，取前 70 字。
function summaryOf(content, limit = 70) {
  const plain = (content || '')
    .replace(/```[\s\S]*?```/g, ' ')
    .replace(/!\[[^\]]*\]\([^)]*\)/g, ' ')
    .replace(/\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replace(/[#>*`_~-]/g, '')
    .replace(/\s+/g, ' ')
    .trim();
  return plain.length > limit ? `${plain.slice(0, limit)}…` : plain;
}

function NoticeItem({ notice, unread, onClick }) {
  const theme = useTheme();
  const meta = levelMeta(notice.level);
  const colors = levelColors(theme, notice.level);

  return (
    <Box
      onClick={onClick}
      sx={{
        display: 'flex',
        gap: 1.5,
        px: 2,
        py: 1.75,
        cursor: 'pointer',
        borderRadius: '10px',
        transition: 'background-color .15s ease',
        '&:hover': { bgcolor: 'action.hover' }
      }}
    >
      <Box
        sx={{
          width: 30,
          height: 30,
          mt: 0.25,
          flexShrink: 0,
          borderRadius: '9px',
          display: 'grid',
          placeItems: 'center',
          bgcolor: colors.bg,
          color: colors.fg
        }}
      >
        <Icon icon={meta.icon} width={16} height={16} />
      </Box>
      <Box sx={{ minWidth: 0, flex: 1 }}>
        <Stack direction="row" spacing={0.75} alignItems="center">
          {unread && <Box sx={{ width: 6, height: 6, borderRadius: '50%', bgcolor: 'primary.main', flexShrink: 0 }} />}
          <Typography
            variant="subtitle2"
            noWrap
            sx={{ fontWeight: unread ? 600 : 500, color: unread ? 'text.primary' : 'text.secondary', flex: 1 }}
          >
            {notice.title}
          </Typography>
          <Typography variant="caption" sx={{ color: 'text.disabled', flexShrink: 0 }}>
            {dayjs(notice.created_at * 1000).format('MM-DD')}
          </Typography>
        </Stack>
        <Typography
          variant="caption"
          sx={{
            display: '-webkit-box',
            WebkitLineClamp: 2,
            WebkitBoxOrient: 'vertical',
            overflow: 'hidden',
            color: 'text.disabled',
            lineHeight: 1.5,
            mt: 0.5
          }}
        >
          {summaryOf(notice.content) || meta.label}
        </Typography>
        {notice.require_ack && !notice.acked && (
          <Chip
            size="small"
            label="需确认"
            sx={{ mt: 0.75, height: 18, fontSize: '0.6875rem', bgcolor: colors.bg, color: colors.fg }}
          />
        )}
      </Box>
    </Box>
  );
}

NoticeItem.propTypes = {
  notice: PropTypes.object,
  unread: PropTypes.bool,
  onClick: PropTypes.func
};

/**
 * NoticeInbox 铃铛打开的公告收件箱。列表与详情在同一抽屉内切换，
 * 不打断用户当前页面。
 */
export default function NoticeInbox() {
  const { inboxOpen, closeInbox, notices, isRead, detailId, setDetailId, markRead } = useNotice();

  const detail = notices.find((n) => n.id === detailId) || null;
  const unreadCount = notices.filter((n) => !isRead(n)).length;

  const openDetail = (notice) => {
    setDetailId(notice.id);
    if (!isRead(notice)) {
      markRead(notice.id, false);
    }
  };

  return (
    <Drawer
      anchor="right"
      open={inboxOpen}
      onClose={closeInbox}
      PaperProps={{ sx: { width: { xs: '100%', sm: 420 }, borderRadius: { sm: '14px 0 0 14px' } } }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, px: 2.5, height: 60, flexShrink: 0 }}>
        {detail ? (
          <IconButton size="small" onClick={() => setDetailId(null)} sx={{ ml: -0.75 }}>
            <Icon icon="solar:alt-arrow-left-bold" width={18} height={18} />
          </IconButton>
        ) : (
          <Icon icon="solar:bell-bing-bold-duotone" width={20} height={20} style={{ color: 'inherit', opacity: 0.7 }} />
        )}
        <Typography variant="h4" sx={{ fontWeight: 600, flex: 1 }}>
          {detail ? '公告详情' : '公告'}
        </Typography>
        {!detail && unreadCount > 0 && (
          <Chip size="small" label={`${unreadCount} 条未读`} color="primary" sx={{ height: 22, fontSize: '0.6875rem' }} />
        )}
        <IconButton size="small" onClick={closeInbox}>
          <Icon icon="solar:close-circle-line-duotone" width={20} height={20} />
        </IconButton>
      </Box>
      <Divider />

      {detail ? (
        <Box sx={{ flex: 1, overflowY: 'auto', px: 2.5, py: 2.5, display: 'flex', flexDirection: 'column' }}>
          <Typography variant="h3" sx={{ fontWeight: 600, lineHeight: 1.4, mb: 1 }}>
            {detail.title}
          </Typography>
          <NoticeMeta notice={detail} />
          <Divider sx={{ my: 2 }} />
          <NoticeContent notice={detail} />
          {detail.require_ack && !detail.acked && (
            <Button
              variant="contained"
              disableElevation
              onClick={() => markRead(detail.id, true)}
              sx={{ mt: 3, borderRadius: '8px' }}
            >
              我已阅读并确认
            </Button>
          )}
          {detail.require_ack && detail.acked && (
            <Stack direction="row" spacing={0.75} alignItems="center" sx={{ mt: 3, color: 'text.disabled' }}>
              <Icon icon="solar:check-circle-bold" width={16} height={16} />
              <Typography variant="caption">已确认</Typography>
            </Stack>
          )}
        </Box>
      ) : (
        <Box sx={{ flex: 1, overflowY: 'auto', px: 0.5, py: 1 }}>
          {notices.length === 0 ? (
            <Stack alignItems="center" spacing={1.5} sx={{ py: 12, color: 'text.disabled' }}>
              <Icon icon="solar:bell-off-bold-duotone" width={44} height={44} />
              <Typography variant="body2">暂无公告</Typography>
            </Stack>
          ) : (
            notices.map((notice) => (
              <NoticeItem key={notice.id} notice={notice} unread={!isRead(notice)} onClick={() => openDetail(notice)} />
            ))
          )}
          {notices.length > 0 && (
            <Typography variant="caption" sx={{ display: 'block', textAlign: 'center', color: 'text.disabled', py: 2 }}>
              仅展示当前生效的公告
            </Typography>
          )}
        </Box>
      )}
    </Drawer>
  );
}
