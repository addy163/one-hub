import { Box, Button, Dialog, DialogContent, IconButton, Typography } from '@mui/material';
import { useTheme } from '@mui/material/styles';
import { Icon } from '@iconify/react';

import { useNotice } from './NoticeContext';
import NoticeContent, { NoticeMeta, levelColors, levelMeta } from './NoticeContent';

/**
 * NoticeDialogs 首次进入时弹出的公告。
 * 多条未读公告会排成队列，逐条展示；标记「必须确认」的公告不可关闭，
 * 必须点确认按钮才能继续。
 */
export const NoticeDialogs = () => {
  const theme = useTheme();
  const { current, queueTotal, queueIndex, closePopup, dismissQueue, openInbox } = useNotice();

  if (!current) return null;

  const meta = levelMeta(current.level);
  const colors = levelColors(theme, current.level);
  const hasNext = queueIndex + 1 < queueTotal;
  const locked = current.require_ack;

  const handleViewAll = async () => {
    await dismissQueue();
    openInbox();
  };

  return (
    <Dialog
      open
      fullWidth
      maxWidth="sm"
      disableEscapeKeyDown={locked}
      onClose={locked ? undefined : closePopup}
      PaperProps={{ sx: { borderRadius: '14px', overflow: 'hidden' } }}
    >
      <Box sx={{ height: 4, bgcolor: colors.main }} />

      <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 1.5, px: 3, pt: 2.5, pb: 1.5 }}>
        <Box
          sx={{
            width: 34,
            height: 34,
            flexShrink: 0,
            borderRadius: '10px',
            display: 'grid',
            placeItems: 'center',
            bgcolor: colors.bg,
            color: colors.fg
          }}
        >
          <Icon icon={meta.icon} width={19} height={19} />
        </Box>
        <Box sx={{ minWidth: 0, flex: 1 }}>
          <Typography variant="h4" sx={{ fontWeight: 600, lineHeight: 1.4, mb: 0.75 }}>
            {current.title}
          </Typography>
          <NoticeMeta notice={current} />
        </Box>
        {!locked && (
          <IconButton size="small" onClick={closePopup} sx={{ mt: -0.5, mr: -0.5, color: 'text.disabled' }}>
            <Icon icon="solar:close-circle-bold" width={20} height={20} />
          </IconButton>
        )}
      </Box>

      <DialogContent sx={{ px: 3, py: 0 }}>
        <Box sx={{ maxHeight: '52vh', overflowY: 'auto', pr: 0.5 }}>
          <NoticeContent notice={current} />
        </Box>
      </DialogContent>

      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, px: 3, py: 2 }}>
        <Typography variant="caption" sx={{ color: 'text.disabled', flex: 1 }}>
          {queueTotal > 1 ? `${queueIndex + 1} / ${queueTotal}` : ''}
        </Typography>
        {queueTotal > 1 && (
          <Button size="small" onClick={handleViewAll} sx={{ color: 'text.secondary' }}>
            查看全部
          </Button>
        )}
        <Button variant="contained" disableElevation onClick={closePopup} sx={{ borderRadius: '8px', px: 2.5 }}>
          {hasNext ? '下一条' : locked ? '我已阅读并确认' : '我知道了'}
        </Button>
      </Box>
    </Dialog>
  );
};

export default NoticeDialogs;
