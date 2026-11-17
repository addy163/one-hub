import PropTypes from 'prop-types';
import { Box, Button, IconButton, Typography } from '@mui/material';
import { alpha, useTheme } from '@mui/material/styles';
import { Icon } from '@iconify/react';

import { useNotice } from './NoticeContext';
import { levelColors, levelMeta } from './NoticeContent';

/**
 * NoticeBanner 勾选了「顶部横幅」的公告。
 * 同时有多条时只展示第一条（置顶优先），其余进收件箱，避免页面被横幅堆满。
 */
export default function NoticeBanner({ sx }) {
  const theme = useTheme();
  const { banners, dismissBanner, openInbox, setDetailId } = useNotice();

  const notice = banners[0];
  if (!notice) return null;

  const meta = levelMeta(notice.level);
  const colors = levelColors(theme, notice.level);
  const more = banners.length - 1;

  const handleView = () => {
    openInbox();
    setDetailId(notice.id);
  };

  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1.25,
        px: 1.75,
        py: 1,
        mb: 2,
        borderRadius: '10px',
        bgcolor: colors.bg,
        border: '1px solid',
        borderColor: alpha(colors.main, theme.palette.mode === 'dark' ? 0.4 : 0.24),
        ...sx
      }}
    >
      <Icon icon={meta.icon} width={17} height={17} style={{ color: colors.main, flexShrink: 0 }} />
      <Typography
        variant="body2"
        noWrap
        sx={{ flex: 1, minWidth: 0, fontSize: '0.8125rem', color: 'text.primary' }}
      >
        <Box component="span" sx={{ fontWeight: 600, mr: 1 }}>
          {notice.title}
        </Box>
        {more > 0 && (
          <Box component="span" sx={{ color: 'text.secondary' }}>
            等 {more + 1} 条公告
          </Box>
        )}
      </Typography>
      <Button size="small" onClick={handleView} sx={{ minWidth: 0, px: 1, fontSize: '0.8125rem', color: colors.main }}>
        查看
      </Button>
      <IconButton size="small" onClick={() => dismissBanner(notice.id)} sx={{ color: 'text.disabled', ml: -0.5 }}>
        <Icon icon="solar:close-circle-line-duotone" width={17} height={17} />
      </IconButton>
    </Box>
  );
}

NoticeBanner.propTypes = {
  sx: PropTypes.object
};
