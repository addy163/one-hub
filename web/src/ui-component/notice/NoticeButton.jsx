import { Avatar, Badge, Box, ButtonBase } from '@mui/material';
import { useTheme } from '@mui/material/styles';

import { Icon } from '@iconify/react';

import { useNotice } from './NoticeContext';

export function NoticeButton() {
  const theme = useTheme();
  const { openInbox, unreadCount } = useNotice();

  return (
    <Box
      sx={{
        ml: 2,
        mr: 3,
        [theme.breakpoints.down('md')]: {
          mr: 2
        }
      }}
    >
      <ButtonBase sx={{ borderRadius: '12px' }} onClick={openInbox} aria-label="公告">
        <Badge
          badgeContent={unreadCount}
          color="error"
          max={99}
          overlap="circular"
          sx={{ '& .MuiBadge-badge': { fontSize: '0.625rem', height: 16, minWidth: 16, px: 0.5 } }}
        >
          <Avatar
            variant="rounded"
            sx={{
              ...theme.typography.commonAvatar,
              ...theme.typography.mediumAvatar,
              ...theme.typography.menuButton,
              transition: 'all .2s ease-in-out',
              borderColor: 'transparent',
              backgroundColor: 'transparent',
              boxShadow: 'none',
              borderRadius: '50%',
              '&:hover': {
                boxShadow: '0 0 10px rgba(0,0,0,0.2)',
                backgroundColor: 'transparent',
                borderRadius: '50%'
              }
            }}
            color="inherit"
          >
            <Icon icon={unreadCount > 0 ? 'solar:bell-bing-bold-duotone' : 'solar:bell-linear'} width="1.6rem" />
          </Avatar>
        </Badge>
      </ButtonBase>
    </Box>
  );
}

export default NoticeButton;
